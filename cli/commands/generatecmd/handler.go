package generatecmd

import (
	"fmt"

	"github.com/alexisvisco/goframe/cli/generators"
	"github.com/spf13/cobra"
)

func handlerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "handler <name> [services...]",
		Short: "Create a new HTTP handler",
		RunE: withFileDiff(func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("handler name is required")
			}
			name := args[0]
			services := args[1:]
			g := generators.Generator{GoModuleName: cmd.Context().Value("module").(string)}
			if err := g.Handler().Create(name, services); err != nil {
				return fmt.Errorf("failed to create handler: %w", err)
			}
			return nil
		}),
	}
	return cmd
}
