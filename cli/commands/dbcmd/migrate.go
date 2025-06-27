package dbcmd

import (
	"fmt"

	"github.com/alexisvisco/goframe/db/migrate"
	"github.com/spf13/cobra"
	"gorm.io/gorm"
)

func migrateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "migrate",
		Aliases: []string{"up", "m"},
		Short:   "Migrate the database with non-applied migrations",
		RunE: func(cmd *cobra.Command, args []string) error {
			connector, ok := cmd.Context().Value("db").(func() (*gorm.DB, error))
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

			err = migrate.New(db).Up(cmd.Context(), migrations)
			if err != nil {
				return fmt.Errorf("failed to apply migrations: %w", err)
			}

			return nil
		},
	}

	return cmd
}
