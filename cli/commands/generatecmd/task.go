package generatecmd

import (
	"fmt"

	"github.com/alexisvisco/goframe/cli/generators"
	"github.com/spf13/cobra"
)

func taskCmd() *cobra.Command {
	var flagDescription string
	cmd := &cobra.Command{
		Use:   "task <name>",
		Short: "Create a new task file",
		Long:  "Create a new task file with the specified name.",
		RunE: withFileDiff(func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("task name is required")
			}
			name := args[0]

			g := generators.Generator{GoModuleName: cmd.Context().Value("module").(string)}
			if err := g.Task().Create(name, flagDescription); err != nil {
				return fmt.Errorf("failed to create task: %w", err)
			}
			return nil
		}),
	}

	cmd.Flags().StringVarP(&flagDescription, "description", "d", "", "Description of the task")

	return cmd
}
