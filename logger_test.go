package pgxslog_test

import (
	"context"
	"log/slog"
	"sync"

	"github.com/jackc/pgx/v5/tracelog"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/pgx-contrib/pgxslog"
)

// testHandler captures log records for assertions.
type testHandler struct {
	mu      sync.Mutex
	records []slog.Record
}

func (h *testHandler) Enabled(_ context.Context, _ slog.Level) bool { return true }

func (h *testHandler) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.records = append(h.records, r.Clone())
	return nil
}

func (h *testHandler) WithAttrs([]slog.Attr) slog.Handler { return h }
func (h *testHandler) WithGroup(string) slog.Handler      { return h }

func (h *testHandler) Reset() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.records = h.records[:0]
}

// disabledAtDebugHandler rejects messages at or below Debug level.
type disabledAtDebugHandler struct {
	testHandler
}

func (h *disabledAtDebugHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level > slog.LevelDebug
}

// findGroupAttr walks the top-level attrs of a record for a group with the given key.
func findGroupAttr(r slog.Record, key string) []slog.Attr {
	var result []slog.Attr
	r.Attrs(func(a slog.Attr) bool {
		if a.Key == key {
			result = a.Value.Group()
			return false
		}
		return true
	})
	return result
}

// findAttrInGroup finds a named attr within a slice of attrs.
func findAttrInGroup(attrs []slog.Attr, key string) *slog.Attr {
	for i := range attrs {
		if attrs[i].Key == key {
			cp := attrs[i]
			return &cp
		}
	}
	return nil
}

