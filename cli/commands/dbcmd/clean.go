package dbcmd

import (
	"database/sql"
	"fmt"
	"log/slog"
	"strings"

	"github.com/spf13/cobra"
)

func cleanCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "clean",
		Aliases: []string{"drop", "reset"},
		Short:   "Drop all tables from the database with cascade",
		Long:    "Remove all tables from the database. This operation is irreversible and will delete all data.",
		RunE: func(cmd *cobra.Command, args []string) error {
			connector, ok := cmd.Context().Value("db").(func() (*sql.DB, error))
			if !ok || connector == nil {
				return fmt.Errorf("database connector not found")
			}

			db, err := connector()
			if err != nil {
				return fmt.Errorf("failed to connect to database: %w", err)
			}
			defer db.Close()

			// Detect database type
			dbType, err := detectDatabaseType(db)
			if err != nil {
				return fmt.Errorf("failed to detect database type: %w", err)
			}

			// Get all table names
			tables, err := getAllTables(db, dbType)
			if err != nil {
				return fmt.Errorf("failed to get table list: %w", err)
			}

			if len(tables) == 0 {
				return nil
			}

			// Drop all tables
			err = dropAllTables(db, dbType, tables)
			if err != nil {
				return fmt.Errorf("failed to drop tables: %w", err)
			}

			slog.Info("successfully drop tables", "tables", strings.Join(tables, ", "))
			return nil
		},
	}

	return cmd
}

// detectDatabaseType detects the database type from the driver name
func detectDatabaseType(db *sql.DB) (string, error) {
	// Try to get driver name through a query-based approach
	var dbType string

	// Test for PostgreSQL
	if err := db.QueryRow("SELECT version()").Scan(&dbType); err == nil {
		if strings.Contains(strings.ToLower(dbType), "postgresql") {
			return "postgres", nil
		}
	}

	// Test for SQLite
	if err := db.QueryRow("SELECT sqlite_version()").Scan(&dbType); err == nil {
		return "sqlite", nil
	}

	return "", fmt.Errorf("unsupported database type")
}

// getAllTables returns all table names for the given database type
func getAllTables(db *sql.DB, dbType string) ([]string, error) {
	var query string

	switch dbType {
	case "postgres":
		query = `
			SELECT tablename 
			FROM pg_tables 
			WHERE schemaname = 'public'
			ORDER BY tablename`
	case "sqlite":
		query = `
			SELECT name 
			FROM sqlite_master 
			WHERE type = 'table' 
			AND name NOT LIKE 'sqlite_%'
			ORDER BY name`
	default:
		return nil, fmt.Errorf("unsupported database type: %s", dbType)
	}

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return nil, err
		}
		tables = append(tables, tableName)
	}

	return tables, rows.Err()
}

// dropAllTables drops all tables with appropriate cascade behavior
func dropAllTables(db *sql.DB, dbType string, tables []string) error {
	switch dbType {
	case "postgres":
		return dropTablesPostgres(db, tables)
	case "sqlite":
		return dropTablesSQLite(db, tables)
	default:
		return fmt.Errorf("unsupported database type: %s", dbType)
	}
}

// dropTablesPostgres drops all tables in PostgreSQL with CASCADE
func dropTablesPostgres(db *sql.DB, tables []string) error {
	if len(tables) == 0 {
		return nil
	}

	// Disable foreign key checks temporarily and drop with CASCADE
	for _, table := range tables {
		query := fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", quoteIdentifier(table, "postgres"))
		if _, err := db.Exec(query); err != nil {
			return fmt.Errorf("failed to drop table %s: %w", table, err)
		}
	}

	return nil
}

// dropTablesSQLite drops all tables in SQLite
func dropTablesSQLite(db *sql.DB, tables []string) error {
	if len(tables) == 0 {
		return nil
	}

	// SQLite doesn't have CASCADE, but we can disable foreign keys temporarily
	if _, err := db.Exec("PRAGMA foreign_keys = OFF"); err != nil {
		return fmt.Errorf("failed to disable foreign key checks: %w", err)
	}

	// Drop all tables
	for _, table := range tables {
		query := fmt.Sprintf("DROP TABLE IF EXISTS %s", quoteIdentifier(table, "sqlite"))
		if _, err := db.Exec(query); err != nil {
			// Re-enable foreign key checks before returning error
			db.Exec("PRAGMA foreign_keys = ON")
			return fmt.Errorf("failed to drop table %s: %w", table, err)
		}
	}

	// Re-enable foreign key checks
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return fmt.Errorf("failed to re-enable foreign key checks: %w", err)
	}

	return nil
}

// quoteIdentifier properly quotes table/column names for different databases
func quoteIdentifier(name, dbType string) string {
	switch dbType {
	case "postgres":
		return fmt.Sprintf(`"%s"`, name)
	case "sqlite":
		return fmt.Sprintf(`"%s"`, name)
	default:
		return name
	}
}
