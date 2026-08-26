package main

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestOpenDatabaseRejectsEmptyURL(t *testing.T) {
	tests := []struct {
		name        string
		databaseURL string
	}{
		{
			name:        "empty",
			databaseURL: "",
		},
		{
			name:        "spaces only",
			databaseURL: "   ",
		},
		{
			name:        "whitespace only",
			databaseURL: "\t\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, err := openDatabase(
				context.Background(),
				tt.databaseURL,
			)

			if db != nil {
				_ = db.Close()

				t.Errorf(
					"openDatabase(%q) returned non-nil database",
					tt.databaseURL,
				)
			}

			if err == nil {
				t.Errorf(
					"openDatabase(%q) error = nil, want non-nil",
					tt.databaseURL,
				)
			}
		})
	}
}

func TestOpenDatabaseConnects(t *testing.T) {
	databaseURL := os.Getenv("MONEY_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("MONEY_TEST_DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	db, err := openDatabase(ctx, databaseURL)
	if err != nil {
		t.Fatalf("openDatabase() returned an unexpected error: %v", err)
	}

	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close database: %v", err)
		}
	})

	var got int

	if err := db.QueryRowContext(
		ctx,
		"SELECT 1",
	).Scan(&got); err != nil {
		t.Fatalf("query database: %v", err)
	}

	const want = 1

	if got != want {
		t.Errorf("SELECT 1 returned %d, want %d", got, want)
	}
}
