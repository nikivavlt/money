package main

import (
	"context"
	"fmt"
)

func openCommandStore(ctx context.Context, getenv func(string) string) (*postgresStore, error) {
	db, err := openDatabase(ctx, getenv("MONEY_DATABASE_URL"))
	if err != nil {
		return nil, fmt.Errorf("connect to database: %w", err)
	}

	return newPostgresStore(db), nil
}
