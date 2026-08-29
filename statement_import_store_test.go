package main

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"money/internal/finance"
	"money/internal/statement"
)

func statementImportTestFingerprint(
	label string,
	id int64,
) Fingerprint {
	return Fingerprint(
		sha256.Sum256(
			[]byte(fmt.Sprintf("%s:%d", label, id)),
		),
	)
}

func createTestUserForStatementImport(
	t *testing.T,
) (
	context.Context,
	*postgresStore,
	User,
) {
	t.Helper()

	ctx, store := openTestPostgresStore(t)

	user, err := store.createUser(
		ctx,
		"Statement Import Test User",
	)
	if err != nil {
		t.Fatalf(
			"create statement import test user: %v",
			err,
		)
	}

	t.Cleanup(func() {
		deleteTestUser(t, store, user.ID)
	})

	return ctx, store, user
}

func newStatementImportForTest(
	userID int64,
) NewStatementImport {
	return NewStatementImport{
		Statement: NewStatement{
			UserID:           userID,
			Source:           statement.Revolut,
			OriginalFilename: "statement-import-test.csv",
			Fingerprint: statementImportTestFingerprint(
				"statement import file",
				userID,
			),
			RawHeader: []string{
				"Date",
				"Description",
				"Amount",
				"Currency",
			},
		},
		Transactions: []NewStatementTransaction{
			{
				Fingerprint: statementImportTestFingerprint(
					"statement import transaction 1",
					userID,
				),
				Transaction: finance.Transaction{
					Date: time.Date(
						2026,
						time.August,
						5,
						0,
						0,
						0,
						0,
						time.UTC,
					),
					Amount: finance.Money{
						Amount:   -2_500,
						Currency: finance.EUR,
					},
					Description:  "Groceries",
					Counterparty: "MAXIMA",
				},
				RawRecord: []string{
					"2026-08-05",
					"MAXIMA",
					"Groceries",
					"25.00",
					"EUR",
					"D",
				},
			},
			{
				Fingerprint: statementImportTestFingerprint(
					"statement import transaction 2",
					userID,
				),
				Transaction: finance.Transaction{
					Date: time.Date(
						2026,
						time.August,
						6,
						0,
						0,
						0,
						0,
						time.UTC,
					),
					Amount: finance.Money{
						Amount:   100_000,
						Currency: finance.EUR,
					},
					Description: "Salary",
				},
				RawRecord: []string{
					"2026-08-06",
					"",
					"Salary",
					"1000.00",
					"EUR",
					"K",
				},
			},
		},
	}
}

func TestCreateStatementImportCommitsStatementAndTransactions(
	t *testing.T,
) {
	ctx, store, user :=
		createTestUserForStatementImport(t)

	input := newStatementImportForTest(user.ID)

	got, err := store.createStatementImport(ctx, input)
	if err != nil {
		t.Fatalf(
			"createStatementImport() returned an unexpected error: %v",
			err,
		)
	}

	// Register statement cleanup before transaction cleanups.
	// t.Cleanup executes callbacks in reverse order, so the
	// transactions will be deleted before the statement.
	t.Cleanup(func() {
		deleteTestStatement(
			t,
			store,
			got.Statement.ID,
		)
	})

	for _, transaction := range got.Transactions {
		transactionID := transaction.ID

		t.Cleanup(func() {
			deleteTestTransaction(
				t,
				store,
				transactionID,
			)
		})
	}

	if got.Statement.ID <= 0 {
		t.Errorf(
			"created statement ID = %d, want positive",
			got.Statement.ID,
		)
	}

	if got.Statement.UserID != user.ID {
		t.Errorf(
			"created statement UserID = %d, want %d",
			got.Statement.UserID,
			user.ID,
		)
	}

	if len(got.Transactions) !=
		len(input.Transactions) {
		t.Fatalf(
			"created %d transactions, want %d",
			len(got.Transactions),
			len(input.Transactions),
		)
	}

	for index, transaction := range got.Transactions {
		if transaction.ID <= 0 {
			t.Errorf(
				"transaction %d ID = %d, want positive",
				index+1,
				transaction.ID,
			)
		}

		if transaction.StatementID != got.Statement.ID {
			t.Errorf(
				"transaction %d StatementID = %d, want %d",
				index+1,
				transaction.StatementID,
				got.Statement.ID,
			)
		}

		if transaction.Fingerprint !=
			input.Transactions[index].Fingerprint {
			t.Errorf(
				"transaction %d Fingerprint = %x, want %x",
				index+1,
				transaction.Fingerprint,
				input.Transactions[index].Fingerprint,
			)
		}

		if transaction.CreatedAt.IsZero() {
			t.Errorf(
				"transaction %d CreatedAt is zero",
				index+1,
			)
		}
	}

	var persistedStatementCount int64

	err = store.db.QueryRowContext(
		ctx,
		`
			SELECT count(*)
			FROM statements
			WHERE id = $1
		`,
		got.Statement.ID,
	).Scan(&persistedStatementCount)
	if err != nil {
		t.Fatalf(
			"count persisted statement: %v",
			err,
		)
	}

	if persistedStatementCount != 1 {
		t.Errorf(
			"persisted statement count = %d, want 1",
			persistedStatementCount,
		)
	}

	var persistedTransactionCount int64

	err = store.db.QueryRowContext(
		ctx,
		`
			SELECT count(*)
			FROM transactions
			WHERE statement_id = $1
		`,
		got.Statement.ID,
	).Scan(&persistedTransactionCount)
	if err != nil {
		t.Fatalf(
			"count persisted transactions: %v",
			err,
		)
	}

	if persistedTransactionCount !=
		int64(len(input.Transactions)) {
		t.Errorf(
			"persisted transaction count = %d, want %d",
			persistedTransactionCount,
			len(input.Transactions),
		)
	}
}

