# pgxslog

[![CI](https://github.com/pgx-contrib/pgxslog/actions/workflows/ci.yml/badge.svg)](https://github.com/pgx-contrib/pgxslog/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/pgx-contrib/pgxslog?include_prereleases)](https://github.com/pgx-contrib/pgxslog/releases)
[![Go Reference](https://pkg.go.dev/badge/github.com/pgx-contrib/pgxslog.svg)](https://pkg.go.dev/github.com/pgx-contrib/pgxslog)
[![License](https://img.shields.io/github/license/pgx-contrib/pgxslog)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![pgx](https://img.shields.io/badge/pgx-v5-blue)](https://github.com/jackc/pgx)

`Logger` is a [`log/slog`](https://pkg.go.dev/log/slog) adapter for [pgx v5](https://github.com/jackc/pgx)
that routes every database event through the standard structured logging pipeline.
Assign it to `tracelog.TraceLog` and queries, batches, prepared statements, and
connections are logged with snake_case keys, automatic SQL name extraction, and
optional per-request logger injection via context.

## Installation

```bash
go get github.com/pgx-contrib/pgxslog
```

## Usage

### Connection pool

```go
config, err := pgxpool.ParseConfig(os.Getenv("PGX_DATABASE_URL"))
if err != nil {
    panic(err)
}

config.ConnConfig.Tracer = &tracelog.TraceLog{
    Logger:   &pgxslog.Logger{},
    LogLevel: tracelog.LogLevelTrace,
}

pool, err := pgxpool.NewWithConfig(context.Background(), config)
if err != nil {
    panic(err)
}
defer pool.Close()
```

### Per-request logger via context

Store a `*slog.Logger` in the context (e.g. one already enriched with a
request ID) and `pgxslog.Logger` will use it instead of `slog.Default()`:

```go
logger := slog.New(slog.NewJSONHandler(os.Stdout, nil)).With("request_id", "abc123")
ctx := context.WithValue(ctx, pgxslog.LoggerKey, logger)

config.ConnConfig.Tracer = &tracelog.TraceLog{
    Logger:   &pgxslog.Logger{ContextKey: pgxslog.LoggerKey},
    LogLevel: tracelog.LogLevelInfo,
}
```

## Development

### DevContainer

Open in VS Code with the Dev Containers extension. The environment provides Go,
PostgreSQL 18, and Nix automatically.

```
PGX_DATABASE_URL=postgres://vscode@postgres:5432/pgxslog?sslmode=disable
```

### Nix

```bash
nix develop          # enter shell with Go
go tool ginkgo run -r
```

### Run tests

```bash
# Unit tests only (no database required)
go tool ginkgo run -r

# With integration tests
export PGX_DATABASE_URL="postgres://localhost/pgxslog?sslmode=disable"
go tool ginkgo run -r
```

## License

[MIT](LICENSE)
