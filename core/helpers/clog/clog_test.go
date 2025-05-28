package clog

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCanonicalLogger(t *testing.T) {
	t.Run("basic logging flow", func(t *testing.T) {
		// Setup
		var buf bytes.Buffer
		logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))

		// Simulate request flow
		ctx := context.Background()
		ctx, line := FromContext(ctx)
		line.Attach(logger).
			Add("request_id", "123").
			Add("method", "POST")

		// Business logic with error check
		err := doSomething()
		if err != nil {
			line.Error(err).
				Add("step", "business_logic").
				Log("business logic failed")
		}

		// Final log
		line.Add("status", 200).
			Add("duration_ms", 150).
			Log("request completed")

		// Verify output
		var logEntry map[string]interface{}
		require.NoError(t, json.Unmarshal(buf.Bytes(), &logEntry))

		assert.Equal(t, "123", logEntry["request_id"])
		assert.Equal(t, "POST", logEntry["method"])
		assert.Equal(t, float64(200), logEntry["status"])
		assert.Equal(t, float64(150), logEntry["duration_ms"])
		assert.Equal(t, "request completed", logEntry["msg"])
	})

	t.Run("error handling", func(t *testing.T) {
		var buf bytes.Buffer
		logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))

		_, line := FromContext(context.Background())
		line.Attach(logger)

		// Simulate error
		err := errors.New("something went wrong")
		line.Error(err).Log("operation failed")

		var logEntry map[string]interface{}
		require.NoError(t, json.Unmarshal(buf.Bytes(), &logEntry))

		assert.Equal(t, "something went wrong", logEntry["error"])
		assert.Equal(t, "ERROR", logEntry["level"])
		assert.Equal(t, "operation failed", logEntry["msg"])
	})

	t.Run("context handling", func(t *testing.T) {
		var buf bytes.Buffer
		logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))

		// Test existing context
		ctx := context.Background()
		ctx1, line1 := FromContext(ctx)
		line1.Add("test", "value").Attach(logger)

		_, line2 := FromContext(ctx1)
		line2.Add("valo", "truc")

		assert.Same(t, line1, line2, "Expected same line instance from same context")

		line2.Log("test message")

		// Verify output
		var logEntry map[string]interface{}
		require.NoError(t, json.Unmarshal(buf.Bytes(), &logEntry))

		assert.Equal(t, "value", logEntry["test"])
		assert.Equal(t, "truc", logEntry["valo"])

		// Test new context
		_, line3 := FromContext(context.Background())
		assert.NotSame(t, line1, line3, "Expected different line instance from new context")

	})

	t.Run("structured attributes", func(t *testing.T) {
		var buf bytes.Buffer
		logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))

		_, line := FromContext(context.Background())
		line.Attach(logger).
			Attr(slog.Group("request",
				slog.String("path", "/api/v1"),
				slog.Int("port", 8080),
			)).
			Log("structured log test")

		var logEntry map[string]interface{}
		require.NoError(t, json.Unmarshal(buf.Bytes(), &logEntry))

		requestData, ok := logEntry["request"].(map[string]interface{})
		require.True(t, ok, "Missing request structure in log")

		assert.Equal(t, "/api/v1", requestData["path"])
		assert.Equal(t, float64(8080), requestData["port"])
		assert.Equal(t, "structured log test", logEntry["msg"])
	})

	t.Run("clone functionality", func(t *testing.T) {
		var buf bytes.Buffer
		logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))

		_, line := FromContext(context.Background())
		line.Attach(logger).
			Add("original", "value")

		// Clone and modify
		clonedLine := line.Clone()
		clonedLine.Add("cloned", "value").Log("cloned line")

		// Original line
		line.Log("original line")

		logs := bytes.Split(buf.Bytes(), []byte("\n"))
		require.Equal(t, 3, len(logs), "Expected 2 log entries plus empty line") // 2 logs + empty line

		var clonedEntry, originalEntry map[string]interface{}
		require.NoError(t, json.Unmarshal(logs[0], &clonedEntry))
		require.NoError(t, json.Unmarshal(logs[1], &originalEntry))

		assert.Equal(t, "value", clonedEntry["original"])
		assert.Equal(t, "value", clonedEntry["cloned"])
		assert.Equal(t, "cloned line", clonedEntry["msg"])

		assert.Equal(t, "value", originalEntry["original"])
		assert.NotContains(t, originalEntry, "cloned")
		assert.Equal(t, "original line", originalEntry["msg"])
	})

	t.Run("default logger fallback", func(t *testing.T) {
		_, line := FromContext(context.Background())

		// This should not panic even without an attached logger
		assert.NotPanics(t, func() {
			line.Add("test", "value").Log("using default logger")
		})
	})
}

// Helper function
func doSomething() error {
	return nil
}
