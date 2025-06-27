package dbcmd

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/spf13/cobra"
	"gorm.io/gorm"
)

func cleanCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "clean",
		Aliases: []string{"drop", "reset"},
		Short:   "Drop all tables from the database with cascade",
		Long:    "Remove all tables from the database. This operation is irreversible and will delete all data.",
		RunE: func(cmd *cobra.Command, args []string) error {
			connector, ok := cmd.Context().Value("db").(func() (*gorm.DB, error))
			if !ok || connector == nil {
				return fmt.Errorf("database connector not found")
			}

			db, err := connector()
			if err != nil {
				return fmt.Errorf("failed to connect to database: %w", err)
			}

			// Detect database type
			dbType, err := detectDatabaseTypeGORM(db)
			if err != nil {
				return fmt.Errorf("failed to detect database type: %w", err)
			}

			// Get all table names
			tables, err := getAllTablesGORM(db, dbType)
			if err != nil {
				return fmt.Errorf("failed to get table list: %w", err)
			}

			if len(tables) == 0 {
				return nil
			}

			// Drop all tables
			if err := dropAllTablesGORM(db, dbType, tables); err != nil {
				return fmt.Errorf("failed to drop tables: %w", err)
			}

			slog.Info("successfully dropped tables", "tables", strings.Join(tables, ", "))
			return nil
		},
	}

	return cmd
}

func detectDatabaseTypeGORM(db *gorm.DB) (string, error) {
	dialect := db.Dialector.Name()
	switch dialect {
	case "postgres":
		return "postgres", nil
	case "sqlite":
		return "sqlite", nil
	default:
		return "", fmt.Errorf("unsupported database dialect: %s", dialect)
	}
}

func getAllTablesGORM(db *gorm.DB, dbType string) ([]string, error) {
	var tables []string

	switch dbType {
	case "postgres":
		rows, err := db.Raw(`SELECT tablename FROM pg_tables WHERE schemaname = 'public'`).Rows()
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		for rows.Next() {
			var table string
			if err := rows.Scan(&table); err != nil {
				return nil, err
			}
			tables = append(tables, table)
		}

	case "sqlite":
		rows, err := db.Raw(`SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'`).Rows()
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		for rows.Next() {
			var table string
			if err := rows.Scan(&table); err != nil {
				return nil, err
			}
			tables = append(tables, table)
		}

	default:
		return nil, fmt.Errorf("unsupported database type: %s", dbType)
	}

	return tables, nil
}

func dropAllTablesGORM(db *gorm.DB, dbType string, tables []string) error {
	switch dbType {
	case "postgres":
		for _, table := range tables {
			stmt := fmt.Sprintf(`DROP TABLE IF EXISTS "%s" CASCADE`, table)
			if err := db.Exec(stmt).Error; err != nil {
				return fmt.Errorf("failed to drop table %s: %w", table, err)
			}
		}
	case "sqlite":
		if err := db.Exec("PRAGMA foreign_keys = OFF").Error; err != nil {
			return fmt.Errorf("failed to disable foreign keys: %w", err)
		}
		for _, table := range tables {
			stmt := fmt.Sprintf(`DROP TABLE IF EXISTS "%s"`, table)
			if err := db.Exec(stmt).Error; err != nil {
				_ = db.Exec("PRAGMA foreign_keys = ON") // ensure re-enable even if error
				return fmt.Errorf("failed to drop table %s: %w", table, err)
			}
		}
		if err := db.Exec("PRAGMA foreign_keys = ON").Error; err != nil {
			return fmt.Errorf("failed to re-enable foreign keys: %w", err)
		}
	default:
		return fmt.Errorf("unsupported database type: %s", dbType)
	}
	return nil
}
