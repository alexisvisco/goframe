package taskcmd

import (
	"github.com/spf13/cobra"
)

func NewCmdRootTask(subCommands ...*cobra.Command) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "task <subcommand> [flags]",
		Aliases: []string{"t"},
		Short:   "Run your tasks",
	}

	for _, subCmd := range subCommands {
		cmd.AddCommand(subCmd)
	}

	return cmd
}
