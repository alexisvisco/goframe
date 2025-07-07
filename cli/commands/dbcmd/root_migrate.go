package dbcmd

import (
	"github.com/spf13/cobra"
)

func NewCmdRootMigrate() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "db <subcommand> [flags]",
		Short: "Database management",
	}

	cmd.AddCommand(migrateCmd())
	cmd.AddCommand(rollbackCmd())
	cmd.AddCommand(cleanCmd())
	cmd.AddCommand(seedsRootCmd())

	return cmd
}
