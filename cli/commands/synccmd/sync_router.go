package synccmd

import (
	"fmt"

	"github.com/alexisvisco/goframe/cli/generators"
	"github.com/alexisvisco/goframe/cli/generators/genhelper"
	"github.com/alexisvisco/goframe/cli/generators/genhttp"
	"github.com/spf13/cobra"
)

func syncRouter() *cobra.Command {
	var flagPkg string
	cmd := &cobra.Command{
		Use:   "router",
		Short: "Populate router with routes from handlers",
		Long: "This command scans the specified package for HTTP handlers and generates or updates the router with the collected routes." +
			"\nIt will find all recursive folder that are not 'root' folders (i.e folders that contain router.go or registry.go) and collect routes from handlers in those folders.",
		RunE: genhelper.WithFileDiff(func(cmd *cobra.Command, args []string) error {
			workdir, _ := cmd.Context().Value("workdir").(string)

			g := &generators.Generator{
				GoModuleName: cmd.Context().Value("module").(string),
				WorkDir:      workdir,
			}

			httpGen := &genhttp.HTTPGenerator{
				Gen: g,
			}

			err := httpGen.GenerateRoutes()
			if err != nil {
				return fmt.Errorf("failed to generate routes: %w", err)
			}

			return nil
		}),
	}

	cmd.Flags().StringVar(&flagPkg, "pkg", genhttp.DefaultBasePath, "Package path where handlers are located (e.g., internal/v1handler/dashboard)")

	return cmd
}
