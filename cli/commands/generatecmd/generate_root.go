package generatecmd

import "github.com/spf13/cobra"

func NewCmdRootGenerate(subCommands ...*cobra.Command) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "generate <subcommand> [flags]",
		Aliases: []string{"gen", "g"},
		Short:   "Generate code",
	}

	cmd.AddCommand(migrationCmd())
	cmd.AddCommand(taskCmd())
       cmd.AddCommand(workerCmd())
       cmd.AddCommand(repositoryCmd())
       cmd.AddCommand(serviceCmd())
       cmd.AddCommand(mailerCmd())
	for _, subCmd := range subCommands {
		cmd.AddCommand(subCmd)
	}

	return cmd
}
