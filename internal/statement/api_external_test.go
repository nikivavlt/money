package statement_test

import (
	"crypto/sha256"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"money/internal/finance"
	"money/internal/statement"
)

func TestPrepareSwedbankStatement(t *testing.T) {
	rawInput := "" +
		"Account No,Date,Beneficiary,Details,Amount,Currency,D/K,Record ID,Code\n" +
		"LT123,2026-08-05,MAXIMA,Card purchase,25.50,EUR,D,record-123,CARD\n"

	location := time.FixedZone("test", 2*60*60)

	got, err := statement.Prepare(strings.NewReader(rawInput), location)
	if err != nil {
		t.Fatalf("statement.Prepare() returned an unexpected error: %v", err)
	}

	if got.Source != statement.Swedbank {
		t.Errorf("Source = %q, want %q", got.Source, statement.Swedbank)
	}

	wantFingerprint := statement.Fingerprint(sha256.Sum256([]byte(rawInput)))

	if got.Fingerprint != wantFingerprint {
		t.Errorf("Fingerprint = %x, want %x", got.Fingerprint, wantFingerprint)
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

	if !slices.Equal(got.RawHeader, wantHeader) {
		t.Errorf("RawHeader = %q, want %q", got.RawHeader, wantHeader)
	}

	wantSummary := statement.Summary{
		ImportedRows: 1,
		UniqueRows:   1,
	}

	if got.Summary != wantSummary {
		t.Errorf("Summary = %+v, want %+v", got.Summary, wantSummary)
	}

	if len(got.Transactions) != 1 {
		t.Fatalf("transaction count = %d, want 1", len(got.Transactions))
	}

	preparedTransaction := got.Transactions[0]

	if preparedTransaction.Fingerprint == (statement.Fingerprint{}) {
		t.Error("transaction Fingerprint is empty")
	}

	wantRawRecord := []string{
		"LT123",
		"2026-08-05",
		"MAXIMA",
		"Card purchase",
		"25.50",
		"EUR",
		"D",
		"record-123",
		"CARD",
	}

	if !slices.Equal(preparedTransaction.RawRecord, wantRawRecord) {
		t.Errorf("transaction RawRecord = %q, want %q", preparedTransaction.RawRecord, wantRawRecord)
	}

	transaction := preparedTransaction.Transaction

	wantDate := time.Date(2026, time.August, 5, 0, 0, 0, 0, location)

	if !transaction.Date.Equal(wantDate) {
		t.Errorf("transaction date = %v, want %v", transaction.Date, wantDate)
	}

	wantMoney := finance.Money{
		Amount:   -2_550,
		Currency: finance.EUR,
	}

	if transaction.Amount != wantMoney {
		t.Errorf("transaction amount = %+v, want %+v", transaction.Amount, wantMoney)
	}

	if transaction.Description != "Card purchase" {
		t.Errorf("Description = %q, want %q", transaction.Description, "Card purchase")
	}

	if transaction.Counterparty != "MAXIMA" {
		t.Errorf("Counterparty = %q, want %q", transaction.Counterparty, "MAXIMA")
	}
}

func TestPreparePreservesUnknownFormatError(t *testing.T) {
	input := strings.NewReader(
		"Date,Description,Amount\n" +
			"2026-08-05,Groceries,-25.50\n",
	)

	got, err := statement.Prepare(
		input,
		time.UTC,
	)
	if err == nil {
		t.Fatal("statement.Prepare() error = nil, want non-nil")
	}

	if got.Source != "" ||
		got.Fingerprint != (statement.Fingerprint{}) ||
		got.RawHeader != nil ||
		got.Transactions != nil ||
		got.Summary != (statement.Summary{}) {
		t.Errorf("statement.Prepare() result = %+v, want zero result", got)
	}

	if !errors.Is(
		err,
		statement.ErrUnknownStatementFormat,
	) {
		t.Errorf(
			"statement.Prepare() error = %v, want ErrUnknownStatementFormat",
			err,
		)
	}

	if got.Source != "" ||
		got.Transactions != nil ||
		got.Summary != (statement.Summary{}) {
		t.Errorf(
			"statement.Prepare() result = %+v, want zero result",
			got,
		)
	}
}
