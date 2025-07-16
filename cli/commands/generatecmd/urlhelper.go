package generatecmd

import (
	"fmt"

	"github.com/alexisvisco/goframe/cli/generators"
	"github.com/alexisvisco/goframe/cli/generators/genhelper"
	"github.com/alexisvisco/goframe/cli/generators/genurlhelper"
	"github.com/spf13/cobra"
)

func urlHelperCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "url-helper [packages...]",
		Short: "Generate url helper functions for routes",
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
			gen := genurlhelper.URLHelperGenerator{Gen: g}
			if err := gen.Generate(routes); err != nil {
				return fmt.Errorf("failed to generate url helper: %w", err)
			}
			return nil
		}),
	}
	return cmd
}
