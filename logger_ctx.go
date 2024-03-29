package pgxslog

import (
	"context"
	"log/slog"
	"reflect"
)

// LoggerKey represents the context key of the logger.
var LoggerKey = &ContextKey{
	name: reflect.TypeOf(ContextKey{}).PkgPath(),
}

// ContextKey represents a context key.
type ContextKey struct {
	name string
}

// String returns the context key as a string.
func (k *ContextKey) String() string {
	return k.name
}

// FromContext returns the logger from a given context.
func FromContext(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(LoggerKey).(*slog.Logger); ok {
		return logger
	}

	return slog.Default()
}

// WithContext provides the logger in a given context.
func WithContext(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, LoggerKey, logger)
}
