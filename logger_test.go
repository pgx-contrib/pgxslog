package pgxslog_test

import (
	"context"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/tracelog"
	"github.com/pgx-contrib/pgxslog"
)

var count int

func ExampleLogger() {
	config, err := pgxpool.ParseConfig(os.Getenv("PGX_DATABASE_URL"))
	if err != nil {
		panic(err)
	}

	config.ConnConfig.Tracer = &tracelog.TraceLog{
		Logger:   &pgxslog.Logger{},
		LogLevel: tracelog.LogLevelTrace,
	}

	conn, err := pgxpool.NewWithConfig(context.TODO(), config)
	if err != nil {
		panic(err)
	}

	row := conn.QueryRow(context.TODO(), "SELECT 1")
	if err := row.Scan(&count); err != nil {
		panic(err)
	}
}
