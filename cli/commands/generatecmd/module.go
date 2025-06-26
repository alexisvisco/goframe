package generatecmd

import (
	"fmt"

	"github.com/alexisvisco/goframe/cli/generators"
	"github.com/alexisvisco/goframe/cli/generators/genauth"
	"github.com/alexisvisco/goframe/cli/generators/gendb"
	"github.com/alexisvisco/goframe/cli/generators/genhelper"
	"github.com/alexisvisco/goframe/cli/generators/genhttp"
	"github.com/alexisvisco/goframe/cli/generators/genmailer"
	"github.com/alexisvisco/goframe/cli/generators/genservice"
	"github.com/alexisvisco/goframe/cli/generators/genworker"
	"github.com/spf13/cobra"
)

func moduleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "module <name>",
		Short: "Install a set of files that form a module for a particular functionality or feature",
		RunE: genhelper.WithFileDiff(func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("migration name is required")
			}

			g := &generators.Generator{
				GoModuleName: cmd.Context().Value("module").(string),
				ORMType:      "gorm",
			}

			svcGen := &genservice.ServiceGenerator{Gen: g}
			dbGen := &gendb.DatabaseGenerator{Gen: g}
			httpGen := &genhttp.HTTPGenerator{Gen: g}
			workerGen := &genworker.WorkerGenerator{Gen: g}
			mailerGen := &genmailer.MailerGenerator{Gen: g, Wf: workerGen}
			authGen := &genauth.AuthGenerator{
				Gen:              g,
				MailerGenerator:  mailerGen,
				ServiceGenerator: svcGen,
				HTTPGenerator:    httpGen,
				DBGenerator:      dbGen,
			}

			name := args[0]

			switch name {
			case "auth":
				err := authGen.Generate()
				if err != nil {
					return fmt.Errorf("failed to generate auth module: %w", err)
				}
			}

			return nil
		}),
	}

	return cmd
}
