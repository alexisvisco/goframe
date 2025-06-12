package generatecmd

import (
	"fmt"
	"time"

	"github.com/alexisvisco/goframe/cli/generators"
	"github.com/spf13/cobra"
)

func migrationCmd() *cobra.Command {
	var flagSql bool
	cmd := &cobra.Command{
		Use:   "migration <name>",
		Short: "Create a new migration file",
		Long:  "Create a new migration file with a timestamp and the specified name.",
		RunE: withFileDiff(func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("migration name is required")
			}

			name := args[0]

			g := generators.Generator{
				GoModuleName: cmd.Context().Value("module").(string),
				ORMType:      "gorm",
			}

			return g.Databases().CreateMigration(generators.CreateMigrationParams{
				Sql:  flagSql,
				Name: name,
				At:   time.Now(),
			})
		}),
	}

	cmd.Flags().BoolVarP(&flagSql, "sql", "s", false, "WriteTo SQL migration file instead of Go code")

	return cmd
}
