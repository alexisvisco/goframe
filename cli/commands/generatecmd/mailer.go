package generatecmd

import (
	"fmt"

	"github.com/alexisvisco/goframe/cli/generators"
	"github.com/spf13/cobra"
)

func mailerCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "mailer <name> <action>",
		Short: "Create a new mailer action",
		RunE: withFileDiff(func(cmd *cobra.Command, args []string) error {
			if len(args) < 2 {
				return fmt.Errorf("mailer name and action are required")
			}
			name := args[0]
			action := args[1]
			g := generators.Generator{GoModuleName: cmd.Context().Value("module").(string)}
			if err := g.Mailer().Create(name, action); err != nil {
				return fmt.Errorf("failed to create mailer: %w", err)
			}
			return nil
		}),
	}
}
