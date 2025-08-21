package genscaffold

import (
	"embed"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/alexisvisco/goframe/cli/generators"
	"github.com/alexisvisco/goframe/cli/generators/gendb"
	"github.com/alexisvisco/goframe/cli/generators/genhelper"
	"github.com/alexisvisco/goframe/cli/generators/genhttp"
	"github.com/alexisvisco/goframe/cli/generators/genservice"
	"github.com/alexisvisco/goframe/core/helpers/str"
	"github.com/alexisvisco/goframe/core/helpers/typeutil"
	"github.com/gertd/go-pluralize"
)

type ScaffoldGenerator struct {
	Gen *generators.Generator

	// Sub-generators
	ServiceGen *genservice.ServiceGenerator
	HTTPGen    *genhttp.HTTPGenerator
	DBGen      *gendb.DatabaseGenerator

	// Pluralize client
	pluralize *pluralize.Client

	// Flags
	NoService   bool
	NoHandler   bool
	NoMigration bool
	CRUD        bool
}

type Field struct {
	Name       string
	Type       string
	GoType     string
	SqlType    string
	NamePascal string
	NameSnake  string
}

//go:embed templates
var fs embed.FS

// Type mappings
var cliToGoType = map[string]string{
	"string":     "string",
	"text":       "string",
	"integer":    "int",
	"int":        "int",
	"bigint":     "int64",
	"float":      "float64",
	"decimal":    "decimal.Decimal",
	"boolean":    "bool",
	"binary":     "[]byte",
	"date":       "time.Time",
	"time":       "time.Time",
	"datetime":   "time.Time",
	"timestamp":  "time.Time",
	"timestampz": "time.Time",
}

var cliToSqlType = map[string]string{
	"string":     "VARCHAR(255)",
	"text":       "TEXT",
	"integer":    "INTEGER",
	"int":        "INTEGER",
	"bigint":     "BIGINT",
	"float":      "FLOAT",
	"decimal":    "DECIMAL(10,2)",
	"boolean":    "BOOLEAN",
	"binary":     "BYTEA",
	"date":       "DATE",
	"time":       "TIME",
	"datetime":   "TIMESTAMP",
	"timestamp":  "TIMESTAMP",
	"timestampz": "TIMESTAMPTZ",
}

// NewScaffoldGenerator creates a new scaffold generator with sub-generators
func NewScaffoldGenerator(gen *generators.Generator) *ScaffoldGenerator {
	return &ScaffoldGenerator{
		Gen:        gen,
		ServiceGen: &genservice.ServiceGenerator{Gen: gen},
		HTTPGen:    &genhttp.HTTPGenerator{Gen: gen},
		DBGen:      &gendb.DatabaseGenerator{Gen: gen},
		pluralize:  pluralize.NewClient(),
	}
}

// GenerateScaffold generates a complete scaffold for a model
func (s *ScaffoldGenerator) GenerateScaffold(name string, fieldSpecs []string) error {
	// Parse fields
	fields, err := s.parseFields(fieldSpecs)
	if err != nil {
		return fmt.Errorf("failed to parse fields: %w", err)
	}

	// Generate model type and service interface in one file
	if err := s.generateModelAndServiceInterface(name, fields); err != nil {
		return fmt.Errorf("failed to generate model and service interface: %w", err)
	}

	// Generate service implementation
	if !s.NoService {
		if err := s.generateServiceImplementation(name, fields); err != nil {
			return fmt.Errorf("failed to generate service implementation: %w", err)
		}

		// Update service registry
		if err := s.ServiceGen.Update(); err != nil {
			return fmt.Errorf("failed to update service registry: %w", err)
		}
	}

	// Generate handler
	if !s.NoHandler {
		var services []string
		if !s.NoService {
			services = append(services, name)
		}
		if err := s.HTTPGen.GenerateHandler(name, services); err != nil {
			return fmt.Errorf("failed to generate handler: %w", err)
		}
	}

	// Generate CRUD routes if requested
	if s.CRUD {
		if err := s.GenerateCRUDRoutes(name, fields); err != nil {
			return fmt.Errorf("failed to generate CRUD routes: %w", err)
		}
	}

	// Generate migration
	if !s.NoMigration {
		if err := s.generateMigration(name, fields); err != nil {
			return fmt.Errorf("failed to generate migration: %w", err)
		}
	}

	return nil
}

