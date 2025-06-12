package generatecmd

import (
	"fmt"

	"github.com/alexisvisco/goframe/cli/generators"
	"github.com/alexisvisco/goframe/cli/generators/genhelper"
	"github.com/alexisvisco/goframe/cli/generators/gentask"
	"github.com/spf13/cobra"
)

func taskCmd() *cobra.Command {
	var flagDescription string
	cmd := &cobra.Command{
		Use:   "task <name>",
		Short: "GenerateHandler a new task file",
		Long:  "GenerateHandler a new task file with the specified name.",
		RunE: genhelper.WithFileDiff(func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("task name is required")
			}
			name := args[0]
			g := generators.Generator{GoModuleName: cmd.Context().Value("module").(string)}
			taskGen := gentask.TaskGenerator{Gen: &g}
			if err := taskGen.GenerateTask(name, flagDescription); err != nil {
				return fmt.Errorf("failed to create task: %w", err)
			}
			return nil
		}),
	}

	cmd.Flags().StringVarP(&flagDescription, "description", "d", "", "Description of the task")

	return cmd
}
