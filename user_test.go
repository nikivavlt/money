package main

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"
	"time"
)

func openTestPostgresStore(
	t *testing.T,
) (context.Context, *postgresStore) {
	t.Helper()

	databaseURL := os.Getenv("MONEY_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("MONEY_TEST_DATABASE_URL is not set")
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	t.Cleanup(cancel)

	db, err := openDatabase(ctx, databaseURL)
	if err != nil {
		t.Fatalf("openDatabase() returned an unexpected error: %v", err)
	}

	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close database: %v", err)
		}
	})

	return ctx, newPostgresStore(db)
}

func deleteTestUser(
	t *testing.T,
	store *postgresStore,
	id int64,
) {
	t.Helper()

	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	result, err := store.db.ExecContext(
		ctx,
		"DELETE FROM users WHERE id = $1",
		id,
	)
	if err != nil {
		t.Errorf("delete test user %d: %v", id, err)
		return
	}

	deleted, err := result.RowsAffected()
	if err != nil {
		t.Errorf(
			"get deleted count for user %d: %v",
			id,
			err,
		)
		return
	}

	if deleted != 1 {
		t.Errorf(
			"deleted %d rows for user %d, want 1",
			deleted,
			id,
		)
	}
}

func TestCreateUserRejectsEmptyName(t *testing.T) {
	store := newPostgresStore(nil)

	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "empty",
			input: "",
		},
		{
			name:  "spaces only",
			input: "   ",
		},
		{
			name:  "whitespace only",
			input: "\t\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := store.createUser(
				context.Background(),
				tt.input,
			)

			if err == nil {
				t.Fatalf(
					"createUser(%q) error = nil, want non-nil",
					tt.input,
				)
			}

			if got != (User{}) {
				t.Errorf(
					"createUser(%q) = %+v, want zero User",
					tt.input,
					got,
				)
			}
		})
	}
}

func TestCreateUserInsertsUser(t *testing.T) {
	ctx, store := openTestPostgresStore(t)

	got, err := store.createUser(ctx, "  Nikita Test  ")
	if err != nil {
		t.Fatalf("createUser() returned an unexpected error: %v", err)
	}

	t.Cleanup(func() {
		deleteTestUser(t, store, got.ID)
	})

	if got.ID <= 0 {
		t.Errorf("createUser() ID = %d, want positive", got.ID)
	}

	if got.Name != "Nikita Test" {
		t.Errorf(
			"createUser() Name = %q, want %q",
			got.Name,
			"Nikita Test",
		)
	}

	if got.CreatedAt.IsZero() {
		t.Error("createUser() CreatedAt is zero")
	}

	var persistedName string

	err = store.db.QueryRowContext(
		ctx,
		"SELECT name FROM users WHERE id = $1",
		got.ID,
	).Scan(&persistedName)
	if err != nil {
		t.Fatalf("query inserted user: %v", err)
	}

	if persistedName != got.Name {
		t.Errorf(
			"persisted user name = %q, want %q",
			persistedName,
			got.Name,
		)
	}
}

func TestFindUserByID(t *testing.T) {
	ctx, store := openTestPostgresStore(t)

	created, err := store.createUser(ctx, "Find User Test")
	if err != nil {
		t.Fatalf("createUser() returned an unexpected error: %v", err)
	}

	t.Cleanup(func() {
		deleteTestUser(t, store, created.ID)
	})

	t.Run("existing user", func(t *testing.T) {
		got, err := store.findUserByID(ctx, created.ID)
		if err != nil {
			t.Fatalf(
				"findUserByID(%d) returned an unexpected error: %v",
				created.ID,
				err,
			)
		}

		if got.ID != created.ID {
			t.Errorf(
				"findUserByID() ID = %d, want %d",
				got.ID,
				created.ID,
			)
		}

		if got.Name != created.Name {
			t.Errorf(
				"findUserByID() Name = %q, want %q",
				got.Name,
				created.Name,
			)
		}

		if !got.CreatedAt.Equal(created.CreatedAt) {
			t.Errorf(
				"findUserByID() CreatedAt = %v, want %v",
				got.CreatedAt,
				created.CreatedAt,
			)
		}
	})

	t.Run("missing user", func(t *testing.T) {
		got, err := store.findUserByID(ctx, -1)
		if err == nil {
			t.Fatal("findUserByID(-1) error = nil, want non-nil")
		}

		if !errors.Is(err, sql.ErrNoRows) {
			t.Errorf(
				"findUserByID(-1) error = %v, want it to match sql.ErrNoRows",
				err,
			)
		}

		if got != (User{}) {
			t.Errorf(
				"findUserByID(-1) = %+v, want zero User",
				got,
			)
		}
	})
}

func TestListUsers(t *testing.T) {
	ctx, store := openTestPostgresStore(t)

	first, err := store.createUser(ctx, "List Users First")
	if err != nil {
		t.Fatalf("create first test user: %v", err)
	}

	t.Cleanup(func() {
		deleteTestUser(t, store, first.ID)
	})

	second, err := store.createUser(ctx, "List Users Second")
	if err != nil {
		t.Fatalf("create second test user: %v", err)
	}

	t.Cleanup(func() {
		deleteTestUser(t, store, second.ID)
	})

	got, err := store.listUsers(ctx)
	if err != nil {
		t.Fatalf("listUsers() returned an unexpected error: %v", err)
	}

	for i := 1; i < len(got); i++ {
		if got[i-1].ID > got[i].ID {
			t.Errorf(
				"listUsers() is not ordered by ID: user %d appears before user %d",
				got[i-1].ID,
				got[i].ID,
			)
		}
	}

	wantUsers := []User{
		first,
		second,
	}

	for _, want := range wantUsers {
		var found User
		exists := false

		for _, user := range got {
			if user.ID == want.ID {
				found = user
				exists = true
				break
			}
		}

		if !exists {
			t.Errorf(
				"listUsers() does not contain created user %d",
				want.ID,
			)
			continue
		}

		if found.Name != want.Name {
			t.Errorf(
				"user %d name = %q, want %q",
				want.ID,
				found.Name,
				want.Name,
			)
		}

		if !found.CreatedAt.Equal(want.CreatedAt) {
			t.Errorf(
				"user %d CreatedAt = %v, want %v",
				want.ID,
				found.CreatedAt,
				want.CreatedAt,
			)
		}
	}
}
