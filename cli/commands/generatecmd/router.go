package generatecmd

import (
	"fmt"

	"github.com/alexisvisco/goframe/cli/generators"
	"github.com/alexisvisco/goframe/cli/generators/genhelper"
	"github.com/alexisvisco/goframe/cli/generators/genhttp"
	"github.com/spf13/cobra"
)

func routerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "router [packages...]",
		Short: "Populate router with routes from handlers",
		RunE: genhelper.WithFileDiff(func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				args = []string{"v1handler"}
			}

			workdir, _ := cmd.Context().Value("workdir").(string)
			routes, err := collectRoutes(workdir, args)
			if err != nil {
				return err
			}

			g := &generators.Generator{GoModuleName: cmd.Context().Value("module").(string)}
			httpGen := &genhttp.HTTPGenerator{Gen: g}
			if err := httpGen.GenerateRoutes(routes); err != nil {
				return fmt.Errorf("failed to update router: %w", err)
			}

			return nil
		}),
	}

	return cmd
}
