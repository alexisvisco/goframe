package generatecmd

import (
	"github.com/alexisvisco/goframe/cli/generators"
	"github.com/alexisvisco/goframe/cli/generators/genauth"
	"github.com/alexisvisco/goframe/cli/generators/gendb"
	"github.com/alexisvisco/goframe/cli/generators/genhelper"
	"github.com/alexisvisco/goframe/cli/generators/genhttp"
	"github.com/alexisvisco/goframe/cli/generators/genimgvariant"
	"github.com/alexisvisco/goframe/cli/generators/genmailer"
	"github.com/alexisvisco/goframe/cli/generators/genservice"
	"github.com/alexisvisco/goframe/cli/generators/genworker"
	"github.com/spf13/cobra"
)

func moduleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "module",
		Short: "Install modules for particular functionalities or features",
	}

	// Auth subcommand
	authCmd := &cobra.Command{
		Use:   "auth",
		Short: "Generate authentication module",
		RunE: genhelper.WithFileDiff(func(cmd *cobra.Command, args []string) error {
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
			return authGen.Generate()
		}),
	}

	// Image variant subcommand
	imgVariantCmd := &cobra.Command{
		Use:   "image-variant",
		Short: "Generate image variant module",
		RunE: genhelper.WithFileDiff(func(cmd *cobra.Command, args []string) error {
			g := &generators.Generator{
				GoModuleName: cmd.Context().Value("module").(string),
				ORMType:      "gorm",
			}

			svcGen := &genservice.ServiceGenerator{Gen: g}
			dbGen := &gendb.DatabaseGenerator{Gen: g}
			workerGen := &genworker.WorkerGenerator{Gen: g}

			imgVariantGen := &genimgvariant.ImageVariantGenerator{
				Gen:               g,
				ServiceGenerator:  svcGen,
				WorkflowGenerator: workerGen,
				DBGenerator:       dbGen,
			}
			return imgVariantGen.Generate()
		}),
	}

	// Add subcommands
	cmd.AddCommand(authCmd)
	cmd.AddCommand(imgVariantCmd)

	return cmd
}
