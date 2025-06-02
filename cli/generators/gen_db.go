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
	GenerateDatabaseFiles struct {
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

func (g *GenerateDatabaseFiles) CreateMigration(params CreateMigrationParams) error {
	if params.Sql {
		if _, err := g.generateSQLMigrationFile(params); err != nil {
			return fmt.Errorf("failed to generate SQL migration file: %w", err)
		}
	} else {
		if err := g.generateGoMigrationFile(params); err != nil {
			return fmt.Errorf("failed to generate Go migration file: %w", err)
		}
	}

	return g.UpdateMigrations()
}

func (g *GenerateDatabaseFiles) UpdateMigrations() error {
	file, err := os.OpenFile("db/migrations.go", os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to create migrations.go file: %v", err)
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

	return gh.
		WithVar("migrations", list).
		Generate(file)
}

func (g *GenerateDatabaseFiles) buildMigrationList() (bool, bool, []string) {
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

func (g *GenerateDatabaseFiles) generateGoMigrationFile(c CreateMigrationParams) error {
	timestamp := c.At.UTC().Format("20060102150405")
	nameSnakeCase := str.ToSnakeCase(c.Name)
	namePascalCase := str.ToPascalCase(c.Name)
	version := fmt.Sprintf("%s_%s", timestamp, nameSnakeCase)
	structName := namePascalCase + timestamp

	nowUTC := time.Now().UTC()
	date := fmt.Sprintf("%d, %d, %d, %d, %d, %d, %d",
		nowUTC.Year(), nowUTC.Month(), nowUTC.Day(),
		nowUTC.Hour(), nowUTC.Minute(), nowUTC.Second(), time.Now().Nanosecond())

	file, err := os.Create(fmt.Sprintf("db/migrations/%s.go", version))
	if err != nil {
		return fmt.Errorf("failed to create migration file: %w", err)
	}
	defer file.Close()

	return genhelper.New("migration", templates.DBMigrationsFileGo).
		WithVar("struct", structName).
		WithVar("date", date).
		WithVar("version", version).
		WithVar("name", nameSnakeCase).
		Generate(file)
}

func (g *GenerateDatabaseFiles) generateSQLMigrationFile(c CreateMigrationParams) (string, error) {
	timestamp := c.At.UTC().Format("20060102150405")
	nameSnakeCase := str.ToSnakeCase(c.Name)
	version := fmt.Sprintf("%s_%s", timestamp, nameSnakeCase)

	file, err := os.Create(fmt.Sprintf("db/migrations/%s.sql", version))
	if err != nil {
		return "", fmt.Errorf("failed to create SQL migration file: %w", err)
	}

	defer file.Close()

	gh := genhelper.New("migration", templates.DBMigrationsFileSQL).
		WithVar("name", version).
		WithVar("up", c.Up).
		WithVar("down", c.Down)

	err = gh.Generate(file)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("db/migrations/%s.sql", version), nil
}
