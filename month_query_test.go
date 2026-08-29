package main

import (
	"context"
	"strings"
	"testing"
	"time"

	"money/internal/finance"
)

func TestSummarizeMonthByUserRejectsInvalidInput(t *testing.T) {
	store := newPostgresStore(nil)

	got, err := store.summarizeMonthByUser(context.Background(), 0, time.Now())
	if err == nil {
		t.Fatal("summarizeMonthByUser() user error = nil, want non-nil")
	}

	if got != nil {
		t.Errorf("summarizeMonthByUser() = %+v, want nil", got)
	}

	got, err = store.summarizeMonthByUser(context.Background(), 1, time.Time{})
	if err == nil {
		t.Fatal("summarizeMonthByUser() month error = nil, want non-nil")
	}

	if got != nil {
		t.Errorf("summarizeMonthByUser() = %+v, want nil", got)
	}
}

func TestSummarizeMonthByUserAggregatesByCurrency(t *testing.T) {
	ctx, store := openTestPostgresStore(t)

	firstUser, err := store.createUser(ctx, "Monthly Summary First User")
	if err != nil {
		t.Fatalf("create first test user: %v", err)
	}

	t.Cleanup(func() {
		deleteTestUser(t, store, firstUser.ID)
	})

	secondUser, err := store.createUser(ctx, "Monthly Summary Second User")
	if err != nil {
		t.Fatalf("create second test user: %v", err)
	}

	t.Cleanup(func() {
		deleteTestUser(t, store, secondUser.ID)
	})

	firstUserInput := "" +
		"Account No,Date,Beneficiary,Details,Amount,Currency,D/K,Record ID,Code\n" +
		"LT-FIRST,2026-07-31,EMPLOYER,Previous salary,500.00,EUR,K,record-1,TRANSFER\n" +
		"LT-FIRST,2026-08-01,EMPLOYER,Salary,1000.00,EUR,K,record-2,TRANSFER\n" +
		"LT-FIRST,2026-08-15,MAXIMA,Groceries,25.50,EUR,D,record-3,CARD\n" +
		"LT-FIRST,2026-08-31,SHOP,Refund,5.00,EUR,K,record-4,REFUND\n" +
		"LT-FIRST,2026-09-01,SHOP,Next month purchase,10.00,EUR,D,record-5,CARD\n" +
		"LT-FIRST,2026-08-20,CLIENT,USD income,50.00,USD,K,record-6,TRANSFER\n" +
		"LT-FIRST,2026-08-21,SHOP,USD purchase,12.34,USD,D,record-7,CARD\n"

	firstImport, err := importStatement(
		ctx,
		store,
		firstUser.ID,
		"monthly-first-user.csv",
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
		"LT-SECOND,2026-08-10,EMPLOYER,Other user salary,9999.99,EUR,K,record-1,TRANSFER\n"

	secondImport, err := importStatement(
		ctx,
		store,
		secondUser.ID,
		"monthly-second-user.csv",
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

	requestedMonth := time.Date(
		2026,
		time.August,
		20,
		15,
		30,
		0,
		0,
		time.FixedZone("test", 3*60*60),
	)

	got, err := store.summarizeMonthByUser(
		ctx,
		firstUser.ID,
		requestedMonth,
	)
	if err != nil {
		t.Fatalf("summarizeMonthByUser() returned an unexpected error: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("summary count = %d, want 2", len(got))
	}

	monthStart := time.Date(
		2026,
		time.August,
		1,
		0,
		0,
		0,
		0,
		time.UTC,
	)

	wantEUR := monthlyCashFlow{
		Month:            monthStart,
		Currency:         finance.EUR,
		TransactionCount: 3,
		Income: finance.Money{
			Amount:   100_500,
			Currency: finance.EUR,
		},
		Expenses: finance.Money{
			Amount:   2_550,
			Currency: finance.EUR,
		},
		Savings: finance.Money{
			Amount:   97_950,
			Currency: finance.EUR,
		},
	}

	wantUSD := monthlyCashFlow{
		Month:            monthStart,
		Currency:         finance.USD,
		TransactionCount: 2,
		Income: finance.Money{
			Amount:   5_000,
			Currency: finance.USD,
		},
		Expenses: finance.Money{
			Amount:   1_234,
			Currency: finance.USD,
		},
		Savings: finance.Money{
			Amount:   3_766,
			Currency: finance.USD,
		},
	}

	if got[0] != wantEUR {
		t.Errorf("EUR summary = %+v, want %+v", got[0], wantEUR)
	}

	if got[1] != wantUSD {
		t.Errorf("USD summary = %+v, want %+v", got[1], wantUSD)
	}

	empty, err := store.summarizeMonthByUser(
		ctx,
		firstUser.ID,
		time.Date(2026, time.October, 1, 0, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatalf("summarize empty month: %v", err)
	}

	if empty != nil {
		t.Errorf("empty month summary = %+v, want nil", empty)
	}
}
