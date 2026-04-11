# pgxslog

A `log/slog` adapter for [pgx v5](https://github.com/jackc/pgx).

[![Go Reference](https://pkg.go.dev/badge/github.com/pgx-contrib/pgxslog.svg)](https://pkg.go.dev/github.com/pgx-contrib/pgxslog)
[![CI](https://github.com/pgx-contrib/pgxslog/actions/workflows/ci.yml/badge.svg)](https://github.com/pgx-contrib/pgxslog/actions/workflows/ci.yml)

## Installation

```bash
go get github.com/pgx-contrib/pgxslog
```

## Usage

```go
import (
    "github.com/jackc/pgx/v5/pgxpool"
    "github.com/jackc/pgx/v5/tracelog"
    "github.com/pgx-contrib/pgxslog"
)

config, err := pgxpool.ParseConfig(os.Getenv("PGX_DATABASE_URL"))
if err != nil {
    panic(err)
}

config.ConnConfig.Tracer = &tracelog.TraceLog{
    Logger:   &pgxslog.Logger{},
    LogLevel: tracelog.LogLevelTrace,
}

pool, err := pgxpool.NewWithConfig(context.Background(), config)
```

### Per-request logger via context

```go
logger := slog.New(slog.NewJSONHandler(os.Stdout, nil)).With("request_id", "abc123")
ctx := context.WithValue(ctx, pgxslog.LoggerKey, logger)

// pgxslog.Logger will use the logger stored in the context
config.ConnConfig.Tracer = &tracelog.TraceLog{
    Logger:   &pgxslog.Logger{ContextKey: pgxslog.LoggerKey},
    LogLevel: tracelog.LogLevelInfo,
}
```

## API Reference

### `Logger`

```go
type Logger struct {
    // ContextKey, when set, causes Log to look up a *slog.Logger in the
    // context using this key. Falls back to slog.Default() if absent.
    ContextKey any
}
```

Implements `tracelog.Logger`. Wraps each log call in a `"query"` slog group containing the data fields provided by pgx.

### `ConvertSeverity(severity tracelog.LogLevel) slog.Level`

Maps pgx log levels to slog levels. Levels below `slog.LevelDebug` (Trace and None) are passed through as-is so handlers can filter them.

### `ConvertArgs(args []any) []any`

Dereferences pointer arguments for cleaner log output.

### `ConvertAttr(key string, value any) slog.Attr`

Converts camelCase or PascalCase keys to `snake_case` and wraps the value in a `slog.Attr`.

### `NameRegexp`

Regular expression (`^--\s+name:\s+(\w+)`) used to extract a named operation from a SQL comment. When matched, a `"name"` attribute is added to the log record.

### `LoggerKey`

A package-level `*ContextKey` that can be used as the `Logger.ContextKey` to store/retrieve a `*slog.Logger` in a context.

## Development

This project uses [Nix](https://nixos.org/) + [devcontainer](https://containers.dev/) for reproducible development environments.

```bash
# Open devShell (requires Nix with flakes enabled)
nix develop

# Run tests
go tool ginkgo run -r

# Validate the Nix flake
nix flake check
```

### VS Code / devcontainer

Open the project in VS Code and select **Reopen in Container** when prompted. The devcontainer starts a PostgreSQL 18 service and sets `PGX_DATABASE_URL` automatically.
