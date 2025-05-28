package fxutil

import (
	"log/slog"

	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
)

type SlogLogger struct{}

var _ fxevent.Logger = (*SlogLogger)(nil)

func (s *SlogLogger) LogEvent(event fxevent.Event) {
	switch e := event.(type) {
	case *fxevent.OnStartExecuting:
		// Silence start executing logs
	case *fxevent.OnStopExecuting:
		// Silence stop executing logs
	case *fxevent.Supplied:
		// Silence supplied logs
	case *fxevent.Provided:
		// Silence provided logs
	case *fxevent.Decorated:
		// Silence decorated logs
	case *fxevent.Invoked:
		// Only log if there was an error
		if e.Err != nil {
			slog.Error("invoke failed", slog.Any("error", e.Err))
		}
	case *fxevent.Started:
		// Only log if there was an error
		if e.Err != nil {
			slog.Error("start failed", slog.Any("error", e.Err))
		}
	case *fxevent.Stopped:
		// Only log if there was an error
		if e.Err != nil {
			slog.Error("stop failed", slog.Any("error", e.Err))
		}
	}
}

func Logger() fx.Option {
	return fx.WithLogger(func() fxevent.Logger {
		return &SlogLogger{}
	})
}
