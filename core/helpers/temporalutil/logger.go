package temporalutil

import (
	"log/slog"
	"os"
)

/*
Logger interface {
		Debug(msg string, keyvals ...interface{})
		Info(msg string, keyvals ...interface{})
		Warn(msg string, keyvals ...interface{})
		Error(msg string, keyvals ...interface{})
	}
*/

// Logger is a simple logger implementation for use with Temporal workflows and activities.
// It uses slog.Logger for logging messages at different levels.
type Logger struct {
	logger *slog.Logger
}

// NewLogger creates a new Logger instance with default configuration
func NewLogger() *Logger {
	return &Logger{
		logger: slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		})),
	}
}

// NewLoggerWithHandler creates a new Logger instance with a custom slog handler
func NewLoggerWithHandler(handler slog.Handler) *Logger {
	return &Logger{
		logger: slog.New(handler),
	}
}

// NewLoggerWithSlog creates a new Logger instance using an existing slog.Logger
func NewLoggerWithSlog(logger *slog.Logger) *Logger {
	return &Logger{
		logger: logger,
	}
}

// Debug logs a debug message with optional key-value pairs
func (l *Logger) Debug(msg string, keyvals ...interface{}) {
	l.logger.Debug(msg, keyvals...)
}

// Info logs an info message with optional key-value pairs
func (l *Logger) Info(msg string, keyvals ...interface{}) {
	l.logger.Info(msg, keyvals...)
}

// Warn logs a warning message with optional key-value pairs
func (l *Logger) Warn(msg string, keyvals ...interface{}) {
	l.logger.Warn(msg, keyvals...)
}

// Error logs an error message with optional key-value pairs
func (l *Logger) Error(msg string, keyvals ...interface{}) {
	l.logger.Error(msg, keyvals...)
}
