package main

import (
	"context"
	"strings"
	"testing"
	"time"

	"money/internal/finance"
	"money/internal/statement"
)

func TestListTransactionsByUserRejectsInvalidUserID(t *testing.T) {
	store := newPostgresStore(nil)

	for _, userID := range []int64{0, -1} {
		got, err := store.listTransactionsByUser(context.Background(), userID)
		if err == nil {
			t.Fatalf("listTransactionsByUser(%d) error = nil, want non-nil", userID)
		}

		if err.Error() != "transaction user ID must be positive" {
			t.Errorf("listTransactionsByUser(%d) error = %q, want %q", userID, err, "transaction user ID must be positive")
		}

		if got != nil {
			t.Errorf("listTransactionsByUser(%d) = %+v, want nil", userID, got)
		}
	}
}

func TestListTransactionsByUserReturnsOnlyUserTransactions(t *testing.T) {
	ctx, store := openTestPostgresStore(t)

	firstUser, err := store.createUser(ctx, "Transaction List First User")
	if err != nil {
		t.Fatalf("create first test user: %v", err)
	}

	t.Cleanup(func() {
		deleteTestUser(t, store, firstUser.ID)
	})

	secondUser, err := store.createUser(ctx, "Transaction List Second User")
	if err != nil {
		t.Fatalf("create second test user: %v", err)
	}

	t.Cleanup(func() {
		deleteTestUser(t, store, secondUser.ID)
	})

	firstUserInput := "" +
		"Account No,Date,Beneficiary,Details,Amount,Currency,D/K,Record ID,Code\n" +
		"LT-FIRST,2026-08-28,EMPLOYER,Salary,1000.00,EUR,K,first-record-1,TRANSFER\n" +
		"LT-FIRST,2026-08-29,MAXIMA,Groceries,25.50,EUR,D,first-record-2,CARD\n"

	firstImport, err := importStatement(
		ctx,
		store,
		firstUser.ID,
		"first-user.csv",
		strings.NewReader(firstUserInput),
		time.UTC,
	)
	if err != nil {
		t.Fatalf("import first user statement: %v", err)
	}

	t.Cleanup(func() {
		for _, transaction := range firstImport.Stored.Transactions {
			deleteTestTransaction(t, store, transaction.ID)
		}

		deleteTestStatement(t, store, firstImport.Stored.Statement.ID)
	})

	secondUserInput := "" +
		"Account No,Date,Beneficiary,Details,Amount,Currency,D/K,Record ID,Code\n" +
		"LT-SECOND,2026-08-30,SHOP,Other purchase,99.99,EUR,D,second-record-1,CARD\n"

	secondImport, err := importStatement(
		ctx,
		store,
		secondUser.ID,
		"second-user.csv",
		strings.NewReader(secondUserInput),
		time.UTC,
	)
	if err != nil {
		t.Fatalf("import second user statement: %v", err)
	}

	t.Cleanup(func() {
		for _, transaction := range secondImport.Stored.Transactions {
			deleteTestTransaction(t, store, transaction.ID)
		}

		deleteTestStatement(t, store, secondImport.Stored.Statement.ID)
	})

	got, err := store.listTransactionsByUser(ctx, firstUser.ID)
	if err != nil {
		t.Fatalf("listTransactionsByUser() returned an unexpected error: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("transaction count = %d, want 2", len(got))
	}

	newer := got[0]
	older := got[1]

	wantNewerID := firstImport.Stored.Transactions[1].ID
	wantOlderID := firstImport.Stored.Transactions[0].ID

	if newer.ID != wantNewerID {
		t.Errorf("newer transaction ID = %d, want %d", newer.ID, wantNewerID)
	}

	if older.ID != wantOlderID {
		t.Errorf("older transaction ID = %d, want %d", older.ID, wantOlderID)
	}

	if !newer.Transaction.Date.Equal(time.Date(2026, time.August, 29, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("newer transaction date = %v, want 2026-08-29", newer.Transaction.Date)
	}

	if !older.Transaction.Date.Equal(time.Date(2026, time.August, 28, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("older transaction date = %v, want 2026-08-28", older.Transaction.Date)
	}

	if newer.Transaction.Amount != (finance.Money{Amount: -2_550, Currency: finance.EUR}) {
		t.Errorf("newer transaction amount = %+v, want -25.50 EUR", newer.Transaction.Amount)
	}

	if older.Transaction.Amount != (finance.Money{Amount: 100_000, Currency: finance.EUR}) {
		t.Errorf("older transaction amount = %+v, want 1000.00 EUR", older.Transaction.Amount)
	}

	if newer.Transaction.Description != "Groceries" {
		t.Errorf("newer description = %q, want %q", newer.Transaction.Description, "Groceries")
	}

	if newer.Transaction.Counterparty != "MAXIMA" {
		t.Errorf("newer counterparty = %q, want %q", newer.Transaction.Counterparty, "MAXIMA")
	}

	if newer.Source != statement.Swedbank {
		t.Errorf("newer source = %q, want %q", newer.Source, statement.Swedbank)
	}

	if newer.OriginalFilename != "first-user.csv" {
		t.Errorf("newer filename = %q, want %q", newer.OriginalFilename, "first-user.csv")
	}

	if newer.StatementID != firstImport.Stored.Statement.ID {
		t.Errorf("newer StatementID = %d, want %d", newer.StatementID, firstImport.Stored.Statement.ID)
	}

	for _, transaction := range got {
		if transaction.ID == secondImport.Stored.Transactions[0].ID {
			t.Errorf("list contains second user's transaction ID %d", transaction.ID)
		}
	}
}
