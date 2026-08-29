package statement

import (
	"errors"
	"slices"
	"strings"
	"testing"
)

func TestReadImportedStatementRevolut(t *testing.T) {
	input := strings.NewReader(
		"Type,Product,Started Date,Completed Date,Description,Amount,Fee,Currency,State,Balance\n" +
			`Card Payment,Current,2026-08-04 10:00:00,2026-08-04 10:01:00,"SHOP, VILNIUS",-12.34,0,EUR,COMPLETED,100.00` +
			"\n",
	)

	imported, err := readImportedStatement(input)
	if err != nil {
		t.Fatalf("readImportedStatement() returned an unexpected error: %v", err)
	}

	if imported.source != Revolut {
		t.Errorf("source = %q, want %q", imported.source, Revolut)
	}

	want := []importedTransaction{
		{
			source:          Revolut,
			accountText:     "Current",
			occurredAtText:  "2026-08-04 10:00:00",
			completedAtText: "2026-08-04 10:01:00",
			amountText:      "-12.34",
			feeText:         "0",
			currencyText:    "EUR",
			rawDescription:  "SHOP, VILNIUS",
			typeText:        "Card Payment",
			stateText:       "COMPLETED",
		},
	}

	if !slices.Equal(imported.transactions, want) {
		t.Errorf("transactions = %+v, want %+v", imported.transactions, want)
	}

	if len(imported.rawRecords) != 1 {
		t.Fatalf("raw record count = %d, want 1", len(imported.rawRecords))
	}
}

func TestReadImportedStatementSwedbank(t *testing.T) {
	input := strings.NewReader(
		"Account No,Date,Beneficiary,Details,Amount,Currency,D/K,Record ID,Code\n" +
			`LT123,2026-08-05,MAXIMA,"Card purchase, Vilnius",25.50,EUR,D,record-123,CARD` +
			"\n",
	)

	imported, err := readImportedStatement(input)
	if err != nil {
		t.Fatalf("readImportedStatement() returned an unexpected error: %v", err)
	}

	if imported.source != Swedbank {
		t.Errorf("source = %q, want %q", imported.source, Swedbank)
	}

	want := []importedTransaction{
		{
			source:           Swedbank,
			accountText:      "LT123",
			occurredAtText:   "2026-08-05",
			amountText:       "25.50",
			currencyText:     "EUR",
			directionText:    "D",
			rawDescription:   "Card purchase, Vilnius",
			counterpartyText: "MAXIMA",
			externalID:       "record-123",
			typeText:         "CARD",
		},
	}

	if !slices.Equal(imported.transactions, want) {
		t.Errorf("transactions = %+v, want %+v", imported.transactions, want)
	}

	if len(imported.rawRecords) != 1 {
		t.Fatalf("raw record count = %d, want 1", len(imported.rawRecords))
	}
}

func TestReadImportedStatementPreservesRawData(t *testing.T) {
	input := strings.NewReader(
		"Type,Product,Started Date,Completed Date,Description,Amount,Fee,Currency,State\n" +
			`Card Payment,Current,2026-08-04 10:00:00,2026-08-04 10:01:00,"SHOP, VILNIUS",-12.34,0,EUR,COMPLETED` +
			"\n",
	)

	imported, err := readImportedStatement(input)
	if err != nil {
		t.Fatalf("readImportedStatement() returned an unexpected error: %v", err)
	}

	wantHeader := []string{
		"Type",
		"Product",
		"Started Date",
		"Completed Date",
		"Description",
		"Amount",
		"Fee",
		"Currency",
		"State",
	}

	if !slices.Equal(imported.rawHeader, wantHeader) {
		t.Errorf("raw header = %q, want %q", imported.rawHeader, wantHeader)
	}

	if len(imported.rawRecords) != 1 {
		t.Fatalf("raw record count = %d, want 1", len(imported.rawRecords))
	}

	wantRecord := []string{
		"Card Payment",
		"Current",
		"2026-08-04 10:00:00",
		"2026-08-04 10:01:00",
		"SHOP, VILNIUS",
		"-12.34",
		"0",
		"EUR",
		"COMPLETED",
	}

	if !slices.Equal(imported.rawRecords[0], wantRecord) {
		t.Errorf("raw record = %q, want %q", imported.rawRecords[0], wantRecord)
	}

	if len(imported.transactions) != len(imported.rawRecords) {
		t.Errorf("transaction count = %d, raw record count = %d", len(imported.transactions), len(imported.rawRecords))
	}
}

func TestReadImportedStatementRejectsUnknownFormat(t *testing.T) {
	input := strings.NewReader(
		"Date,Description,Amount\n" +
			"2026-08-05,Groceries,-25.50\n",
	)

	imported, err := readImportedStatement(input)
	if err == nil {
		t.Fatal("readImportedStatement() error = nil, want non-nil")
	}

	if !errors.Is(err, ErrUnknownStatementFormat) {
		t.Errorf("readImportedStatement() error = %v, want ErrUnknownStatementFormat", err)
	}

	if !importedStatementIsZero(imported) {
		t.Errorf("readImportedStatement() = %+v, want zero result", imported)
	}
}

func TestReadImportedStatementRejectsAmbiguousFormat(t *testing.T) {
	input := strings.NewReader(
		"Type,Product,Started Date,Completed Date,Description," +
			"Amount,Fee,Currency,State," +
			"Account No,Date,Beneficiary,Details,D/K,Record ID,Code\n",
	)

	imported, err := readImportedStatement(input)
	if err == nil {
		t.Fatal("readImportedStatement() error = nil, want non-nil")
	}

	if !errors.Is(err, ErrAmbiguousStatementFormat) {
		t.Errorf("readImportedStatement() error = %v, want ErrAmbiguousStatementFormat", err)
	}

	if !importedStatementIsZero(imported) {
		t.Errorf("readImportedStatement() = %+v, want zero result", imported)
	}
}

func TestReadImportedStatementDoesNotReturnPartialResults(t *testing.T) {
	input := strings.NewReader(
		"Type,Product,Started Date,Completed Date,Description,Amount,Fee,Currency,State\n" +
			"Card Payment,Current,2026-08-04 10:00:00,2026-08-04 10:01:00,SHOP,-12.34,0,EUR,COMPLETED\n" +
			`Card Payment,Current,2026-08-05 10:00:00,2026-08-05 10:01:00,"unterminated`,
	)

	imported, err := readImportedStatement(input)
	if err == nil {
		t.Fatal("readImportedStatement() error = nil, want non-nil")
	}

	if !importedStatementIsZero(imported) {
		t.Errorf("readImportedStatement() returned partial result: %+v", imported)
	}
}

func importedStatementIsZero(value importedStatement) bool {
	return value.source == "" &&
		value.rawHeader == nil &&
		value.transactions == nil &&
		value.rawRecords == nil
}
