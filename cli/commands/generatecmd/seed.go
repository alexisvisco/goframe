package generatecmd

import (
	"fmt"
	"time"

	"github.com/alexisvisco/goframe/cli/generators"
	"github.com/alexisvisco/goframe/cli/generators/gendb"
	"github.com/alexisvisco/goframe/cli/generators/genhelper"
	"github.com/spf13/cobra"
)

func seedCmd() *cobra.Command {
	var flagSql bool
	cmd := &cobra.Command{
		Use:   "seed <name>",
		Short: "Generate a new seed file",
		Long:  "Generate a new seed file with a timestamp and the specified name.",
		RunE: genhelper.WithFileDiff(func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("migration name is required")
			}

			name := args[0]

			g := generators.Generator{
				GoModuleName: cmd.Context().Value("module").(string),
				ORMType:      "gorm",
			}

			genDB := &gendb.DatabaseGenerator{Gen: &g}

			return genDB.GenerateSeed(gendb.CreateSeedParams{
				Sql:  flagSql,
				Name: name,
				At:   time.Now(),
			})
		}),
	}

	cmd.Flags().BoolVarP(&flagSql, "sql", "s", false, "WriteTo SQL migration file instead of Go code")

	return cmd
}
