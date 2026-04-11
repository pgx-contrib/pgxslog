package pgxslog

import (
	"context"
	"log/slog"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"time"
	"unicode"

	"github.com/jackc/pgx/v5/tracelog"
)

// NameRegexp is a regular expression to extract the operation name from a SQL query.
var NameRegexp = regexp.MustCompile(`^--\s+name:\s+(\w+)`)

var _ tracelog.Logger = (*Logger)(nil)

// Logger is a tracelog.Logger that logs to the given logger.
type Logger struct {
	// ContextKey is the context key of the logger.
	ContextKey any
}

// Log implements tracelog.Logger.
func (x *Logger) Log(ctx context.Context, severity tracelog.LogLevel, message string, data map[string]any) {
	var pcs [1]uintptr

	// prepare the trace
	switch message {
	case "Query":
		runtime.Callers(7, pcs[:])
	case "BatchQuery", "BatchClose":
		runtime.Callers(6, pcs[:])
	default:
		runtime.Callers(4, pcs[:])
	}

	// map the level
	level := ConvertSeverity(severity)

	logger := x.logger(ctx)
	if !logger.Enabled(ctx, level) {
		return
	}

	// prepare the attributes
	var attrs []slog.Attr
	// add the severity if it's not mapped
	if level < slog.LevelDebug {
		attrs = append(attrs, slog.Any("PGX_LOG_LEVEL", severity))
	}

	// add the attributes
	for k, v := range data {
		switch k {
		case "sql":
			if value, ok := v.(string); ok {
				if match := NameRegexp.FindStringSubmatch(value); len(match) == 2 {
					attrs = append(attrs, slog.Any("name", match[1]))
				}
			}
		case "args":
			if args, ok := v.([]any); ok {
				// override the value
				v = ConvertArgs(args)
			}
		}

		attrs = append(attrs, ConvertAttr(k, v))
	}

	attr := slog.Attr{
		Key:   "query",
		Value: slog.GroupValue(attrs...),
	}

	record := slog.NewRecord(time.Now(), level, message, pcs[0])
	record.AddAttrs(attr)
	_ = logger.Handler().Handle(ctx, record)
}

func (x *Logger) logger(ctx context.Context) *slog.Logger {
	if key := x.ContextKey; key != nil {
		if logger, ok := ctx.Value(key).(*slog.Logger); ok {
			return logger
		}
	}

	return slog.Default()
}

// ConvertSeverity converts the severity to a slog.Level.
func ConvertSeverity(severity tracelog.LogLevel) slog.Level {
	level := slog.LevelDebug - slog.Level(severity)

	// prepare the record
	switch severity {
	case tracelog.LogLevelDebug:
		level = slog.LevelDebug
	case tracelog.LogLevelInfo:
		level = slog.LevelInfo
	case tracelog.LogLevelWarn:
		level = slog.LevelWarn
	case tracelog.LogLevelError:
		level = slog.LevelError
	}

	return level
}

// ConvertArgs converts the arguments to a collection of interfaces.
func ConvertArgs(args []any) []any {
	var vc []any

	for _, value := range args {
		// reflect the parameter
		vref := reflect.ValueOf(value)
		vref = reflect.Indirect(vref)
		// prepare the collection
		switch {
		case !vref.IsValid():
			vc = append(vc, value)
		case !vref.CanInterface():
			vc = append(vc, value)
		default:
			vc = append(vc, vref.Interface())
		}
	}

	return vc
}

// ConvertAttr converts the key and value to an attribute.
func ConvertAttr(key string, value any) slog.Attr {
	runes := []rune(key)
	builder := &strings.Builder{}

	for i, ch := range runes {
		if unicode.IsSpace(ch) {
			if builder.Len() > 0 {
				builder.WriteRune('_')
			}
			continue
		}

		if unicode.IsUpper(ch) && builder.Len() > 0 {
			prev := runes[i-1]
			var next rune
			if i+1 < len(runes) {
				next = runes[i+1]
			}
			// insert underscore when transitioning from lower→upper,
			// or at the end of an uppercase run before a lowercase letter
			// (e.g. "SQLQuery" → "sql_query", not "s_q_l_query")
			if unicode.IsLower(prev) || (unicode.IsUpper(prev) && unicode.IsLower(next)) {
				builder.WriteRune('_')
			}
		}

		builder.WriteRune(unicode.ToLower(ch))
	}

	// create the attribute
	return slog.Any(builder.String(), value)
}

// ContextKey represents a context key.
type ContextKey struct {
	name string
}

// String returns the context key as a string.
func (k *ContextKey) String() string {
	return k.name
}

// LoggerKey represents the context key of the logger.
var LoggerKey = &ContextKey{
	name: reflect.TypeOf(ContextKey{}).PkgPath() + "Logger",
}
