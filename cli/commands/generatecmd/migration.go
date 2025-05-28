package generatecmd

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/alexisvisco/goframe/cli/generators/genhelper"
	"github.com/alexisvisco/goframe/cli/generators/templates"
	"github.com/alexisvisco/goframe/core/helpers/str"
	"github.com/spf13/cobra"
)

func migrationCmd() *cobra.Command {
	var flagSql bool
	cmd := &cobra.Command{
		Use:   "migration <name>",
		Short: "Create a new migration file",
		Long:  "Create a new migration file with a timestamp and the specified name.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("migration name is required")
			}

			name := args[0]

			if flagSql {
				if err := generateSQLMigrationFile(name); err != nil {
					return fmt.Errorf("failed to generate SQL migration file: %w", err)
				}
				slog.Info("migration created", "name", name, "type", "SQL")
			} else {
				if err := generateGoMigrationFile(name); err != nil {
					return fmt.Errorf("failed to generate Go migration file: %w", err)
				}
				slog.Info("migration created", "name", name, "type", "Go")
			}

			file, err := os.OpenFile("db/migrations.go", os.O_WRONLY|os.O_TRUNC, 0644)
			if err != nil {
				return fmt.Errorf("failed to create migrations.go file: %v", err)
			}

			hasSQLMigrations, hasGoMigrations, list := buildMigrationList()

			g := genhelper.New("db", templates.DBMigrationsGo)

			if hasSQLMigrations {
				g.WithImport("embed", "embed").
					WithVar("has_sql_migrations", "true")
			}

			if hasGoMigrations {
				g.WithImport(filepath.Join(cmd.Context().Value("module").(string), "db/migrations"), "migrations")
			}

			return g.
				WithVar("migrations", list).
				Generate(file)
		},
	}

	cmd.Flags().BoolVarP(&flagSql, "sql", "s", false, "Generate SQL migration file instead of Go code")

	return cmd
}

func buildMigrationList() (bool, bool, []string) {
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

func generateGoMigrationFile(name string) error {
	timestamp := time.Now().UTC().Format("20060102150405")
	nameSnakeCase := str.ToSnakeCase(name)
	namePascalCase := str.ToPascalCase(name)
	version := fmt.Sprintf("%s_%s", timestamp, nameSnakeCase)
	structName := namePascalCase + timestamp

	// func Date(year int, month Month, day, hour, min, sec, nsec int, loc *Location) Time {
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

func generateSQLMigrationFile(name string) error {
	timestamp := time.Now().UTC().Format("20060102150405")
	nameSnakeCase := str.ToSnakeCase(name)
	version := fmt.Sprintf("%s_%s", timestamp, nameSnakeCase)

	file, err := os.Create(fmt.Sprintf("db/migrations/%s.sql", version))
	if err != nil {
		return fmt.Errorf("failed to create SQL migration file: %w", err)
	}

	defer file.Close()

	return genhelper.New("migration", templates.DBMigrationsFileSQL).
		WithVar("name", version).
		Generate(file)
}
