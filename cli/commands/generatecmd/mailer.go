package generatecmd

import (
	"fmt"

	"github.com/alexisvisco/goframe/cli/generators"
	"github.com/alexisvisco/goframe/cli/generators/genhelper"
	"github.com/alexisvisco/goframe/cli/generators/genmailer"
	"github.com/spf13/cobra"
)

func mailerCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "mailer <name> <action>",
		Short: "GenerateHandler a new mailer action",
		RunE: genhelper.WithFileDiff(func(cmd *cobra.Command, args []string) error {
			if len(args) < 2 {
				return fmt.Errorf("mailer name and action are required")
			}
			name := args[0]
			action := args[1]
			g := &generators.Generator{GoModuleName: cmd.Context().Value("module").(string)}
			genMailer := genmailer.MailerGenerator{Gen: g}
			if err := genMailer.GenerateMailer(name, action); err != nil {
				return fmt.Errorf("failed to create mailer: %w", err)
			}
			return nil
		}),
	}
}
