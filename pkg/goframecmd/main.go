package goframecmd

import (
	"context"
	"errors"
)

type ExitCode int

const (
	exitOK      ExitCode = 0
	exitError   ExitCode = 1
	exitCancel  ExitCode = 2
	exitAuth    ExitCode = 4
	exitPending ExitCode = 8
)

func Main() ExitCode {

	ctx, ctxCancel := context.WithCancel(context.Background())
	defer ctxCancel()

	cmdRoot := root.NewCmdRoot()
	if _, err := cmdRoot.ExecuteContextC(ctx); err != nil {
		switch {
		case errors.Is(err, baepoerrors.AuthError):
			return exitAuth
		default:
			return exitError
		}
	}

	return exitOK
}
