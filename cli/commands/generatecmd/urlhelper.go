package generatecmd

import (
	"github.com/alexisvisco/goframe/cli/generators"
	"github.com/alexisvisco/goframe/cli/generators/genhelper"
	"github.com/alexisvisco/goframe/cli/generators/genurlhelper"
	"github.com/spf13/cobra"
)

func urlHelperCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "urlhelper",
		Short: "Generate URL helper functions",
		RunE: genhelper.WithFileDiff(func(cmd *cobra.Command, args []string) error {
			g := &generators.Generator{GoModuleName: cmd.Context().Value("module").(string)}
			gen := genurlhelper.URLHelperGenerator{Gen: g}
			return gen.Generate()
		}),
	}
}
