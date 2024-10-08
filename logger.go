package pgxslog

import (
	"context"
	"log/slog"
	"reflect"
	"regexp"
	"runtime"
	"time"

	"github.com/jackc/pgx/v5/tracelog"
)

var _ tracelog.Logger = (*Logger)(nil)

// Logger is a tracelog.Logger that logs to the given logger.
type Logger struct{}

// Log implements tracelog.Logger.
func (x *Logger) Log(ctx context.Context, level tracelog.LogLevel, msg string, data map[string]any) {
	var pcs [1]uintptr

	// prepare the trace
	switch msg {
	case "Query":
		runtime.Callers(7, pcs[:])
	case "BatchQuery", "BatchClose":
		runtime.Callers(6, pcs[:])
	default:
		runtime.Callers(4, pcs[:])
	}

	// create the record
	record := slog.NewRecord(time.Now(), slog.LevelDebug, msg, pcs[0])
	// prepare the record
	switch level {
	case tracelog.LogLevelDebug:
		record.Level = slog.LevelDebug
	case tracelog.LogLevelInfo:
		record.Level = slog.LevelInfo
	case tracelog.LogLevelWarn:
		record.Level = slog.LevelWarn
	case tracelog.LogLevelError:
		record.Level = slog.LevelError
	default:
		record.Level = slog.LevelDebug - slog.Level(level)
		record.AddAttrs(slog.Any("PGX_LOG_LEVEL", level))
	}

	// add the attributes
	for k, v := range data {
		switch k {
		case "sql":
			if value, ok := v.(string); ok {
				if match := pattern.FindStringSubmatch(value); len(match) == 2 {
					record.AddAttrs(slog.Any("sql_operation", match[1]))
				}
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

		record.AddAttrs(slog.Any(k, v))
	}

	logger := FromContext(ctx)
	logger.Handler().Handle(ctx, record)
}

var pattern = regexp.MustCompile(`^--\s+name:\s+(\w+)`)
