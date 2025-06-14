package generatecmd

import (
	"fmt"

	"github.com/alexisvisco/goframe/cli/generators"
	"github.com/alexisvisco/goframe/cli/generators/genhelper"
	"github.com/alexisvisco/goframe/cli/generators/genhttp"
	"github.com/alexisvisco/goframe/cli/generators/genhttpexample"
	"github.com/alexisvisco/goframe/cli/generators/genrepository"
	"github.com/alexisvisco/goframe/cli/generators/genservice"
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

			//cfgGen := &genconfig.ConfigGenerator{Gen: g}
			//coreGen := &gencore.CoreGenerator{Gen: g}
			repoGen := &genrepository.RepositoryGenerator{Gen: g}
			svcGen := &genservice.ServiceGenerator{Gen: g}
			//dbGen := &gendb.DatabaseGenerator{Gen: g}
			//storageGen := &genstorage.StorageGenerator{Gen: g, DBGen: dbGen}
			//dockerGen := &gendocker.DockerGenerator{Gen: g}
			httpGen := &genhttp.HTTPGenerator{Gen: g}
			//workerGen := &genworker.WorkerGenerator{Gen: g}
			//mailerGen := &genmailer.MailerGenerator{Gen: g, Wf: workerGen}
			exampleHttpGen := &genhttpexample.NoteExampleGenerator{
				Gen:     g,
				GenHTTP: httpGen,
				GenSvc:  svcGen,
				GenRepo: repoGen,
			}

			name := args[0]

			switch name {
			case "example":
				err := exampleHttpGen.Generate()
				if err != nil {
					return fmt.Errorf("failed to create example module: %w", err)
				}
			}

			return nil
		}),
	}

	return cmd
}
