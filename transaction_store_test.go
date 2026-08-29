package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"testing"
	"time"

	"money/internal/finance"
	"money/internal/statement"
)

func validNewTransactionForTest() NewTransaction {
	return NewTransaction{
		StatementID: 1,
		Fingerprint: Fingerprint(
			sha256.Sum256(
				[]byte("transaction validation test"),
			),
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
	}
}

func deleteTestTransaction(
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
		"DELETE FROM transactions WHERE id = $1",
		id,
	)
	if err != nil {
		t.Errorf(
			"delete test transaction %d: %v",
			id,
			err,
		)
		return
	}

	deleted, err := result.RowsAffected()
	if err != nil {
		t.Errorf(
			"get deleted count for transaction %d: %v",
			id,
			err,
		)
		return
	}

	if deleted != 1 {
		t.Errorf(
			"deleted %d rows for transaction %d, want 1",
			deleted,
			id,
		)
	}
}

func createTestStatementForTransaction(
	t *testing.T,
) (
	context.Context,
	*postgresStore,
	Statement,
) {
	t.Helper()

	ctx, store := openTestPostgresStore(t)

	user, err := store.createUser(
		ctx,
		"Transaction Store Test User",
	)
	if err != nil {
		t.Fatalf("create transaction test user: %v", err)
	}

	t.Cleanup(func() {
		deleteTestUser(t, store, user.ID)
	})

	fingerprint := Fingerprint(
		sha256.Sum256(
			[]byte(fmt.Sprintf(
				"transaction store statement:%d",
				user.ID,
			)),
		),
	)

	created, err := store.createStatement(
		ctx,
		NewStatement{
			UserID:           user.ID,
			Source:           statement.Revolut,
			OriginalFilename: "transaction-store-test.csv",
			Fingerprint:      fingerprint,
			RawHeader: []string{
				"Date",
				"Description",
				"Amount",
				"Currency",
			},
		},
	)
	if err != nil {
		t.Fatalf(
			"create transaction test statement: %v",
			err,
		)
	}

	t.Cleanup(func() {
		deleteTestStatement(t, store, created.ID)
	})

	return ctx, store, created
}

func TestValidateNewTransactionRejectsInvalidInput(
	t *testing.T,
) {
	tests := []struct {
		name      string
		change    func(*NewTransaction)
		wantError string
	}{
		{
			name: "zero statement ID",
			change: func(input *NewTransaction) {
				input.StatementID = 0
			},
			wantError: "transaction statement ID must be positive",
		},
		{
			name: "negative statement ID",
			change: func(input *NewTransaction) {
				input.StatementID = -1
			},
			wantError: "transaction statement ID must be positive",
		},
		{
			name: "empty fingerprint",
			change: func(input *NewTransaction) {
				input.Fingerprint = Fingerprint{}
			},
			wantError: "transaction fingerprint is empty",
		},
		{
			name: "zero date",
			change: func(input *NewTransaction) {
				input.Transaction.Date = time.Time{}
			},
			wantError: "transaction date is zero",
		},
		{
			name: "unsupported currency",
			change: func(input *NewTransaction) {
				input.Transaction.Amount.Currency =
					finance.Currency("GBP")
			},
			wantError: `transaction currency: unsupported currency "GBP"`,
		},
		{
			name: "empty currency",
			change: func(input *NewTransaction) {
				input.Transaction.Amount.Currency = ""
			},
			wantError: `transaction currency: unsupported currency ""`,
		},
		{
			name: "empty description",
			change: func(input *NewTransaction) {
				input.Transaction.Description = ""
			},
			wantError: "transaction description is empty",
		},
		{
			name: "whitespace description",
			change: func(input *NewTransaction) {
				input.Transaction.Description = " \t\n "
			},
			wantError: "transaction description is empty",
		},
		{
			name: "nil raw record",
			change: func(input *NewTransaction) {
				input.RawRecord = nil
			},
			wantError: "transaction raw record is empty",
		},
		{
			name: "empty non-nil raw record",
			change: func(input *NewTransaction) {
				input.RawRecord = []string{}
			},
			wantError: "transaction raw record is empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := validNewTransactionForTest()
			tt.change(&input)

			err := validateNewTransaction(input)
			if err == nil {
				t.Fatal(
					"validateNewTransaction() error = nil, want non-nil",
				)
			}

			if err.Error() != tt.wantError {
				t.Errorf(
					"validateNewTransaction() error = %q, want %q",
					err,
					tt.wantError,
				)
			}
		})
	}
}

