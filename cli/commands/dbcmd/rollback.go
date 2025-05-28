package dbcmd

import (
	"database/sql"
	"fmt"

	"github.com/alexisvisco/goframe/db/migrate"
	"github.com/spf13/cobra"
)

func rollbackCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "rollback",
		Aliases: []string{"down", "r"},
		Short:   "Migrate the database to the previous version",
		RunE: func(cmd *cobra.Command, args []string) error {
			connector, ok := cmd.Context().Value("db").(func() (*sql.DB, error))
			if !ok || connector == nil {
				return fmt.Errorf("database connector not found")
			}

			db, err := connector()
			if err != nil {
				return fmt.Errorf("failed to connect to database: %w", err)
			}

			migrations, ok := cmd.Context().Value("migrations").([]migrate.Migration)
			if !ok || migrations == nil {
				return fmt.Errorf("migrations not found")
			}

			err = migrate.New(db).Down(cmd.Context(), migrations)
			if err != nil {
				return fmt.Errorf("failed to apply migrations: %w", err)
			}

			return nil
		},
	}

	return cmd
}
