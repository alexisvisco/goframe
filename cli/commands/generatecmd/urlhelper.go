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
			g := cmd.Context().Value("generator").(*generators.Generator)
			gen := genurlhelper.URLHelperGenerator{Gen: g}
			if err := gen.Generate(); err != nil {
				return fmt.Errorf("failed to generate url helper: %w", err)
			}
			return nil
		}),
	}
	return cmd
}
