package main

import (
	"context"
	"database/sql"
)

type rowQuerier interface {
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

var (
	_ rowQuerier = (*sql.DB)(nil)
	_ rowQuerier = (*sql.Tx)(nil)
)

type postgresStore struct {
	db *sql.DB
}

func newPostgresStore(db *sql.DB) *postgresStore {
	return &postgresStore{
		db: db,
	}
}
