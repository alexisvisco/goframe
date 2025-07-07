package dbcmd

import (
	"fmt"

	"github.com/alexisvisco/goframe/db/seed"
	"github.com/spf13/cobra"
	"gorm.io/gorm"
)

func seedsUpCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "up",
		Short:   "Run unapplied seeds",
		Aliases: []string{"run"},
		RunE: func(cmd *cobra.Command, args []string) error {
			connector, ok := cmd.Context().Value("db").(func() (*gorm.DB, error))
			if !ok || connector == nil {
				return fmt.Errorf("database connector not found")
			}

			db, err := connector()
			if err != nil {
				return fmt.Errorf("failed to connect to database: %w", err)
			}

			seeds, ok := cmd.Context().Value("seeds").([]seed.Seed)
			if !ok || seeds == nil {
				return fmt.Errorf("seeds not found")
			}

			if err := seed.New(db).Up(cmd.Context(), seeds); err != nil {
				return fmt.Errorf("failed to run seeds: %w", err)
			}

			return nil
		},
	}

	return cmd
}