func TestCreateStatementImportRollsBackEverythingOnTransactionFailure(
	t *testing.T,
) {
	ctx, store, user :=
		createTestUserForStatementImport(t)

	input := newStatementImportForTest(user.ID)

	// Both financial transactions now have the same fingerprint.
	// The first INSERT succeeds; the second violates the unique
	// constraint on transactions.fingerprint.
	duplicateFingerprint :=
		input.Transactions[0].Fingerprint

	input.Transactions[1].Fingerprint =
		duplicateFingerprint

	got, err := store.createStatementImport(ctx, input)
	if err == nil {
		t.Fatal(
			"createStatementImport() error = nil, want non-nil",
		)
	}

	if !strings.Contains(
		err.Error(),
		"import transaction 2",
	) {
		t.Errorf(
			"createStatementImport() error = %q, want transaction position 2",
			err,
		)
	}

	if !reflect.DeepEqual(
		got,
		StoredStatementImport{},
	) {
		t.Errorf(
			"createStatementImport() = %+v, want zero StoredStatementImport",
			got,
		)
	}

	var statementCount int64

	err = store.db.QueryRowContext(
		ctx,
		`
			SELECT count(*)
			FROM statements
			WHERE user_id = $1
			  AND fingerprint = $2
		`,
		user.ID,
		input.Statement.Fingerprint[:],
	).Scan(&statementCount)
	if err != nil {
		t.Fatalf(
			"count rolled-back statements: %v",
			err,
		)
	}

	if statementCount != 0 {
		t.Errorf(
			"rolled-back statement count = %d, want 0",
			statementCount,
		)
	}

	var transactionCount int64

	err = store.db.QueryRowContext(
		ctx,
		`
			SELECT count(*)
			FROM transactions
			WHERE fingerprint = $1
		`,
		duplicateFingerprint[:],
	).Scan(&transactionCount)
	if err != nil {
		t.Fatalf(
			"count rolled-back transactions: %v",
			err,
		)
	}

	if transactionCount != 0 {
		t.Errorf(
			"rolled-back transaction count = %d, want 0",
			transactionCount,
		)
	}
}

func TestCreateStatementImportRejectsDuplicateStatement(t *testing.T) {
	ctx, store := openTestPostgresStore(t)

	user, err := store.createUser(
		ctx,
		"Duplicate Import Test User",
	)
	if err != nil {
		t.Fatalf("create test user: %v", err)
	}

	statementFingerprint := Fingerprint(sha256.Sum256(
		[]byte(fmt.Sprintf(
			"duplicate import statement:%d",
			user.ID,
		)),
	))

	transactionFingerprint := Fingerprint(sha256.Sum256(
		[]byte(fmt.Sprintf(
			"duplicate import transaction:%d",
			user.ID,
		)),
	))

	input := NewStatementImport{
		Statement: NewStatement{
			UserID:           user.ID,
			Source:           statement.Revolut,
			OriginalFilename: "duplicate-import.csv",
			Fingerprint:      statementFingerprint,
			RawHeader: []string{
				"Date",
				"Description",
				"Amount",
			},
		},
		Transactions: []NewStatementTransaction{
			{
				Fingerprint: transactionFingerprint,
				Transaction: finance.Transaction{
					Date: time.Date(
						2026,
						time.August,
						29,
						0,
						0,
						0,
						0,
						time.UTC,
					),
					Amount: finance.Money{
						Amount:   -1299,
						Currency: finance.EUR,
					},
					Description:  "Test purchase",
					Counterparty: "Test Shop",
				},
				RawRecord: []string{
					"2026-08-29",
					"Test purchase",
					"-12.99",
				},
			},
		},
	}

	first, err := store.createStatementImport(ctx, input)
	if err != nil {
		t.Fatalf(
			"first createStatementImport() returned an unexpected error: %v",
			err,
		)
	}

	t.Cleanup(func() {
		for _, transaction := range first.Transactions {
			deleteTestTransaction(t, store, transaction.ID)
		}

		deleteTestStatement(t, store, first.Statement.ID)
		deleteTestUser(t, store, user.ID)
	})

	got, err := store.createStatementImport(ctx, input)
	if err == nil {
		t.Fatal(
			"second createStatementImport() error = nil, want non-nil",
		)
	}

	if !errors.Is(err, ErrStatementAlreadyImported) {
		t.Errorf(
			"second createStatementImport() error = %v, want errors.Is(err, ErrStatementAlreadyImported)",
			err,
		)
	}

	if !reflect.DeepEqual(got, StoredStatementImport{}) {
		t.Errorf(
			"second createStatementImport() = %+v, want zero result",
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
		statementFingerprint[:],
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

	var transactionCount int

	err = store.db.QueryRowContext(
		ctx,
		`
			SELECT COUNT(*)
			FROM transactions
			WHERE statement_id = $1
		`,
		first.Statement.ID,
	).Scan(&transactionCount)
	if err != nil {
		t.Fatalf("count persisted transactions: %v", err)
	}

	if transactionCount != len(first.Transactions) {
		t.Errorf(
			"persisted transaction count = %d, want %d",
			transactionCount,
			len(first.Transactions),
		)
	}
}
