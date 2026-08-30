package main

import (
	"context"
	"database/sql"
)

type rowQuerier interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type queryExecutor interface {
	rowQuerier
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

var (
	_ queryExecutor = (*sql.DB)(nil)
	_ queryExecutor = (*sql.Tx)(nil)
)

type postgresStore struct {
	db *sql.DB
}

func newPostgresStore(db *sql.DB) *postgresStore {
	return &postgresStore{
		db: db,
	}
}
