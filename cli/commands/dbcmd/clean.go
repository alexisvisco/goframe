package dbcmd

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/spf13/cobra"
	"gorm.io/gorm"
)

func cleanCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "clean",
		Aliases: []string{"drop", "reset"},
		Short:   "Drop all tables and objects from the database",
		Long:    "Remove all tables and database objects. For PostgreSQL, drops and recreates the public schema. This operation is irreversible and will delete all data.",
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := getDB(cmd)
			if err != nil {
				return err
			}

			dialect := db.Dialector.Name()
			switch dialect {
			case "postgres":
				return cleanPostgres(db)
			case "sqlite":
				return cleanSQLite(db)
			default:
				return fmt.Errorf("unsupported database dialect: %s", dialect)
			}
		},
	}
}

func getDB(cmd *cobra.Command) (*gorm.DB, error) {
	connector, ok := cmd.Context().Value("db").(func() (*gorm.DB, error))
	if !ok || connector == nil {
		return nil, fmt.Errorf("database connector not found")
	}

	db, err := connector()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return db, nil
}

func cleanPostgres(db *gorm.DB) error {
	statements := []string{
		"DROP SCHEMA IF EXISTS public CASCADE",
		"CREATE SCHEMA public",
		"GRANT ALL ON SCHEMA public TO PUBLIC",
	}

	for _, stmt := range statements {
		if err := db.Exec(stmt).Error; err != nil {
			return fmt.Errorf("failed to execute '%s': %w", stmt, err)
		}
	}

	slog.Info("successfully recreated public schema")
	return nil
}

func cleanSQLite(db *gorm.DB) error {
	tables, err := getSQLiteTables(db)
	if err != nil {
		return fmt.Errorf("failed to get table list: %w", err)
	}

	if len(tables) == 0 {
		return nil
	}

	if err := db.Exec("PRAGMA foreign_keys = OFF").Error; err != nil {
		return fmt.Errorf("failed to disable foreign keys: %w", err)
	}

	for _, table := range tables {
		stmt := fmt.Sprintf(`DROP TABLE IF EXISTS "%s"`, table)
		if err := db.Exec(stmt).Error; err != nil {
			_ = db.Exec("PRAGMA foreign_keys = ON") // ensure re-enable
			return fmt.Errorf("failed to drop table %s: %w", table, err)
		}
	}

	if err := db.Exec("PRAGMA foreign_keys = ON").Error; err != nil {
		return fmt.Errorf("failed to re-enable foreign keys: %w", err)
	}

	slog.Info("successfully dropped tables", "tables", strings.Join(tables, ", "))
	return nil
}

func getSQLiteTables(db *gorm.DB) ([]string, error) {
	rows, err := db.Raw(`SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'`).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			return nil, err
		}
		tables = append(tables, table)
	}

	return tables, nil
}
