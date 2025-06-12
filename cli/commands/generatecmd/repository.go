package generatecmd

import (
	"fmt"

	"github.com/alexisvisco/goframe/cli/generators"
	"github.com/alexisvisco/goframe/cli/generators/genhelper"
	"github.com/alexisvisco/goframe/cli/generators/genrepository"
	"github.com/spf13/cobra"
)

func repositoryCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "repository <name>",
		Aliases: []string{"repo"},
		Short:   "GenerateHandler a new repository",
		RunE: genhelper.WithFileDiff(func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("repository name is required")
			}
			name := args[0]
			g := generators.Generator{GoModuleName: cmd.Context().Value("module").(string)}
			genRepo := genrepository.RepositoryGenerator{Gen: &g}
			if err := genRepo.GenerateRepository(name); err != nil {
				return fmt.Errorf("failed to create repository: %w", err)
			}
			return nil
		}),
	}
}
