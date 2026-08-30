package main

import (
	"bytes"
	"testing"
	"time"

	"money/internal/finance"
	"money/internal/statement"
)

func TestWriteTransactions(t *testing.T) {
	var output bytes.Buffer

	transactions := []listedTransaction{
		{
			ID:               11,
			StatementID:      5,
			Source:           statement.Swedbank,
			OriginalFilename: "statement.csv",
			Transaction: finance.Transaction{
				Date: time.Date(2026, time.August, 29, 0, 0, 0, 0, time.UTC),
				Amount: finance.Money{
					Amount:   -2550,
					Currency: finance.EUR,
				},
				Description:  "Groceries",
				Counterparty: "MAXIMA",
			},
			MerchantName:   "Maxima",
			CategoryName:   "Groceries",
			Classification: "rule",
		},
		{
			ID:               10,
			StatementID:      5,
			Source:           statement.Swedbank,
			OriginalFilename: "statement.csv",
			Transaction: finance.Transaction{
				Date: time.Date(2026, time.August, 28, 0, 0, 0, 0, time.UTC),
				Amount: finance.Money{
					Amount:   100_000,
					Currency: finance.EUR,
				},
				Description: "Salary",
			},
		},
	}

	err := writeTransactions(&output, transactions)
	if err != nil {
		t.Fatalf("writeTransactions() returned an unexpected error: %v", err)
	}

	want := "" +
		"ID\tDATE\tAMOUNT\tDESCRIPTION\tCOUNTERPARTY\tMERCHANT\tCATEGORY\tCLASSIFIED_BY\tSOURCE\tSTATEMENT\tFILE\n" +
		"11\t2026-08-29\t-25.50 EUR\t\"Groceries\"\t\"MAXIMA\"\t\"Maxima\"\t\"Groceries\"\trule\tswedbank\t5\t\"statement.csv\"\n" +
		"10\t2026-08-28\t1000.00 EUR\t\"Salary\"\t\"\"\t\"\"\t\"\"\t\tswedbank\t5\t\"statement.csv\"\n"

	if output.String() != want {
		t.Errorf("writeTransactions() output = %q, want %q", output.String(), want)
	}
}

func TestWriteTransactionsEmpty(t *testing.T) {
	var output bytes.Buffer

	err := writeTransactions(&output, nil)
	if err != nil {
		t.Fatalf("writeTransactions() returned an unexpected error: %v", err)
	}

	if output.String() != "No transactions.\n" {
		t.Errorf("writeTransactions() output = %q, want %q", output.String(), "No transactions.\n")
	}
}