func TestValidateNewTransactionAcceptsValidInput(
	t *testing.T,
) {
	tests := []struct {
		name   string
		change func(*NewTransaction)
	}{
		{
			name: "ordinary transaction",
			change: func(input *NewTransaction) {
				// Keep the standard valid input.
			},
		},
		{
			name: "zero amount",
			change: func(input *NewTransaction) {
				input.Transaction.Amount.Amount = 0
			},
		},
		{
			name: "empty counterparty",
			change: func(input *NewTransaction) {
				input.Transaction.Counterparty = ""
			},
		},
		{
			name: "empty fields inside raw record",
			change: func(input *NewTransaction) {
				input.RawRecord = []string{
					"2026-08-05",
					"",
					"Groceries",
					"25.00",
					"",
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := validNewTransactionForTest()
			tt.change(&input)

			if err := validateNewTransaction(input); err != nil {
				t.Errorf(
					"validateNewTransaction() returned an unexpected error: %v",
					err,
				)
			}
		})
	}
}

func TestValidateNewTransactionPreservesCurrencyError(
	t *testing.T,
) {
	input := validNewTransactionForTest()
	input.Transaction.Amount.Currency =
		finance.Currency("GBP")

	err := validateNewTransaction(input)
	if err == nil {
		t.Fatal(
			"validateNewTransaction() error = nil, want non-nil",
		)
	}

	if !errors.Is(err, finance.ErrUnsupportedCurrency) {
		t.Errorf(
			"validateNewTransaction() error = %v, want it to match ErrUnsupportedCurrency",
			err,
		)
	}
}

func TestCreateTransactionRejectsInvalidInputBeforeDatabaseAccess(
	t *testing.T,
) {
	store := newPostgresStore(nil)
	input := validNewTransactionForTest()
	input.StatementID = 0

	got, err := store.createTransaction(
		context.Background(),
		input,
	)
	if err == nil {
		t.Fatal(
			"createTransaction() error = nil, want non-nil",
		)
	}

	const wantError = "transaction statement ID must be positive"

	if err.Error() != wantError {
		t.Errorf(
			"createTransaction() error = %q, want %q",
			err,
			wantError,
		)
	}

	if !reflect.DeepEqual(
		got,
		StoredTransaction{},
	) {
		t.Errorf(
			"createTransaction() = %+v, want zero StoredTransaction",
			got,
		)
	}
}

func TestCreateTransactionInsertsTransaction(
	t *testing.T,
) {
	ctx, store, createdStatement :=
		createTestStatementForTransaction(t)

	wantRawRecord := []string{
		"2026-08-05",
		"MAXIMA",
		"Groceries",
		"25.00",
		"EUR",
		"D",
	}

	fingerprint := Fingerprint(
		sha256.Sum256(
			[]byte(fmt.Sprintf(
				"transaction test:%d",
				createdStatement.ID,
			)),
		),
	)

	input := NewTransaction{
		StatementID: createdStatement.ID,
		Fingerprint: fingerprint,
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
			Description:  "  Groceries  ",
			Counterparty: "  MAXIMA  ",
		},
		RawRecord: slices.Clone(wantRawRecord),
	}

	got, err := store.createTransaction(ctx, input)
	if err != nil {
		t.Fatalf(
			"createTransaction() returned an unexpected error: %v",
			err,
		)
	}

	t.Cleanup(func() {
		deleteTestTransaction(t, store, got.ID)
	})

	if got.ID <= 0 {
		t.Errorf(
			"createTransaction() ID = %d, want positive",
			got.ID,
		)
	}

	if got.StatementID != input.StatementID {
		t.Errorf(
			"createTransaction() StatementID = %d, want %d",
			got.StatementID,
			input.StatementID,
		)
	}

	if got.Fingerprint != input.Fingerprint {
		t.Errorf(
			"createTransaction() Fingerprint = %x, want %x",
			got.Fingerprint,
			input.Fingerprint,
		)
	}

	if !got.Transaction.Date.Equal(
		input.Transaction.Date,
	) {
		t.Errorf(
			"createTransaction() Date = %v, want %v",
			got.Transaction.Date,
			input.Transaction.Date,
		)
	}

	if got.Transaction.Amount !=
		input.Transaction.Amount {
		t.Errorf(
			"createTransaction() Amount = %+v, want %+v",
			got.Transaction.Amount,
			input.Transaction.Amount,
		)
	}

	if got.Transaction.Description != "Groceries" {
		t.Errorf(
			"createTransaction() Description = %q, want %q",
			got.Transaction.Description,
			"Groceries",
		)
	}

	if got.Transaction.Counterparty != "MAXIMA" {
		t.Errorf(
			"createTransaction() Counterparty = %q, want %q",
			got.Transaction.Counterparty,
			"MAXIMA",
		)
	}

	if !slices.Equal(
		got.RawRecord,
		wantRawRecord,
	) {
		t.Errorf(
			"createTransaction() RawRecord = %q, want %q",
			got.RawRecord,
			wantRawRecord,
		)
	}

	if got.CreatedAt.IsZero() {
		t.Error(
			"createTransaction() CreatedAt is zero",
		)
	}

	input.RawRecord[0] = "Changed"

	if !slices.Equal(
		got.RawRecord,
		wantRawRecord,
	) {
		t.Errorf(
			"returned RawRecord changed after input mutation: got %q, want %q",
			got.RawRecord,
			wantRawRecord,
		)
	}

	var (
		persistedStatementID  int64
		persistedFingerprint  []byte
		persistedDate         time.Time
		persistedAmount       int64
		persistedCurrency     string
		persistedDescription  string
		persistedCounterparty sql.NullString
		persistedRawJSON      []byte
		persistedCreatedAt    time.Time
	)

	err = store.db.QueryRowContext(
		ctx,
		`
			SELECT
				statement_id,
				fingerprint,
				transaction_date,
				amount_minor,
				currency,
				description,
				counterparty,
				raw_record,
				created_at
			FROM transactions
			WHERE id = $1
		`,
		got.ID,
	).Scan(
		&persistedStatementID,
		&persistedFingerprint,
		&persistedDate,
		&persistedAmount,
		&persistedCurrency,
		&persistedDescription,
		&persistedCounterparty,
		&persistedRawJSON,
		&persistedCreatedAt,
	)
	if err != nil {
		t.Fatalf(
			"query inserted transaction: %v",
			err,
		)
	}

	var persistedRawRecord []string

	if err := json.Unmarshal(
		persistedRawJSON,
		&persistedRawRecord,
	); err != nil {
		t.Fatalf(
			"decode persisted transaction raw record: %v",
			err,
		)
	}

	if persistedStatementID != input.StatementID {
		t.Errorf(
			"persisted statement ID = %d, want %d",
			persistedStatementID,
			input.StatementID,
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

	const dateLayout = "2006-01-02"

	if persistedDate.Format(dateLayout) !=
		input.Transaction.Date.Format(dateLayout) {
		t.Errorf(
			"persisted date = %v, want calendar date %v",
			persistedDate,
			input.Transaction.Date,
		)
	}

	if persistedAmount !=
		int64(input.Transaction.Amount.Amount) {
		t.Errorf(
			"persisted amount = %d, want %d",
			persistedAmount,
			input.Transaction.Amount.Amount,
		)
	}

	if persistedCurrency !=
		string(input.Transaction.Amount.Currency) {
		t.Errorf(
			"persisted currency = %q, want %q",
			persistedCurrency,
			input.Transaction.Amount.Currency,
		)
	}

	if persistedDescription != "Groceries" {
		t.Errorf(
			"persisted description = %q, want %q",
			persistedDescription,
			"Groceries",
		)
	}

	if !persistedCounterparty.Valid {
		t.Error(
			"persisted counterparty is NULL, want non-NULL",
		)
	} else if persistedCounterparty.String != "MAXIMA" {
		t.Errorf(
			"persisted counterparty = %q, want %q",
			persistedCounterparty.String,
			"MAXIMA",
		)
	}

	if !slices.Equal(
		persistedRawRecord,
		wantRawRecord,
	) {
		t.Errorf(
			"persisted raw record = %q, want %q",
			persistedRawRecord,
			wantRawRecord,
		)
	}

	if !persistedCreatedAt.Equal(got.CreatedAt) {
		t.Errorf(
			"persisted CreatedAt = %v, want %v",
			persistedCreatedAt,
			got.CreatedAt,
		)
	}
}

func TestCreateTransactionStoresEmptyCounterpartyAsNull(
	t *testing.T,
) {
	ctx, store, createdStatement :=
		createTestStatementForTransaction(t)

	input := NewTransaction{
		StatementID: createdStatement.ID,
		Fingerprint: Fingerprint(
			sha256.Sum256(
				[]byte(fmt.Sprintf(
					"empty counterparty:%d",
					createdStatement.ID,
				)),
			),
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
			Description:  "Salary",
			Counterparty: " \t\n ",
		},
		RawRecord: []string{
			"2026-08-06",
			"Salary",
			"1000.00",
			"EUR",
		},
	}

	got, err := store.createTransaction(ctx, input)
	if err != nil {
		t.Fatalf(
			"createTransaction() returned an unexpected error: %v",
			err,
		)
	}

	t.Cleanup(func() {
		deleteTestTransaction(t, store, got.ID)
	})

	if got.Transaction.Counterparty != "" {
		t.Errorf(
			"createTransaction() Counterparty = %q, want empty",
			got.Transaction.Counterparty,
		)
	}

	var persistedCounterparty sql.NullString

	err = store.db.QueryRowContext(
		ctx,
		`
			SELECT counterparty
			FROM transactions
			WHERE id = $1
		`,
		got.ID,
	).Scan(&persistedCounterparty)
	if err != nil {
		t.Fatalf(
			"query transaction counterparty: %v",
			err,
		)
	}

	if persistedCounterparty.Valid {
		t.Errorf(
			"persisted counterparty = %q, want NULL",
			persistedCounterparty.String,
		)
	}
}
