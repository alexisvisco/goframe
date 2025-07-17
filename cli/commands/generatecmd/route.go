package generatecmd

import (
	"fmt"

	"github.com/alexisvisco/goframe/cli/generators"
	"github.com/alexisvisco/goframe/cli/generators/genhelper"
	"github.com/alexisvisco/goframe/cli/generators/genhttp"
	"github.com/spf13/cobra"
)

func routeCmd() *cobra.Command {
	var generateFile bool
	var noMiddleware bool
	cmd := &cobra.Command{
		Use:   "route <handler> <method>",
		Short: "Add a route method to an existing handler",
		RunE: genhelper.WithFileDiff(func(cmd *cobra.Command, args []string) error {
			if len(args) < 2 {
				return fmt.Errorf("handler and method are required")
			}
			handler := args[0]
			method := args[1]
			g := &generators.Generator{GoModuleName: cmd.Context().Value("module").(string)}
			genHandler := &genhttp.HTTPGenerator{Gen: g}
			if err := genHandler.GenerateRoute(handler, method, generateFile, noMiddleware); err != nil {
				return fmt.Errorf("failed to create route: %w", err)
			}
			return nil
		}),
	}
	cmd.Flags().BoolVar(&generateFile, "file", false, "generate route in a separate file")
	cmd.Flags().BoolVar(&noMiddleware, "nomiddleware", false, "use httpx.Wrap instead of middleware chain")
	return cmd
}
