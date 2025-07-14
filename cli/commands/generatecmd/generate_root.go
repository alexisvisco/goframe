package generatecmd

import (
	"github.com/alexisvisco/goframe/cli/generators/gentsclient"
	"github.com/alexisvisco/goframe/http/apidoc"
	"github.com/spf13/cobra"
)

func NewCmdRootGenerate(subCommands ...*cobra.Command) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "generate <subcommand> [flags]",
		Aliases: []string{"gen", "g"},
		Short:   "WriteTo code",
	}

	cmd.AddCommand(migrationCmd())
	cmd.AddCommand(taskCmd())
	cmd.AddCommand(workerCmd())
	cmd.AddCommand(serviceCmd())
	cmd.AddCommand(handlerCmd())
	cmd.AddCommand(mailerCmd())
	cmd.AddCommand(seedCmd())
	cmd.AddCommand(moduleCmd())
	cmd.AddCommand(NewCmdGenClient())
	for _, subCmd := range subCommands {
		cmd.AddCommand(subCmd)
	}

	return cmd
}

func NewCmdGenClient() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "client <pkg path> <struct name> <method name>",
		Aliases: []string{"gen", "g"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 3 {
				return cmd.Help()
			}

			workdir, _ := cmd.Context().Value("workdir").(string)
			r, err := apidoc.ParseRoute(workdir, args[0], args[1], args[2])
			if err != nil {
				return err
			}

			generator := gentsclient.NewTypescriptClientGenerator()

			generator.AddSchema("", true, *r.Request)
			for _, response := range r.StatusToResponse {
				if response.Response != nil {
					generator.AddSchema("", true, *response.Response)
				}
			}

			generator.AddRoute(*r)

			// Print the JSON result to stdout
			cmd.Println(generator.File())
			return nil
		},
	}

	return cmd
}
