package generatecmd

import (
	"fmt"

	"github.com/alexisvisco/goframe/cli/generators"
	"github.com/alexisvisco/goframe/cli/generators/genhelper"
	"github.com/alexisvisco/goframe/cli/generators/genhttp"
	"github.com/spf13/cobra"
)

func routeCmd() *cobra.Command {
	var flagFile bool
	var flagNoMiddleware bool
	var flagPkg string
	var flagPath string
	var flagRequestBody string
	var flagResponseBody string
	var flagRouteBody string
	cmd := &cobra.Command{
		Use:   "route <handler> <route name>",
		Short: "Add a route to an existing handler",
		RunE: genhelper.WithFileDiff(func(cmd *cobra.Command, args []string) error {
			if len(args) < 2 {
				return fmt.Errorf("handler and method are required")
			}
			handler := args[0]
			routeName := args[1]
			g := cmd.Context().Value("generator").(*generators.Generator)
			genHandler := &genhttp.HTTPGenerator{Gen: g, BasePath: flagPkg}
			config := genhttp.RouteConfig{
				Handler:      handler,
				Name:         routeName,
				Path:         flagPath,
				NewFile:      flagFile,
				NoMiddleware: flagNoMiddleware,
				RequestBody:  flagRequestBody,
				ResponseBody: flagResponseBody,
				RouteBody:    flagRouteBody,
			}
			if err := genHandler.GenerateRoute(config); err != nil {
				return fmt.Errorf("failed to create route: %w", err)
			}
			return nil
		}),
	}
	cmd.Flags().BoolVar(&flagFile, "file", false, "generate route in a separate file")
	cmd.Flags().BoolVar(&flagNoMiddleware, "no-middleware", false, "use httpx.Wrap instead of middleware chain")
	cmd.Flags().StringVar(&flagPkg, "pkg", "internal/v1handler", "Package name for the generated route")
	cmd.Flags().StringVar(&flagPath, "path", "", "HTTP path for the route")
	cmd.Flags().StringVar(&flagRequestBody, "request-body", "", "Custom request body struct fields")
	cmd.Flags().StringVar(&flagResponseBody, "response-body", "", "Custom response body struct fields")
	cmd.Flags().StringVar(&flagRouteBody, "route-body", "", "Custom route implementation body")
	return cmd
}
