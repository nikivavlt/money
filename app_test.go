package main

import (
	"money/internal/finance"
	"slices"
	"testing"
)

func TestNewAppCopiesInitialTransactions(t *testing.T) {
	input := []finance.Transaction{
		{
			Description: "Groceries",
			Amount: finance.Money{
				Amount:   -2_500,
				Currency: finance.EUR,
			},
		},
	}

	want := slices.Clone(input)
	app := newApp(input)

	input[0].Description = "Changed outside application"
	input[0].Amount.Amount = -9_999

	got := app.transactionsSnapshot()

	if !slices.Equal(got, want) {
		t.Errorf(
			"transactionsSnapshot() = %+v, want %+v",
			got,
			want,
		)
	}
}

func TestApplicationAddTransaction(t *testing.T) {
	app := newApp(nil)

	transaction := finance.Transaction{
		Description: "Salary",
		Amount: finance.Money{
			Amount:   100_000,
			Currency: finance.EUR,
		},
	}

	app.addTransaction(transaction)

	want := []finance.Transaction{
		transaction,
	}

	got := app.transactionsSnapshot()

	if !slices.Equal(got, want) {
		t.Errorf(
			"transactionsSnapshot() after addTransaction() = %+v, want %+v",
			got,
			want,
		)
	}
}

func TestApplicationAddTransactionCopiesValue(t *testing.T) {
	app := newApp(nil)

	transaction := finance.Transaction{
		Description: "Groceries",
		Amount: finance.Money{
			Amount:   -2_500,
			Currency: finance.EUR,
		},
	}

	app.addTransaction(transaction)

	transaction.Description = "Changed original"
	transaction.Amount.Amount = -9_999

	want := []finance.Transaction{
		{
			Description: "Groceries",
			Amount: finance.Money{
				Amount:   -2_500,
				Currency: finance.EUR,
			},
		},
	}

	got := app.transactionsSnapshot()

	if !slices.Equal(got, want) {
		t.Errorf(
			"transactionsSnapshot() = %+v, want %+v",
			got,
			want,
		)
	}
}

func TestApplicationTransactionsSnapshotIsIndependent(t *testing.T) {
	app := newApp([]finance.Transaction{
		{
			Description: "Spotify",
			Amount: finance.Money{
				Amount:   -999,
				Currency: finance.EUR,
			},
		},
	})

	snapshot := app.transactionsSnapshot()

	snapshot[0].Description = "Changed snapshot"
	snapshot[0].Amount.Amount = -5_000
	snapshot = append(snapshot, finance.Transaction{
		Description: "Added only to snapshot",
	})

	want := []finance.Transaction{
		{
			Description: "Spotify",
			Amount: finance.Money{
				Amount:   -999,
				Currency: finance.EUR,
			},
		},
	}

	got := app.transactionsSnapshot()

	if !slices.Equal(got, want) {
		t.Errorf(
			"modifying snapshot changed application: got %+v, want %+v",
			got,
			want,
		)
	}
}

func TestNewAppWithNilTransactions(t *testing.T) {
	app := newApp(nil)

	if app == nil {
		t.Fatal("newApp(nil) returned nil")
	}

	if got := len(app.transactionsSnapshot()); got != 0 {
		t.Errorf(
			"newApp(nil) contains %d transactions, want 0",
			got,
		)
	}

	app.addTransaction(finance.Transaction{
		Description: "Salary",
		Amount: finance.Money{
			Amount:   100_000,
			Currency: finance.EUR,
		},
	})

	if got := len(app.transactionsSnapshot()); got != 1 {
		t.Errorf(
			"after addTransaction(), application contains %d transactions, want 1",
			got,
		)
	}
}

func TestApplicationPreservesTransactionWithEmptyDescription(t *testing.T) {
	app := newApp(nil)

	transaction := finance.Transaction{
		Description: "",
		Amount: finance.Money{
			Amount:   -1_000,
			Currency: finance.EUR,
		},
	}

	app.addTransaction(transaction)

	want := []finance.Transaction{
		transaction,
	}

	got := app.transactionsSnapshot()

	if !slices.Equal(got, want) {
		t.Errorf(
			"transactionsSnapshot() = %+v, want %+v",
			got,
			want,
		)
	}
}
