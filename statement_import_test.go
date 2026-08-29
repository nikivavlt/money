package main

import (
	"context"
	"crypto/sha256"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"money/internal/finance"
	"money/internal/statement"
)

func TestNewStatementImport(t *testing.T) {
	statementFingerprint := statement.Fingerprint(
		sha256.Sum256([]byte("statement")),
	)

	firstTransactionFingerprint := statement.Fingerprint(
		sha256.Sum256([]byte("transaction 1")),
	)

	secondTransactionFingerprint := statement.Fingerprint(
		sha256.Sum256([]byte("transaction 2")),
	)

	prepared := statement.Prepared{
		Source:      statement.Swedbank,
		Fingerprint: statementFingerprint,
		RawHeader: []string{
			"Date",
			"Description",
			"Amount",
		},
		Transactions: []statement.PreparedTransaction{
			{
				Fingerprint: firstTransactionFingerprint,
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
					Description:  "First purchase",
					Counterparty: "First shop",
				},
				RawRecord: []string{
					"2026-08-29",
					"First purchase",
					"-12.99",
				},
			},
			{
				Fingerprint: secondTransactionFingerprint,
				Transaction: finance.Transaction{
					Date: time.Date(
						2026,
						time.August,
						30,
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
					Counterparty: "Employer",
				},
				RawRecord: []string{
					"2026-08-30",
					"Salary",
					"1000.00",
				},
			},
		},
	}

	got := newStatementImport(
		42,
		" statement.csv ",
		prepared,
	)

	if got.Statement.UserID != 42 {
		t.Errorf("UserID = %d, want 42", got.Statement.UserID)
	}

	if got.Statement.Source != statement.Swedbank {
		t.Errorf("Source = %q, want %q", got.Statement.Source, statement.Swedbank)
	}

	if got.Statement.OriginalFilename != " statement.csv " {
		t.Errorf("OriginalFilename = %q, want %q", got.Statement.OriginalFilename, " statement.csv ")
	}

	if got.Statement.Fingerprint != Fingerprint(statementFingerprint) {
		t.Errorf("statement Fingerprint = %x, want %x", got.Statement.Fingerprint, statementFingerprint)
	}

	if !slices.Equal(got.Statement.RawHeader, prepared.RawHeader) {
		t.Errorf("RawHeader = %q, want %q", got.Statement.RawHeader, prepared.RawHeader)
	}

	if len(got.Transactions) != 2 {
		t.Fatalf("transaction count = %d, want 2", len(got.Transactions))
	}

	if got.Transactions[0].Fingerprint != Fingerprint(firstTransactionFingerprint) {
		t.Errorf("first Fingerprint = %x, want %x", got.Transactions[0].Fingerprint, firstTransactionFingerprint)
	}

	if got.Transactions[0].Transaction != prepared.Transactions[0].Transaction {
		t.Errorf("first Transaction = %+v, want %+v", got.Transactions[0].Transaction, prepared.Transactions[0].Transaction)
	}

	if !slices.Equal(got.Transactions[0].RawRecord, prepared.Transactions[0].RawRecord) {
		t.Errorf("first RawRecord = %q, want %q", got.Transactions[0].RawRecord, prepared.Transactions[0].RawRecord)
	}

	if got.Transactions[1].Fingerprint != Fingerprint(secondTransactionFingerprint) {
		t.Errorf("second Fingerprint = %x, want %x", got.Transactions[1].Fingerprint, secondTransactionFingerprint)
	}

	if got.Transactions[1].Transaction != prepared.Transactions[1].Transaction {
		t.Errorf("second Transaction = %+v, want %+v", got.Transactions[1].Transaction, prepared.Transactions[1].Transaction)
	}

	if !slices.Equal(got.Transactions[1].RawRecord, prepared.Transactions[1].RawRecord) {
		t.Errorf("second RawRecord = %q, want %q", got.Transactions[1].RawRecord, prepared.Transactions[1].RawRecord)
	}

	prepared.RawHeader[0] = "Changed"
	prepared.Transactions[0].RawRecord[0] = "Changed"

	if got.Statement.RawHeader[0] != "Date" {
		t.Errorf("RawHeader changed through input alias: %q", got.Statement.RawHeader)
	}

	if got.Transactions[0].RawRecord[0] != "2026-08-29" {
		t.Errorf("RawRecord changed through input alias: %q", got.Transactions[0].RawRecord)
	}
}

