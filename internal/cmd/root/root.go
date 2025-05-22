package root

import (
	"github.com/alexisvisco/goframe/internal/cmd/create"
	"github.com/spf13/cobra"
)

func NewCmdRoot() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "<command> <subcommand> [flags]",
		Short:         "Goframe CLI",
		Long:          `Create a new Goframe project`,
		SilenceErrors: true,
		Annotations: map[string]string{
			"versionInfo": "0.0.1",
		},
		SilenceUsage: true,
		Version:      "0.0.1",
	}

	cmd.AddCommand(create.NewInitCmd())

	return cmd
}
