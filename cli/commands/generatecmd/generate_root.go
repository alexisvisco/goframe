package generatecmd

import "github.com/spf13/cobra"

func NewCmdRootGenerate() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "generate <subcommand> [flags]",
		Aliases: []string{"gen", "g"},
		Short:   "Generate code",
	}

	cmd.AddCommand(migrationCmd())

	return cmd
}
