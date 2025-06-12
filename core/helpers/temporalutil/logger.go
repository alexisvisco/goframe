package temporalutil

import (
	"log/slog"
	"os"

	"github.com/alexisvisco/goframe/core/helpers/str"
)

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
	transformedKeyvals := transformKeyvals(keyvals...)
	l.logger.Debug(msg, transformedKeyvals...)
}

// Info logs an info message with optional key-value pairs
func (l *Logger) Info(msg string, keyvals ...interface{}) {
	transformedKeyvals := transformKeyvals(keyvals...)
	l.logger.Info(msg, transformedKeyvals...)
}

// Warn logs a warning message with optional key-value pairs
func (l *Logger) Warn(msg string, keyvals ...interface{}) {
	transformedKeyvals := transformKeyvals(keyvals...)
	l.logger.Warn(msg, transformedKeyvals...)
}

// Error logs an error message with optional key-value pairs
func (l *Logger) Error(msg string, keyvals ...interface{}) {
	transformedKeyvals := transformKeyvals(keyvals...)
	l.logger.Error(msg, transformedKeyvals...)
}

// transformKeyvals converts string keys to snake_case using str.ToSnakeCase
func transformKeyvals(keyvals ...interface{}) []interface{} {
	result := make([]interface{}, len(keyvals))

	for i := 0; i < len(keyvals); i++ {
		// If this is a key (even index) and it's a string, convert it to snake_case
		if i%2 == 0 && i+1 < len(keyvals) {
			if key, ok := keyvals[i].(string); ok {
				result[i] = str.ToSnakeCase(key)
			} else {
				result[i] = keyvals[i]
			}
		} else {
			result[i] = keyvals[i]
		}
	}

	return result
}
