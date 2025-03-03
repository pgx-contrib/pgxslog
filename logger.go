package pgxslog

import (
	"bufio"
	"context"
	"log/slog"
	"reflect"
	"regexp"
	"runtime"
	"strings"
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

				// overwrite the value
				v = TrimQuery(value)
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

	logger := x.logger(ctx)
	logger.LogAttrs(ctx, level, message, attr)
}

func (x *Logger) logger(ctx context.Context) *slog.Logger {
	if key := x.ContextKey; key == nil {
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
	reader := strings.NewReader(key)
	scanner := bufio.NewScanner(reader)
	scanner.Split(bufio.ScanRunes)

	builder := &strings.Builder{}
	// scan the query and fill the builder
	for scanner.Scan() {
		for _, ch := range scanner.Text() {
			if unicode.IsSpace(ch) {
				if builder.Len() > 0 {
					builder.WriteString("_")
				}
				continue
			}

			if unicode.IsUpper(ch) {
				if builder.Len() > 0 {
					builder.WriteString("_")
				}
			}

			ch = unicode.ToLower(ch)
			// write the character
			builder.WriteRune(ch)
		}
	}

	key = builder.String()
	// create teh attribute
	return slog.Any(key, value)
}

// TrimQuery trims the query by removing the comments.
func TrimQuery(query string) string {
	reader := strings.NewReader(query)
	scanner := bufio.NewScanner(reader)

	builder := &strings.Builder{}
	// scan the query and fill the builder
	for scanner.Scan() {
		text := scanner.Text()
		text = strings.TrimSpace(text)

		index := strings.Index(text, "--")

		if index == 0 {
			continue
		}

		if index > 0 {
			text = text[:index]
		}

		text = strings.TrimSpace(text)

		if builder.Len() > 0 {
			builder.WriteString(" ")
		}

		builder.WriteString(text)
	}

	return builder.String()
}
