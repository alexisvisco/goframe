package dbcmd

import (
	"fmt"

	"github.com/alexisvisco/goframe/db/seed"
	"github.com/spf13/cobra"
	"gorm.io/gorm"
)

func seedsShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show",
		Short: "List applied seeds",
		RunE: func(cmd *cobra.Command, args []string) error {
			connector, ok := cmd.Context().Value("db").(func() (*gorm.DB, error))
			if !ok || connector == nil {
				return fmt.Errorf("database connector not found")
			}

			db, err := connector()
			if err != nil {
				return fmt.Errorf("failed to connect to database: %w", err)
			}

			seeder := seed.New(db)
			list, err := seeder.Applied(cmd.Context())
			if err != nil {
				return fmt.Errorf("failed to list seeds: %w", err)
			}

			for _, s := range list {
				fmt.Fprintln(cmd.OutOrStdout(), s)
			}
			return nil
		},
	}

	return cmd
}