func (s *ScaffoldGenerator) parseFields(fieldSpecs []string) ([]Field, error) {
	var fields []Field

	// Add default ID field if not specified
	hasID := false
	for _, spec := range fieldSpecs {
		parts := strings.Split(spec, ":")
		if len(parts) > 0 && parts[0] == "id" {
			hasID = true
			break
		}
	}

	if !hasID {
		fields = append(fields, Field{
			Name:       "id",
			Type:       "string",
			GoType:     "string",
			SqlType:    "VARCHAR(255) PRIMARY KEY",
			NamePascal: "ID",
			NameSnake:  "id",
		})
	}

	for _, spec := range fieldSpecs {
		parts := strings.Split(spec, ":")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid field specification: %s (expected format: name:type)", spec)
		}

		name := parts[0]
		fieldType := parts[1]

		// Handle special case for id:int (auto-increment)
		if name == "id" && fieldType == "int" {
			fields = append(fields, Field{
				Name:       name,
				Type:       fieldType,
				GoType:     "int",
				SqlType:    "SERIAL PRIMARY KEY",
				NamePascal: "ID",
				NameSnake:  "id",
			})
			continue
		}

		goType, ok := cliToGoType[fieldType]
		if !ok {
			return nil, fmt.Errorf("unsupported field type: %s", fieldType)
		}

		sqlType, ok := cliToSqlType[fieldType]
		if !ok {
			return nil, fmt.Errorf("unsupported field type: %s", fieldType)
		}

		// Handle primary key for id field
		if name == "id" {
			sqlType += " PRIMARY KEY"
		}

		fields = append(fields, Field{
			Name:       name,
			Type:       fieldType,
			GoType:     goType,
			SqlType:    sqlType,
			NamePascal: str.ToPascalCase(name),
			NameSnake:  str.ToSnakeCase(name),
		})
	}

	// Add default timestamps
	fields = append(fields, Field{
		Name:       "created_at",
		Type:       "timestamp",
		GoType:     "time.Time",
		SqlType:    "TIMESTAMP DEFAULT CURRENT_TIMESTAMP",
		NamePascal: "CreatedAt",
		NameSnake:  "created_at",
	})

	fields = append(fields, Field{
		Name:       "updated_at",
		Type:       "timestamp",
		GoType:     "time.Time",
		SqlType:    "TIMESTAMP DEFAULT CURRENT_TIMESTAMP",
		NamePascal: "UpdatedAt",
		NameSnake:  "updated_at",
	})

	return fields, nil
}

func (s *ScaffoldGenerator) generateModelAndServiceInterface(name string, fields []Field) error {
	path := filepath.Join("internal/types", fmt.Sprintf("%s.go", str.ToSnakeCase(name)))

	// Determine ID type from fields
	idType := "string" // default
	for _, field := range fields {
		if field.Name == "id" {
			idType = field.GoType
			break
		}
	}

	file := generators.FileConfig{
		Path:     path,
		Template: typeutil.Must(fs.ReadFile("templates/model_and_service.go.tmpl")),
		Gen: func(g *genhelper.GenHelper) {
			// Use proper pluralization
			tableName := str.ToSnakeCase(s.pluralize.Plural(name))

			g.WithVar("name_pascal", str.ToPascalCase(name)).
				WithVar("name_camel", str.ToCamelCase(name)).
				WithVar("table_name", tableName).
				WithVar("fields", fields).
				WithVar("no_service", s.NoService).
				WithVar("id_type", idType)

			if !s.NoService {
				g.WithImport("context", "")
				g.WithImport("github.com/alexisvisco/goframe/db/pagination", "")
			}

			for _, field := range fields {
				imp := s.importFromType(field)
				if imp != "" {
					g.WithImport(imp, "")
				}
			}
		},
	}

	return s.Gen.GenerateFile(file)
}

func (s *ScaffoldGenerator) importFromType(field Field) string {
	if field.Type == "decimal" {
		return "github.com/shopspring/decimal"
	} else if field.Type == "date" || field.Type == "time" || field.Type == "datetime" || field.Type == "timestamp" || field.Type == "timestampz" {
		return "time"
	}
	return ""
}

func (s *ScaffoldGenerator) generateServiceImplementation(name string, fields []Field) error {
	path := filepath.Join("internal/service", fmt.Sprintf("service_%s.go", str.ToSnakeCase(name)))

	// Determine ID type from fields
	idType := "string" // default
	for _, field := range fields {
		if field.Name == "id" {
			idType = field.GoType
			break
		}
	}

	file := generators.FileConfig{
		Path:     path,
		Template: typeutil.Must(fs.ReadFile("templates/service_implementation.go.tmpl")),
		Gen: func(g *genhelper.GenHelper) {
			g.WithVar("name_pascal", str.ToPascalCase(name)).
				WithVar("name_camel", str.ToCamelCase(name)).
				WithVar("name_snake", str.ToSnakeCase(name)).
				WithVar("id_type", idType).
				WithImport(filepath.Join(s.Gen.GoModuleName, "internal/types"), "types").
				WithImport("gorm.io/gorm", "gorm").
				WithImport("errors", "").
				WithImport("context", "").
				WithImport("github.com/alexisvisco/goframe/db/dbutil", "").
				WithImport("github.com/alexisvisco/goframe/db/pagination", "")

			// Add CUID2 import for string IDs
			if idType == "string" {
				g.WithImport("github.com/nrednav/cuid2", "cuid2")
			}
		},
	}

	return s.Gen.GenerateFile(file)
}

func (s *ScaffoldGenerator) generateMigration(name string, fields []Field) error {
	// Use proper pluralization
	tableName := str.ToSnakeCase(s.pluralize.Plural(name))

	// Build CREATE TABLE statement
	var columns []string
	for _, field := range fields {
		columnName := str.ToSnakeCase(field.Name)
		columns = append(columns, fmt.Sprintf("    %s %s", columnName, field.SqlType))
	}

	upSQL := fmt.Sprintf("CREATE TABLE %s (\n%s\n);", tableName, strings.Join(columns, ",\n"))
	downSQL := fmt.Sprintf("DROP TABLE IF EXISTS %s;", tableName)

	migrationName := fmt.Sprintf("create_%s", tableName)

	return s.DBGen.GenerateMigration(gendb.CreateMigrationParams{
		Sql:  true,
		Name: migrationName,
		At:   time.Now(),
		Up:   upSQL,
		Down: downSQL,
	})
}
