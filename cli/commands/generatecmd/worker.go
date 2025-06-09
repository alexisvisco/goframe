package generatecmd

import (
	"fmt"

	"github.com/alexisvisco/goframe/cli/generators"
	"github.com/alexisvisco/goframe/core/configuration"
	"github.com/spf13/cobra"
)

func workerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "worker <subcommand> [flags]",
		Short: "Generate worker related files",
	}

	cmd.AddCommand(workerWorkflowCmd())
	cmd.AddCommand(workerActivityCmd())

	return cmd
}

func workerWorkflowCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "workflow <name> [activities...]",
		Aliases: []string{"wf"},
		Short:   "Create a new workflow",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, ok := cmd.Context().Value("config.worker").(configuration.Worker)
			if !ok || cfg.Type != configuration.WorkerTypeTemporal {
				return fmt.Errorf("only available for temporal workers, got %v", cfg.Type)
			}

			if len(args) < 1 {
				return fmt.Errorf("workflow name is required")
			}
			name := args[0]
			acts := []string{}
			if len(args) > 1 {
				acts = args[1:]
			}

			g := generators.Generator{GoModuleName: cmd.Context().Value("module").(string)}
			wg := g.Worker()
			if err := wg.CreateWorkflow(name, acts); err != nil {
				return fmt.Errorf("failed to create workflow: %w", err)
			}
			if err := wg.UpdateOrCreateRegistrations(); err != nil {
				return fmt.Errorf("failed to update registrations: %w", err)
			}
			return nil
		},
	}
}

func workerActivityCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "activity <name>",
		Short: "Create a new activity",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, ok := cmd.Context().Value("config.worker").(configuration.Worker)
			if !ok || cfg.Type != configuration.WorkerTypeTemporal {
				return fmt.Errorf("only available for temporal workers, got %v", cfg.Type)
			}
			if len(args) < 1 {
				return fmt.Errorf("activity name is required")
			}
			name := args[0]
			g := generators.Generator{GoModuleName: cmd.Context().Value("module").(string)}
			wg := g.Worker()
			if err := wg.CreateActivity(name); err != nil {
				return fmt.Errorf("failed to create activity: %w", err)
			}
			if err := wg.UpdateOrCreateRegistrations(); err != nil {
				return fmt.Errorf("failed to update registrations: %w", err)
			}
			return nil
		},
	}
}
