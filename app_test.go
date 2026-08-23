package main

import (
	"slices"
	"testing"
)

func TestNewAppCopiesInitialTransactions(t *testing.T) {
	input := []Transaction{
		{
			Description: "Groceries",
			Amount: Money{
				Amount:   -2_500,
				Currency: EUR,
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

	transaction := Transaction{
		Description: "Salary",
		Amount: Money{
			Amount:   100_000,
			Currency: EUR,
		},
	}

	app.addTransaction(transaction)

	want := []Transaction{
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

	transaction := Transaction{
		Description: "Groceries",
		Amount: Money{
			Amount:   -2_500,
			Currency: EUR,
		},
	}

	app.addTransaction(transaction)

	transaction.Description = "Changed original"
	transaction.Amount.Amount = -9_999

	want := []Transaction{
		{
			Description: "Groceries",
			Amount: Money{
				Amount:   -2_500,
				Currency: EUR,
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
	app := newApp([]Transaction{
		{
			Description: "Spotify",
			Amount: Money{
				Amount:   -999,
				Currency: EUR,
			},
		},
	})

	snapshot := app.transactionsSnapshot()

	snapshot[0].Description = "Changed snapshot"
	snapshot[0].Amount.Amount = -5_000
	snapshot = append(snapshot, Transaction{
		Description: "Added only to snapshot",
	})

	want := []Transaction{
		{
			Description: "Spotify",
			Amount: Money{
				Amount:   -999,
				Currency: EUR,
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

	app.addTransaction(Transaction{
		Description: "Salary",
		Amount: Money{
			Amount:   100_000,
			Currency: EUR,
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

	transaction := Transaction{
		Description: "",
		Amount: Money{
			Amount:   -1_000,
			Currency: EUR,
		},
	}

	app.addTransaction(transaction)

	want := []Transaction{
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
