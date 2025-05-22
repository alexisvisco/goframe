package main

import (
	"context"
	"os"

	"github.com/alexisvisco/goframe/internal/cmd/root"
)

var (
	exitOK    = 0
	exitError = 1
)

func main() {
	os.Exit(run())
}

func run() int {
	ctx, ctxCancel := context.WithCancel(context.Background())
	defer ctxCancel()

	cmdRoot := root.NewCmdRoot()
	if _, err := cmdRoot.ExecuteContextC(ctx); err != nil {
		os.Stderr.WriteString(err.Error() + "\n")
		switch {
		default:
			return exitError
		}
	}

	return exitOK
}
