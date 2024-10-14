package pgxslog

import (
	"bufio"
	"bytes"
	"context"
	"log/slog"
	"reflect"
	"regexp"
	"runtime"
	"strings"

	"github.com/jackc/pgx/v5/tracelog"
)

// NameRegexp is a regular expression to extract the operation name from a SQL query.
var NameRegexp = regexp.MustCompile(`^--\s+name:\s+(\w+)`)

var _ tracelog.Logger = (*Logger)(nil)

// Logger is a tracelog.Logger that logs to the given logger.
type Logger struct {
	// FromContext is a function that returns the logger from the context.
	FromContext func(ctx context.Context) *slog.Logger
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

	var attrs []slog.Attr
	// map the level
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
	default:
		attrs = append(attrs, slog.Any("PGX_LOG_LEVEL", severity))
	}

	// add the attributes
	for k, v := range data {
		switch k {
		case "sql":
			if value, ok := v.(string); ok {
				if match := NameRegexp.FindStringSubmatch(value); len(match) == 2 {
					attrs = append(attrs, slog.Any("sql_operation", match[1]))
				}
				// overwrite the value
				v = TrimQuery(value)
			}
		case "args":
			if args, ok := v.([]any); ok {
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
				// override the value
				v = vc
			}
		}

		attrs = append(attrs, slog.Any(k, v))
	}

	attr := slog.Attr{
		Key:   "pgx",
		Value: slog.GroupValue(attrs...),
	}

	logger := x.logger(ctx)
	logger.LogAttrs(ctx, level, message, attr)
}

func (x *Logger) logger(ctx context.Context) *slog.Logger {
	if x.FromContext != nil {
		return x.FromContext(ctx)
	}

	return FromContext(ctx)
}

// TrimQuery trims the query by removing the comments.
func TrimQuery(query string) string {
	writer := &bytes.Buffer{}
	reader := strings.NewReader(query)
	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		text := scanner.Text()
		text = strings.TrimSpace(text)
		if strings.HasPrefix(text, "--") {
			continue
		}
		writer.WriteString(text)
		writer.WriteString("\n")
	}

	return writer.String()
}
