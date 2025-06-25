package gendb

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/alexisvisco/goframe/cli/generators"
	"github.com/alexisvisco/goframe/cli/generators/genhelper"
	"github.com/alexisvisco/goframe/core/helpers/str"
	"github.com/alexisvisco/goframe/core/helpers/typeutil"
)

type (
	DatabaseGenerator struct {
		Gen *generators.Generator
	}

	CreateMigrationParams struct {
		Sql  bool
		Name string
		At   time.Time

		// only for sql migrations
		Up string

		// only for sql migrations
		Down string
	}
)

//go:embed templates
var fs embed.FS

func (g *DatabaseGenerator) Generate() error {
	if err := g.Gen.CreateDirectory("db/migrations"); err != nil {
		return fmt.Errorf("failed to create migrations directory: %w", err)
	}

	files := []generators.FileConfig{
		g.createDBProvider("internal/provide/provide_db.go"),
		g.updateOrCreateMigrations("db/migrations.go"),
	}

	if err := g.Gen.GenerateFiles(files); err != nil {
		return err
	}

	return nil
}

// createDBProvider creates the FileConfig for the database provider
func (g *DatabaseGenerator) createDBProvider(path string) generators.FileConfig {
	return generators.FileConfig{
		Path:     path,
		Template: typeutil.Must(fs.ReadFile("templates/provide_db.go.tmpl")),
		Gen: func(gen *genhelper.GenHelper) {
			gen.WithImport(filepath.Join(g.Gen.GoModuleName, "config"), "config").
				WithImport(filepath.Join(g.Gen.GoModuleName, "db"), "db").
				WithImport("gorm.io/gorm", "gorm").
				WithVar("db", g.Gen.DatabaseType)

			switch g.Gen.DatabaseType {
			case "postgres":
				gen.WithImport("gorm.io/driver/postgres", "postgres")
			case "sqlite":
				gen.WithImport("gorm.io/driver/sqlite", "sqlite")
			}
		},
	}
}

func (g *DatabaseGenerator) GenerateMigration(params CreateMigrationParams) error {
	if len(params.Name) == 0 {
		return fmt.Errorf("migration name is required")
	}

	if params.At.IsZero() {
		params.At = time.Now()
	}

	files := g.CreateMigration(params)

	if err := g.Gen.GenerateFiles(files); err != nil {
		return fmt.Errorf("failed to generate migration files: %w", err)
	}

	return nil
}

func (g *DatabaseGenerator) CreateMigration(params CreateMigrationParams) []generators.FileConfig {
	var migrationFile generators.FileConfig

	if params.Sql {
		migrationFile = g.createSQLMigrationFile(params)
	} else {
		migrationFile = g.createGoMigrationFile(params)
	}

	return []generators.FileConfig{migrationFile, g.updateOrCreateMigrations("db/migrations.go")}
}

func (g *DatabaseGenerator) updateOrCreateMigrations(path string) generators.FileConfig {

	return generators.FileConfig{
		Path:     path,
		Template: typeutil.Must(fs.ReadFile("templates/migrations.go.tmpl")),
		Gen: func(gh *genhelper.GenHelper) {
			hasSQLMigrations, hasGoMigrations, list := g.buildMigrationList()

			if hasSQLMigrations {
				gh.WithImport("embed", "embed").
					WithVar("has_sql_migrations", "true")
			}

			if hasGoMigrations {
				gh.WithImport(filepath.Join(g.Gen.GoModuleName, "db/migrations"), "migrations")
			}

			gh.WithVar("migrations", list)
		},
	}
}

// createGoMigrationFile creates the FileConfig for a Go migration file
func (g *DatabaseGenerator) createGoMigrationFile(c CreateMigrationParams) generators.FileConfig {
	timestamp := c.At.UTC().Format("20060102150405")
	nameSnakeCase := str.ToSnakeCase(c.Name)
	namePascalCase := str.ToPascalCase(c.Name)
	version := fmt.Sprintf("%s_%s", timestamp, nameSnakeCase)
	structName := namePascalCase + timestamp

	nowUTC := time.Now().UTC()
	date := fmt.Sprintf("%d, %d, %d, %d, %d, %d, %d",
		nowUTC.Year(), nowUTC.Month(), nowUTC.Day(),
		nowUTC.Hour(), nowUTC.Minute(), nowUTC.Second(), time.Now().Nanosecond())

	return generators.FileConfig{
		Path:     fmt.Sprintf("db/migrations/%s.go", version),
		Template: typeutil.Must(fs.ReadFile("templates/new_migration.go.tmpl")),
		Gen: func(gen *genhelper.GenHelper) {
			gen.WithVar("struct", structName).
				WithVar("date", date).
				WithVar("version", version).
				WithVar("name", nameSnakeCase)
		},
	}
}

// createSQLMigrationFile creates the FileConfig for a SQL migration file
func (g *DatabaseGenerator) createSQLMigrationFile(c CreateMigrationParams) generators.FileConfig {
	timestamp := c.At.UTC().Format("20060102150405")
	nameSnakeCase := str.ToSnakeCase(c.Name)
	version := fmt.Sprintf("%s_%s", timestamp, nameSnakeCase)

	return generators.FileConfig{
		Path:     fmt.Sprintf("db/migrations/%s.sql", version),
		Template: typeutil.Must(fs.ReadFile("templates/new_migration.sql.tmpl")),
		Gen: func(gen *genhelper.GenHelper) {
			gen.WithVar("name", version).
				WithVar("up", c.Up).
				WithVar("down", c.Down)
		},
	}
}

func (g *DatabaseGenerator) buildMigrationList() (bool, bool, []string) {
	migrationsDir := "db/migrations"

	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return false, false, []string{}
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	var migrations []string
	hasSqlFiles := false
	hasGoFiles := false

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filename := entry.Name()

		if strings.HasSuffix(filename, ".go") {
			hasGoFiles = true
			nameWithoutExt := strings.TrimSuffix(filename, ".go")
			parts := strings.SplitN(nameWithoutExt, "_", 2)

			if len(parts) == 2 {
				timestamp := parts[0]
				nameSnakeCase := parts[1]
				namePascalCase := str.ToPascalCase(nameSnakeCase)
				structName := namePascalCase + timestamp
				migrations = append(migrations, "migrations."+structName+"{},")
			}
		} else if strings.HasSuffix(filename, ".sql") {
			hasSqlFiles = true
			migrations = append(migrations, fmt.Sprintf(`migrate.MigrationFromSQL(fs, "migrations/%s"),`, filename))
		}
	}

	return hasSqlFiles, hasGoFiles, migrations
}
