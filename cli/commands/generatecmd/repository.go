package generatecmd

import (
	"fmt"

	"github.com/alexisvisco/goframe/cli/generators"
	"github.com/spf13/cobra"
)

func repositoryCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "repository <name>",
		Aliases: []string{"repo"},
		Short:   "Create a new repository",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("repository name is required")
			}
			name := args[0]
			g := generators.Generator{GoModuleName: cmd.Context().Value("module").(string)}
			if err := g.Repository().Create(name); err != nil {
				return fmt.Errorf("failed to create repository: %w", err)
			}
			return nil
		},
	}
}