func TestImportStatementPersistsPreparedData(t *testing.T) {
	ctx, store := openTestPostgresStore(t)

	user, err := store.createUser(ctx, "Application Import Test User")
	if err != nil {
		t.Fatalf("create test user: %v", err)
	}

	rawInput := "" +
		"Account No,Date,Beneficiary,Details,Amount,Currency,D/K,Record ID,Code\n" +
		"LT123,2026-08-29,MAXIMA,Groceries,25.50,EUR,D,record-123,CARD\n"

	got, err := importStatement(
		ctx,
		store,
		user.ID,
		" statement.csv ",
		strings.NewReader(rawInput),
		time.UTC,
	)
	if err != nil {
		deleteTestUser(t, store, user.ID)
		t.Fatalf("importStatement() returned an unexpected error: %v", err)
	}

	t.Cleanup(func() {
		for _, transaction := range got.Stored.Transactions {
			deleteTestTransaction(t, store, transaction.ID)
		}

		deleteTestStatement(t, store, got.Stored.Statement.ID)
		deleteTestUser(t, store, user.ID)
	})

	wantSummary := statement.Summary{
		ImportedRows: 1,
		UniqueRows:   1,
	}

	if got.Summary != wantSummary {
		t.Errorf("Summary = %+v, want %+v", got.Summary, wantSummary)
	}

	if got.Stored.Statement.ID <= 0 {
		t.Errorf("statement ID = %d, want positive", got.Stored.Statement.ID)
	}

	if got.Stored.Statement.UserID != user.ID {
		t.Errorf("statement UserID = %d, want %d", got.Stored.Statement.UserID, user.ID)
	}

	if got.Stored.Statement.Source != statement.Swedbank {
		t.Errorf("statement Source = %q, want %q", got.Stored.Statement.Source, statement.Swedbank)
	}

	if got.Stored.Statement.OriginalFilename != "statement.csv" {
		t.Errorf("statement filename = %q, want %q", got.Stored.Statement.OriginalFilename, "statement.csv")
	}

	wantStatementFingerprint := Fingerprint(sha256.Sum256([]byte(rawInput)))

	if got.Stored.Statement.Fingerprint != wantStatementFingerprint {
		t.Errorf("statement Fingerprint = %x, want %x", got.Stored.Statement.Fingerprint, wantStatementFingerprint)
	}

	wantHeader := []string{
		"Account No",
		"Date",
		"Beneficiary",
		"Details",
		"Amount",
		"Currency",
		"D/K",
		"Record ID",
		"Code",
	}

	if !slices.Equal(got.Stored.Statement.RawHeader, wantHeader) {
		t.Errorf("statement RawHeader = %q, want %q", got.Stored.Statement.RawHeader, wantHeader)
	}

	if len(got.Stored.Transactions) != 1 {
		t.Fatalf("stored transaction count = %d, want 1", len(got.Stored.Transactions))
	}

	storedTransaction := got.Stored.Transactions[0]

	if storedTransaction.ID <= 0 {
		t.Errorf("transaction ID = %d, want positive", storedTransaction.ID)
	}

	if storedTransaction.StatementID != got.Stored.Statement.ID {
		t.Errorf("transaction StatementID = %d, want %d", storedTransaction.StatementID, got.Stored.Statement.ID)
	}

	if storedTransaction.Fingerprint == (Fingerprint{}) {
		t.Error("transaction Fingerprint is empty")
	}

	wantTransaction := finance.Transaction{
		Date: time.Date(2026, time.August, 29, 0, 0, 0, 0, time.UTC),
		Amount: finance.Money{
			Amount:   -2_550,
			Currency: finance.EUR,
		},
		Description:  "Groceries",
		Counterparty: "MAXIMA",
	}

	if storedTransaction.Transaction != wantTransaction {
		t.Errorf("stored Transaction = %+v, want %+v", storedTransaction.Transaction, wantTransaction)
	}

	wantRawRecord := []string{
		"LT123",
		"2026-08-29",
		"MAXIMA",
		"Groceries",
		"25.50",
		"EUR",
		"D",
		"record-123",
		"CARD",
	}

	if !slices.Equal(storedTransaction.RawRecord, wantRawRecord) {
		t.Errorf("stored RawRecord = %q, want %q", storedTransaction.RawRecord, wantRawRecord)
	}
}

func TestImportStatementReturnsPreparationErrorBeforeDatabaseAccess(t *testing.T) {
	store := newPostgresStore(nil)

	input := strings.NewReader(
		"Date,Description,Amount\n" +
			"2026-08-29,Groceries,-25.50\n",
	)

	got, err := importStatement(
		context.Background(),
		store,
		1,
		"unknown.csv",
		input,
		time.UTC,
	)
	if err == nil {
		t.Fatal("importStatement() error = nil, want non-nil")
	}

	if !errors.Is(err, statement.ErrUnknownStatementFormat) {
		t.Errorf("importStatement() error = %v, want ErrUnknownStatementFormat", err)
	}

	if got.Stored.Statement.ID != 0 {
		t.Errorf("stored statement ID = %d, want 0", got.Stored.Statement.ID)
	}

	if got.Stored.Transactions != nil {
		t.Errorf("stored transactions = %+v, want nil", got.Stored.Transactions)
	}

	if got.Summary != (statement.Summary{}) {
		t.Errorf("Summary = %+v, want zero Summary", got.Summary)
	}
}
