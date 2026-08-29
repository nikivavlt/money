package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"money/internal/statement"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

func deleteTestStatement(
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
		"DELETE FROM statements WHERE id = $1",
		id,
	)
	if err != nil {
		t.Errorf("delete test statement %d: %v", id, err)
		return
	}

	deleted, err := result.RowsAffected()
	if err != nil {
		t.Errorf(
			"get deleted count for statement %d: %v",
			id,
			err,
		)
		return
	}

	if deleted != 1 {
		t.Errorf(
			"deleted %d rows for statement %d, want 1",
			deleted,
			id,
		)
	}
}

func TestCreateStatementRejectsInvalidInput(t *testing.T) {
	store := newPostgresStore(nil)

	validInput := func() NewStatement {
		return NewStatement{
			UserID:           1,
			Source:           statement.Revolut,
			OriginalFilename: "statement.csv",
			Fingerprint: Fingerprint(
				sha256.Sum256([]byte("validation test")),
			),
			RawHeader: []string{"Date"},
		}
	}

	tests := []struct {
		name      string
		change    func(*NewStatement)
		wantError string
	}{
		{
			name: "zero user ID",
			change: func(input *NewStatement) {
				input.UserID = 0
			},
			wantError: "statement user ID must be positive",
		},
		{
			name: "negative user ID",
			change: func(input *NewStatement) {
				input.UserID = -1
			},
			wantError: "statement user ID must be positive",
		},
		{
			name: "unsupported source",
			change: func(input *NewStatement) {
				input.Source = statement.Source("wise")
			},
			wantError: `unsupported statement source "wise"`,
		},
		{
			name: "empty filename",
			change: func(input *NewStatement) {
				input.OriginalFilename = ""
			},
			wantError: "statement filename is empty",
		},
		{
			name: "whitespace filename",
			change: func(input *NewStatement) {
				input.OriginalFilename = " \t\n "
			},
			wantError: "statement filename is empty",
		},
		{
			name: "empty header",
			change: func(input *NewStatement) {
				input.RawHeader = nil
			},
			wantError: "statement header is empty",
		},
		{
			name: "empty fingerprint",
			change: func(input *NewStatement) {
				input.Fingerprint = Fingerprint{}
			},
			wantError: "statement fingerprint is empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := validInput()
			tt.change(&input)

			got, err := store.createStatement(
				context.Background(),
				input,
			)
			if err == nil {
				t.Fatal("createStatement() error = nil, want non-nil")
			}

			if err.Error() != tt.wantError {
				t.Errorf(
					"createStatement() error = %q, want %q",
					err,
					tt.wantError,
				)
			}

			if !reflect.DeepEqual(got, Statement{}) {
				t.Errorf(
					"createStatement() = %+v, want zero Statement",
					got,
				)
			}
		})
	}
}

func TestCreateStatementInsertsStatement(t *testing.T) {
	ctx, store := openTestPostgresStore(t)

	user, err := store.createUser(ctx, "Statement Test User")
	if err != nil {
		t.Fatalf("create test user: %v", err)
	}

	t.Cleanup(func() {
		deleteTestUser(t, store, user.ID)
	})

	wantHeader := []string{
		"Date",
		`Description "quoted"`,
		"Amount €",
	}

	fingerprint := Fingerprint(sha256.Sum256(
		[]byte(fmt.Sprintf(
			"statement test:%d",
			user.ID,
		)),
	))

	input := NewStatement{
		UserID:           user.ID,
		Source:           statement.Revolut,
		OriginalFilename: "  revolut-test.csv  ",
		Fingerprint:      fingerprint,
		RawHeader:        slices.Clone(wantHeader),
	}

	got, err := store.createStatement(ctx, input)
	if err != nil {
		t.Fatalf(
			"createStatement() returned an unexpected error: %v",
			err,
		)
	}

	t.Cleanup(func() {
		deleteTestStatement(t, store, got.ID)
	})

	if got.ID <= 0 {
		t.Errorf("createStatement() ID = %d, want positive", got.ID)
	}

	if got.UserID != input.UserID {
		t.Errorf(
			"createStatement() UserID = %d, want %d",
			got.UserID,
			input.UserID,
		)
	}

	if got.Source != input.Source {
		t.Errorf(
			"createStatement() Source = %q, want %q",
			got.Source,
			input.Source,
		)
	}

	if got.OriginalFilename != "revolut-test.csv" {
		t.Errorf(
			"createStatement() OriginalFilename = %q, want %q",
			got.OriginalFilename,
			"revolut-test.csv",
		)
	}

	if got.Fingerprint != input.Fingerprint {
		t.Errorf(
			"createStatement() Fingerprint = %x, want %x",
			got.Fingerprint,
			input.Fingerprint,
		)
	}

	if !slices.Equal(got.RawHeader, wantHeader) {
		t.Errorf(
			"createStatement() RawHeader = %q, want %q",
			got.RawHeader,
			wantHeader,
		)
	}

	if got.ImportedAt.IsZero() {
		t.Error("createStatement() ImportedAt is zero")
	}

	input.RawHeader[0] = "Changed"

	if !slices.Equal(got.RawHeader, wantHeader) {
		t.Errorf(
			"returned RawHeader changed after input mutation: got %q, want %q",
			got.RawHeader,
			wantHeader,
		)
	}

	var (
		persistedUserID      int64
		persistedSource      string
		persistedFilename    string
		persistedFingerprint []byte
		persistedHeaderJSON  []byte
		persistedImportedAt  time.Time
	)

	err = store.db.QueryRowContext(
		ctx,
		`
			SELECT
				user_id,
				source,
				original_filename,
				fingerprint,
				raw_header,
				imported_at
			FROM statements
			WHERE id = $1
		`,
		got.ID,
	).Scan(
		&persistedUserID,
		&persistedSource,
		&persistedFilename,
		&persistedFingerprint,
		&persistedHeaderJSON,
		&persistedImportedAt,
	)
	if err != nil {
		t.Fatalf("query inserted statement: %v", err)
	}

	var persistedHeader []string

	if err := json.Unmarshal(
		persistedHeaderJSON,
		&persistedHeader,
	); err != nil {
		t.Fatalf("decode persisted statement header: %v", err)
	}

	if persistedUserID != input.UserID {
		t.Errorf(
			"persisted user ID = %d, want %d",
			persistedUserID,
			input.UserID,
		)
	}

	if persistedSource != string(input.Source) {
		t.Errorf(
			"persisted source = %q, want %q",
			persistedSource,
			input.Source,
		)
	}

	if persistedFilename != "revolut-test.csv" {
		t.Errorf(
			"persisted filename = %q, want %q",
			persistedFilename,
			"revolut-test.csv",
		)
	}

	if !bytes.Equal(
		persistedFingerprint,
		input.Fingerprint[:],
	) {
		t.Errorf(
			"persisted fingerprint = %x, want %x",
			persistedFingerprint,
			input.Fingerprint,
		)
	}

	if !slices.Equal(persistedHeader, wantHeader) {
		t.Errorf(
			"persisted header = %q, want %q",
			persistedHeader,
			wantHeader,
		)
	}

	if !persistedImportedAt.Equal(got.ImportedAt) {
		t.Errorf(
			"persisted ImportedAt = %v, want %v",
			persistedImportedAt,
			got.ImportedAt,
		)
	}
}

