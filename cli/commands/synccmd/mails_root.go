package synccmd

import (
	"github.com/spf13/cobra"
)

func NewCmdSync(subCommands ...*cobra.Command) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync <subcommand>",
		Short: "Synchronize code (registries, router, mails...)",
	}
	cmd.AddCommand(syncMails())
	cmd.AddCommand(syncRouter())
	for _, sc := range subCommands {
		cmd.AddCommand(sc)
	}
	return cmd
}
