package dbcmd

import "github.com/spf13/cobra"

func seedsRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "seeds <subcommand> [flags]",
		Short: "Manage database seeds",
	}

	cmd.AddCommand(seedsUpCmd())
	cmd.AddCommand(seedsShowCmd())

	return cmd
}
