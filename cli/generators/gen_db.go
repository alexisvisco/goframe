package generators

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/alexisvisco/goframe/cli/generators/genhelper"
	"github.com/alexisvisco/goframe/cli/generators/templates"
	"github.com/alexisvisco/goframe/core/helpers/str"
)

type (
	DatabaseGenerator struct {
		g *Generator
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

func (g *DatabaseGenerator) Generate() error {
	if err := g.g.CreateDirectory("db/migrations", CategoryDatabase); err != nil {
		return fmt.Errorf("failed to create migrations directory: %w", err)
	}

	files := []FileConfig{
		g.createDBProvider("internal/providers/db.go"),
	}

	for _, file := range files {
		if err := g.g.GenerateFile(file); err != nil {
			return fmt.Errorf("failed to create database file %s: %w", file.Path, err)
		}
	}

	if err := g.UpdateOrCreateMigrations(); err != nil {
		return err
	}

	return nil
}

// createDBProvider creates the FileConfig for the database provider
func (g *DatabaseGenerator) createDBProvider(path string) FileConfig {
	return FileConfig{
		Path:     path,
		Template: templates.ProvidersProvideDBGo,
		Gen: func(gen *genhelper.GenHelper) {
			gen.WithImport(filepath.Join(g.g.GoModuleName, "config"), "config").
				WithImport(filepath.Join(g.g.GoModuleName, "db"), "db").
				WithImport("gorm.io/gorm", "gorm").
				WithVar("db", g.g.DatabaseType)

			switch g.g.DatabaseType {
			case "postgres":
				gen.WithImport("gorm.io/driver/postgres", "postgres")
			case "sqlite":
				gen.WithImport("gorm.io/driver/sqlite", "sqlite")
			}
		},
		Category:  CategoryDatabase,
		Condition: true,
	}
}
func (g *DatabaseGenerator) CreateMigration(params CreateMigrationParams) error {
	var migrationFile FileConfig

	if params.Sql {
		migrationFile = g.createSQLMigrationFile(params)
	} else {
		migrationFile = g.createGoMigrationFile(params)
	}

	if err := g.g.GenerateFile(migrationFile); err != nil {
		return fmt.Errorf("failed to generate migration file: %w", err)
	}

	return g.UpdateOrCreateMigrations()
}

// createGoMigrationFile creates the FileConfig for a Go migration file
func (g *DatabaseGenerator) createGoMigrationFile(c CreateMigrationParams) FileConfig {
	timestamp := c.At.UTC().Format("20060102150405")
	nameSnakeCase := str.ToSnakeCase(c.Name)
	namePascalCase := str.ToPascalCase(c.Name)
	version := fmt.Sprintf("%s_%s", timestamp, nameSnakeCase)
	structName := namePascalCase + timestamp

	nowUTC := time.Now().UTC()
	date := fmt.Sprintf("%d, %d, %d, %d, %d, %d, %d",
		nowUTC.Year(), nowUTC.Month(), nowUTC.Day(),
		nowUTC.Hour(), nowUTC.Minute(), nowUTC.Second(), time.Now().Nanosecond())

	return FileConfig{
		Path:     fmt.Sprintf("db/migrations/%s.go", version),
		Template: templates.DBMigrationsFileGo,
		Gen: func(gen *genhelper.GenHelper) {
			gen.WithVar("struct", structName).
				WithVar("date", date).
				WithVar("version", version).
				WithVar("name", nameSnakeCase)
		},
		Category:  CategoryDatabase,
		Condition: true,
	}
}

// createSQLMigrationFile creates the FileConfig for a SQL migration file
func (g *DatabaseGenerator) createSQLMigrationFile(c CreateMigrationParams) FileConfig {
	timestamp := c.At.UTC().Format("20060102150405")
	nameSnakeCase := str.ToSnakeCase(c.Name)
	version := fmt.Sprintf("%s_%s", timestamp, nameSnakeCase)

	return FileConfig{
		Path:     fmt.Sprintf("db/migrations/%s.sql", version),
		Template: templates.DBMigrationsFileSQL,
		Gen: func(gen *genhelper.GenHelper) {
			gen.WithVar("name", version).
				WithVar("up", c.Up).
				WithVar("down", c.Down)
		},
		Category:  CategoryDatabase,
		Condition: true,
	}
}

func (g *DatabaseGenerator) UpdateOrCreateMigrations() error {
	// Create or open the migrations.go file
	var file *os.File
	if _, err := os.Stat("db/migrations.go"); os.IsNotExist(err) {
		// If the file does not exist, create it
		file, err = os.Create("db/migrations.go")
		if err != nil {
			return fmt.Errorf("failed to create migrations.go file: %v", err)
		}
	} else {
		// If the file exists, open it for writing
		file, err = os.OpenFile("db/migrations.go", os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			return fmt.Errorf("failed to open migrations.go file: %v", err)
		}
	}

	hasSQLMigrations, hasGoMigrations, list := g.buildMigrationList()

	gh := genhelper.New("db", templates.DBMigrationsGo)

	if hasSQLMigrations {
		gh.WithImport("embed", "embed").
			WithVar("has_sql_migrations", "true")
	}

	if hasGoMigrations {
		gh.WithImport(filepath.Join(g.g.GoModuleName, "db/migrations"), "migrations")
	}

	g.g.TrackFile("db/migrations.go", false, CategoryDatabase)

	return gh.
		WithVar("migrations", list).
		Generate(file)
}

func (g *DatabaseGenerator) buildMigrationList() (bool, bool, []string) {
	migrationsDir := "db/migrations"

	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return false, false, []string{}
	}

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
