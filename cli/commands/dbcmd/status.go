package dbcmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/alexisvisco/goframe/db/migrate"
	"github.com/spf13/cobra"
	"gorm.io/gorm"
)

func statusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "status",
		Aliases: []string{"st"},
		Short:   "Show migration status",
		Long: `Show the status of all migrations, indicating which are applied and which are pending.

Examples:
  # Show migration status
  goframe db status`,
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

			// Get all applied migrations
			applied, err := migrator.Applied(cmd.Context())
			if err != nil {
				return fmt.Errorf("failed to get applied migrations: %w", err)
			}

			// Create a map for quick lookup
			appliedMap := make(map[string]bool)
			for _, version := range applied {
				appliedMap[version] = true
			}

			// Show status header
			// Create tabwriter for aligned output
			w := tabwriter.NewWriter(os.Stdout, 0, 8, 1, ' ', 0)

			// Print header
			fmt.Fprintf(w, "STATUS\tMIGRATION NAME\n")

			// Check each migration
			for _, migration := range migrations {
				name, at := migration.Version()
				version := fmt.Sprintf("%s_%s", at.UTC().Format("20060102150405"), name)

				status := "PENDING"
				if appliedMap[version] {
					status = "APPLIED"
				}

				fmt.Fprintf(w, "%s\t%s\n", status, name)
			}

			w.Flush()

			fmt.Println()
			fmt.Printf("Total: %d migrations (%d applied, %d pending)\n",
				len(migrations),
				len(applied),
				len(migrations)-len(applied))

			return nil
		},
	}

	return cmd
}
