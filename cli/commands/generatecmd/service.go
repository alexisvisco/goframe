package generatecmd

import (
	"fmt"

	"github.com/alexisvisco/goframe/cli/generators"
	"github.com/spf13/cobra"
)

func serviceCmd() *cobra.Command {
	var withRepo bool
	cmd := &cobra.Command{
		Use:   "service <name>",
		Short: "Create a new service",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("service name is required")
			}
			name := args[0]
			g := generators.Generator{GoModuleName: cmd.Context().Value("module").(string)}
			if err := g.Service().Create(name, withRepo); err != nil {
				return fmt.Errorf("failed to create service: %w", err)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&withRepo, "with-repository", false, "inject repository dependency")
	return cmd
}