var _ = Describe("pgxslog unit tests", func() {

	// -----------------------------------------------------------------------
	Describe("ConvertSeverity()", func() {
		DescribeTable("maps pgx levels to slog levels",
			func(severity tracelog.LogLevel, expected slog.Level) {
				Expect(pgxslog.ConvertSeverity(severity)).To(Equal(expected))
			},
			Entry("Error", tracelog.LogLevelError, slog.LevelError),
			Entry("Warn", tracelog.LogLevelWarn, slog.LevelWarn),
			Entry("Info", tracelog.LogLevelInfo, slog.LevelInfo),
			Entry("Debug", tracelog.LogLevelDebug, slog.LevelDebug),
			Entry("Trace", tracelog.LogLevelTrace, slog.LevelDebug-slog.Level(tracelog.LogLevelTrace)),
			Entry("None", tracelog.LogLevelNone, slog.LevelDebug-slog.Level(tracelog.LogLevelNone)),
		)
	})

	// -----------------------------------------------------------------------
	Describe("ConvertArgs()", func() {
		It("dereferences a *int pointer to its raw int value", func() {
			n := 42
			result := pgxslog.ConvertArgs([]any{&n})
			Expect(result).To(HaveLen(1))
			Expect(result[0]).To(Equal(42))
		})

		It("passes through a nil pointer unchanged", func() {
			var p *int
			result := pgxslog.ConvertArgs([]any{p})
			Expect(result).To(HaveLen(1))
			Expect(result[0]).To(Equal(p))
		})

		It("leaves non-pointer values unchanged", func() {
			result := pgxslog.ConvertArgs([]any{"hello", 42, true})
			Expect(result).To(ConsistOf("hello", 42, true))
		})

		It("returns nil for an empty slice", func() {
			result := pgxslog.ConvertArgs([]any{})
			Expect(result).To(BeNil())
		})
	})

	// -----------------------------------------------------------------------
	Describe("ConvertAttr()", func() {
		DescribeTable("converts key to snake_case and preserves value",
			func(key, expectedKey string) {
				val := "test-value"
				attr := pgxslog.ConvertAttr(key, val)
				Expect(attr.Key).To(Equal(expectedKey))
				Expect(attr.Value.String()).To(Equal(val))
			},
			Entry("all lowercase", "sql", "sql"),
			Entry("camelCase", "rowsAffected", "rows_affected"),
			Entry("PascalCase", "Time", "time"),
			Entry("space separator", "row count", "row_count"),
			Entry("multi-word", "BatchSize", "batch_size"),
			Entry("all-caps acronym", "SQL", "sql"),
			Entry("acronym prefix", "SQLQuery", "sql_query"),
			Entry("acronym suffix", "querySQL", "query_sql"),
		)
	})

	// -----------------------------------------------------------------------
	Describe("LoggerKey", func() {
		It("is non-nil", func() {
			Expect(pgxslog.LoggerKey).NotTo(BeNil())
		})

		It("String() is non-empty and contains 'pgxslog'", func() {
			s := pgxslog.LoggerKey.String()
			Expect(s).NotTo(BeEmpty())
			Expect(s).To(ContainSubstring("pgxslog"))
		})
	})

	// -----------------------------------------------------------------------
	Describe("NameRegexp", func() {
		DescribeTable("matches -- name: comments only at query start",
			func(input string, matchExpected bool, expectedName string) {
				match := pgxslog.NameRegexp.FindStringSubmatch(input)
				if matchExpected {
					Expect(match).To(HaveLen(2))
					Expect(match[1]).To(Equal(expectedName))
				} else {
					Expect(match).To(BeEmpty())
				}
			},
			Entry("standard", "-- name: FindUser\nSELECT 1", true, "FindUser"),
			Entry("no prefix", "SELECT 1", false, ""),
			Entry("empty", "", false, ""),
			Entry("mid-query", "SELECT 1\n-- name: X", false, ""),
		)
	})

	// -----------------------------------------------------------------------
	Describe("Logger.Log()", func() {
		var (
			handler  *testHandler
			logger   *pgxslog.Logger
			original *slog.Logger
		)

		BeforeEach(func() {
			original = slog.Default()
			handler = &testHandler{}
			slog.SetDefault(slog.New(handler))
			logger = &pgxslog.Logger{}
		})

		AfterEach(func() {
			slog.SetDefault(original)
		})

		Describe("level mapping", func() {
			DescribeTable("each severity emits the correct slog level",
				func(severity tracelog.LogLevel, expected slog.Level) {
					logger.Log(context.Background(), severity, "test", nil)
					Expect(handler.records).To(HaveLen(1))
					Expect(handler.records[0].Level).To(Equal(expected))
				},
				Entry("Error", tracelog.LogLevelError, slog.LevelError),
				Entry("Warn", tracelog.LogLevelWarn, slog.LevelWarn),
				Entry("Info", tracelog.LogLevelInfo, slog.LevelInfo),
				Entry("Debug", tracelog.LogLevelDebug, slog.LevelDebug),
			)
		})

		It("sets record.Message to the passed message", func() {
			logger.Log(context.Background(), tracelog.LogLevelInfo, "hello world", nil)
			Expect(handler.records).To(HaveLen(1))
			Expect(handler.records[0].Message).To(Equal("hello world"))
		})

		It("wraps data attrs inside a 'query' group", func() {
			logger.Log(context.Background(), tracelog.LogLevelInfo, "Query", map[string]any{
				"sql": "SELECT 1",
			})
			Expect(handler.records).To(HaveLen(1))
			group := findGroupAttr(handler.records[0], "query")
			Expect(group).NotTo(BeEmpty())
		})

		It("adds 'name' attr when sql starts with -- name:", func() {
			logger.Log(context.Background(), tracelog.LogLevelInfo, "Query", map[string]any{
				"sql": "-- name: FindUser\nSELECT 1",
			})
			Expect(handler.records).To(HaveLen(1))
			group := findGroupAttr(handler.records[0], "query")
			nameAttr := findAttrInGroup(group, "name")
			Expect(nameAttr).NotTo(BeNil())
			Expect(nameAttr.Value.String()).To(Equal("FindUser"))
		})

		It("does not add 'name' attr when sql has no -- name: comment", func() {
			logger.Log(context.Background(), tracelog.LogLevelInfo, "Query", map[string]any{
				"sql": "SELECT 1",
			})
			Expect(handler.records).To(HaveLen(1))
			group := findGroupAttr(handler.records[0], "query")
			Expect(findAttrInGroup(group, "name")).To(BeNil())
		})

		It("dereferences pointer args before logging", func() {
			n := 99
			logger.Log(context.Background(), tracelog.LogLevelInfo, "Query", map[string]any{
				"args": []any{&n},
			})
			Expect(handler.records).To(HaveLen(1))
			group := findGroupAttr(handler.records[0], "query")
			argsAttr := findAttrInGroup(group, "args")
			Expect(argsAttr).NotTo(BeNil())
			Expect(argsAttr.Value.Any()).To(ContainElement(99))
		})

		It("adds PGX_LOG_LEVEL for LogLevelTrace (level < Debug)", func() {
			logger.Log(context.Background(), tracelog.LogLevelTrace, "trace msg", nil)
			Expect(handler.records).To(HaveLen(1))
			group := findGroupAttr(handler.records[0], "query")
			Expect(findAttrInGroup(group, "PGX_LOG_LEVEL")).NotTo(BeNil())
		})

		It("does not add PGX_LOG_LEVEL for Debug/Info/Warn/Error", func() {
			for _, severity := range []tracelog.LogLevel{
				tracelog.LogLevelDebug,
				tracelog.LogLevelInfo,
				tracelog.LogLevelWarn,
				tracelog.LogLevelError,
			} {
				handler.Reset()
				logger.Log(context.Background(), severity, "msg", nil)
				Expect(handler.records).To(HaveLen(1))
				group := findGroupAttr(handler.records[0], "query")
				Expect(findAttrInGroup(group, "PGX_LOG_LEVEL")).To(BeNil())
			}
		})

		It("routes to logger stored in context when ContextKey is set", func() {
			ctxHandler := &testHandler{}
			ctxLogger := slog.New(ctxHandler)
			ctx := context.WithValue(context.Background(), pgxslog.LoggerKey, ctxLogger)

			l := &pgxslog.Logger{ContextKey: pgxslog.LoggerKey}
			l.Log(ctx, tracelog.LogLevelInfo, "ctx msg", nil)

			Expect(ctxHandler.records).To(HaveLen(1))
			Expect(handler.records).To(BeEmpty())
		})

		It("falls back to slog.Default() when ContextKey is nil", func() {
			logger.Log(context.Background(), tracelog.LogLevelInfo, "default msg", nil)
			Expect(handler.records).To(HaveLen(1))
		})

		It("does not emit a record when handler is disabled for that level", func() {
			ddh := &disabledAtDebugHandler{}
			slog.SetDefault(slog.New(ddh))
			logger.Log(context.Background(), tracelog.LogLevelDebug, "debug msg", nil)
			Expect(ddh.records).To(BeEmpty())
		})

		It("does not panic with a nil data map", func() {
			Expect(func() {
				logger.Log(context.Background(), tracelog.LogLevelInfo, "nil data", nil)
			}).NotTo(Panic())
			Expect(handler.records).To(HaveLen(1))
		})

		It("converts camelCase data map keys to snake_case", func() {
			logger.Log(context.Background(), tracelog.LogLevelInfo, "Query", map[string]any{
				"rowsAffected": 5,
			})
			Expect(handler.records).To(HaveLen(1))
			group := findGroupAttr(handler.records[0], "query")
			Expect(findAttrInGroup(group, "rows_affected")).NotTo(BeNil())
		})
	})
})
