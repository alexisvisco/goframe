package dbcmd

import (
	"fmt"

	"github.com/alexisvisco/goframe/db/seed"
	"github.com/spf13/cobra"
	"gorm.io/gorm"
)

func seedCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "seed",
		Short: "Run pending database seeds",
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

			err = seed.New(db).Up(
				cmd.Context(),
				seeds,
			)
			if err != nil {
				return fmt.Errorf("failed to execute seeds: %w", err)
			}

			return nil
		},
	}

	return cmd
}
