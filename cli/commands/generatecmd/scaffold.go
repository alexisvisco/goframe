package generatecmd

import (
	"fmt"
	"strings"

	"github.com/alexisvisco/goframe/cli/generators"
	"github.com/alexisvisco/goframe/cli/generators/genhelper"
	"github.com/alexisvisco/goframe/cli/generators/genscaffold"
	"github.com/spf13/cobra"
)

func scaffoldCmd() *cobra.Command {
	var noService bool
	var noHandler bool
	var noMigration bool
	var crud bool

	cmd := &cobra.Command{
		Use:   "scaffold <name> [field:type...]",
		Short: "Generate a complete scaffold with model, service, handler, and migration",
		Long: `Generate a complete scaffold for a resource including:
- Model definition in types package
- Service interface and implementation with CRUD operations
- HTTP handler (if not disabled)
- Database migration (if not disabled)
- CRUD operations (default false)

Field types supported:
  string      - VARCHAR(255)
  text        - TEXT
  integer     - INTEGER
  int         - INTEGER
  bigint      - BIGINT
  float       - FLOAT
  decimal     - DECIMAL(10,2)
  boolean     - BOOLEAN
  binary      - BYTEA
  date        - DATE
  time        - TIME
  datetime    - TIMESTAMP
  timestamp   - TIMESTAMP
  timestampz  - TIMESTAMPTZ

Examples:
  generate scaffold Post title:string content:text author:string
  generate scaffold User name:string email:string age:integer active:boolean
  generate scaffold Product name:string price:decimal description:text --no-handler
`,
		RunE: genhelper.WithFileDiff(func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("scaffold name is required")
			}

			name := args[0]
			fieldSpecs := args[1:]

			// Validate field specifications
			for _, spec := range fieldSpecs {
				if !strings.Contains(spec, ":") {
					return fmt.Errorf("invalid field specification: %s (expected format: name:type)", spec)
				}
			}

			g := cmd.Context().Value("generator").(*generators.Generator)
			scaffoldGen := genscaffold.NewScaffoldGenerator(g)

			// Set flags
			scaffoldGen.NoService = noService
			scaffoldGen.NoHandler = noHandler
			scaffoldGen.NoMigration = noMigration
			scaffoldGen.CRUD = crud

			if err := scaffoldGen.GenerateScaffold(name, fieldSpecs); err != nil {
				return fmt.Errorf("failed to generate scaffold: %w", err)
			}

			return nil
		}),
	}

	cmd.Flags().BoolVar(&noService, "no-service", false, "Skip service generation")
	cmd.Flags().BoolVar(&noHandler, "no-handler", false, "Skip handler generation")
	cmd.Flags().BoolVar(&noMigration, "no-migration", false, "Skip migration generation")
	cmd.Flags().BoolVar(&crud, "crud", false, "Generate CRUD operations")

	return cmd
}
