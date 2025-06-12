package generatecmd

import (
	"fmt"

	"github.com/alexisvisco/goframe/cli/generators"
	"github.com/alexisvisco/goframe/cli/generators/genhelper"
	"github.com/alexisvisco/goframe/cli/generators/genworker"
	"github.com/alexisvisco/goframe/core/configuration"
	"github.com/spf13/cobra"
)

func workerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "worker <subcommand> [flags]",
		Short: "WriteTo worker related files",
	}

	cmd.AddCommand(workerWorkflowCmd())
	cmd.AddCommand(workerActivityCmd())

	return cmd
}

func workerWorkflowCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "workflow <name> [activities...]",
		Aliases: []string{"wf"},
		Short:   "GenerateHandler a new workflow",
		RunE: genhelper.WithFileDiff(func(cmd *cobra.Command, args []string) error {
			cfg, ok := cmd.Context().Value("config.worker").(configuration.Worker)
			if !ok || cfg.Type != configuration.WorkerTypeTemporal {
				return fmt.Errorf("only available for temporal workers, got %v", cfg.Type)
			}

			if len(args) < 1 {
				return fmt.Errorf("workflow name is required")
			}
			name := args[0]
			var acts []string
			if len(args) > 1 {
				acts = args[1:]
			}

			g := generators.Generator{GoModuleName: cmd.Context().Value("module").(string)}
			genWorker := genworker.WorkerGenerator{Gen: &g}
			if err := genWorker.GenerateWorkflow(name, acts); err != nil {
				return fmt.Errorf("failed to create workflow: %w", err)
			}
			return nil
		}),
	}
}

func workerActivityCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "activity <name>",
		Short: "GenerateHandler a new activity",
		RunE: genhelper.WithFileDiff(func(cmd *cobra.Command, args []string) error {
			cfg, ok := cmd.Context().Value("config.worker").(configuration.Worker)
			if !ok || cfg.Type != configuration.WorkerTypeTemporal {
				return fmt.Errorf("only available for temporal workers, got %v", cfg.Type)
			}
			if len(args) < 1 {
				return fmt.Errorf("activity name is required")
			}
			name := args[0]
			g := generators.Generator{GoModuleName: cmd.Context().Value("module").(string)}
			genWorker := genworker.WorkerGenerator{Gen: &g}

			if err := genWorker.GenerateActivity(name); err != nil {
				return fmt.Errorf("failed to create activity: %w", err)
			}
			return nil
		}),
	}
}
