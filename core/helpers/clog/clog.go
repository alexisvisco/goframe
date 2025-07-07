package clog

import (
	"context"
	"log/slog"
	"sync"
)

type contextKey struct{}

type Line struct {
	mu     sync.RWMutex
	attrs  []slog.Attr
	logger *slog.Logger
	level  slog.Level
}

// FromContext retrieves a Line from context or creates a new one
// Returns the potentially updated context and the Line
func FromContext(ctx context.Context) (context.Context, *Line) {
	if line, ok := ctx.Value(contextKey{}).(*Line); ok {
		return ctx, line
	}
	line := &Line{
		attrs: make([]slog.Attr, 0),
		level: slog.LevelInfo,
	}
	return context.WithValue(ctx, contextKey{}, line), line
}

func (l *Line) Attach(logger *slog.Logger) *Line {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.logger = logger
	return l
}

func (l *Line) Add(key string, value any) *Line {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.attrs = append(l.attrs, slog.Any(key, value))
	return l
}

func (l *Line) Attr(attr slog.Attr) *Line {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.attrs = append(l.attrs, attr)
	return l
}

func (l *Line) Error(err error) *Line {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.level = slog.LevelError

	l.attrs = append(l.attrs, slog.String("error", err.Error()))

	return l
}

func (l *Line) Clone() *Line {
	l.mu.RLock()
	defer l.mu.RUnlock()

	newLine := &Line{
		logger: l.logger,
		level:  l.level,
		attrs:  make([]slog.Attr, len(l.attrs)),
	}
	copy(newLine.attrs, l.attrs)
	return newLine
}

func (l *Line) Log(msg string) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	logger := l.logger
	if logger == nil {
		logger = slog.Default()
	}

	logger.LogAttrs(context.Background(), l.level, msg, l.attrs...)
}

func Add(ctx context.Context, keyVals ...any) {
	_, line := FromContext(ctx)
	if line == nil {
		slog.Error("clog: no line found in context, cannot add attributes")
		return
	}

	for i := 0; i < len(keyVals); i += 2 {
		if i+1 < len(keyVals) {
			line.Add(keyVals[i].(string), keyVals[i+1])
		} else {
			line.Add(keyVals[i].(string), nil)
		}
	}
}