func TestIsUniqueViolation(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "unique violation",
			err: &pgconn.PgError{
				Code: postgresUniqueViolation,
			},
			want: true,
		},
		{
			name: "wrapped unique violation",
			err: fmt.Errorf(
				"insert statement: %w",
				&pgconn.PgError{
					Code: postgresUniqueViolation,
				},
			),
			want: true,
		},
		{
			name: "foreign key violation",
			err: &pgconn.PgError{
				Code: "23503",
			},
			want: false,
		},
		{
			name: "ordinary error",
			err:  errors.New("database unavailable"),
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isUniqueViolation(tt.err)

			if got != tt.want {
				t.Errorf(
					"isUniqueViolation() = %v, want %v",
					got,
					tt.want,
				)
			}
		})
	}
}

func TestCreateStatementRejectsDuplicateFingerprint(
	t *testing.T,
) {
	ctx, store := openTestPostgresStore(t)

	user, err := store.createUser(
		ctx,
		"Duplicate Statement Test User",
	)
	if err != nil {
		t.Fatalf("create test user: %v", err)
	}

	t.Cleanup(func() {
		deleteTestUser(t, store, user.ID)
	})

	fingerprint := Fingerprint(sha256.Sum256(
		[]byte(fmt.Sprintf(
			"duplicate statement:%d",
			user.ID,
		)),
	))

	firstInput := NewStatement{
		UserID:           user.ID,
		Source:           statement.Revolut,
		OriginalFilename: "first.csv",
		Fingerprint:      fingerprint,
		RawHeader:        []string{"Date", "Amount"},
	}

	first, err := store.createStatement(ctx, firstInput)
	if err != nil {
		t.Fatalf(
			"first createStatement() returned an unexpected error: %v",
			err,
		)
	}

	t.Cleanup(func() {
		deleteTestStatement(t, store, first.ID)
	})

	duplicateInput := NewStatement{
		UserID:           user.ID,
		Source:           statement.Swedbank,
		OriginalFilename: "different.csv",
		Fingerprint:      fingerprint,
		RawHeader:        []string{"Different", "Header"},
	}

	got, err := store.createStatement(ctx, duplicateInput)
	if err == nil {
		t.Fatal(
			"duplicate createStatement() error = nil, want non-nil",
		)
	}

	if !errors.Is(err, ErrStatementAlreadyImported) {
		t.Errorf(
			"duplicate createStatement() error = %v, want ErrStatementAlreadyImported",
			err,
		)
	}

	if !reflect.DeepEqual(got, Statement{}) {
		t.Errorf(
			"duplicate createStatement() = %+v, want zero Statement",
			got,
		)
	}

	var statementCount int

	err = store.db.QueryRowContext(
		ctx,
		`
			SELECT COUNT(*)
			FROM statements
			WHERE user_id = $1
			  AND fingerprint = $2
		`,
		user.ID,
		fingerprint[:],
	).Scan(&statementCount)
	if err != nil {
		t.Fatalf("count persisted statements: %v", err)
	}

	if statementCount != 1 {
		t.Errorf(
			"persisted statement count = %d, want 1",
			statementCount,
		)
	}
}
