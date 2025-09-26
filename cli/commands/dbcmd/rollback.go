package dbcmd

import (
	"fmt"

	"github.com/alexisvisco/goframe/db/migrate"
	"github.com/spf13/cobra"
	"gorm.io/gorm"
)

func rollbackCmd() *cobra.Command {
	var steps int

	cmd := &cobra.Command{
		Use:     "rollback",
		Aliases: []string{"down", "r"},
		Short:   "Rollback database migrations",
		Long: `Rollback database migrations in reverse chronological order.

Examples:
  # Rollback all applied migrations
  goframe db rollback -s 0

  # Rollback only the last 3 migrations
  goframe db rollback --steps 3

  # Rollback only the last migration
  goframe db rollback -s 1

Use --steps to specify how many migrations to rollback. If not specified, all applied migrations will be rolled back.`,
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

			migrator := migrate.New(db)
			if steps > 0 {
				err = migrator.DownSteps(cmd.Context(), migrations, steps)
			} else {
				err = migrator.DownAll(cmd.Context(), migrations)
			}
			if err != nil {
				return fmt.Errorf("failed to rollback migrations: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().IntVarP(&steps, "steps", "s", 1, "Number of migrations to rollback (0 = rollback all)")

	return cmd
}
