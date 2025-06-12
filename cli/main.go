package main

import (
	"context"
	"os"

	"github.com/alexisvisco/goframe/cli/commands/createcmd"
	"github.com/spf13/cobra"
)

var (
	exitOK    = 0
	exitError = 1
)

func NewCmdRoot() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "<command> <subcommand> [flags]",
		Short:         "Goframe CLI",
		Long:          `GenerateHandler a new Goframe project`,
		SilenceErrors: true,
		Annotations: map[string]string{
			"versionInfo": "0.0.1",
		},
		SilenceUsage: true,
		Version:      "0.0.1",
	}

	cmd.AddCommand(createcmd.NewInitCmd())

	return cmd
}

func main() {
	os.Exit(run())
}

func run() int {
	ctx, ctxCancel := context.WithCancel(context.Background())
	defer ctxCancel()

	cmdRoot := NewCmdRoot()
	if _, err := cmdRoot.ExecuteContextC(ctx); err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		switch {
		default:
			return exitError
		}
	}

	return exitOK
}
