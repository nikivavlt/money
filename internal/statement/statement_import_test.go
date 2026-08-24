package statement

import (
	"errors"
	"slices"
	"strings"
	"testing"
)

func TestImportStatementRevolut(t *testing.T) {
	input := strings.NewReader(
		"Type,Product,Started Date,Completed Date,Description,Amount,Fee,Currency,State,Balance\n" +
			`Card Payment,Current,2026-08-04 10:00:00,2026-08-04 10:01:00,"SHOP, VILNIUS",-12.34,0,EUR,COMPLETED,100.00` +
			"\n",
	)

	source, got, err := importStatement(input)
	if err != nil {
		t.Fatalf("importStatement() returned an unexpected error: %v", err)
	}

	if source != sourceRevolut {
		t.Errorf("importStatement() source = %q, want %q", source, sourceRevolut)
	}

	want := []importedTransaction{
		{
			source:          sourceRevolut,
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

	if !slices.Equal(got, want) {
		t.Errorf("importStatement() transactions = %+v, want %+v", got, want)
	}
}

func TestImportStatementSwedbank(t *testing.T) {
	input := strings.NewReader(
		"Account No,Date,Beneficiary,Details,Amount,Currency,D/K,Record ID,Code\n" +
			`LT123,2026-08-05,MAXIMA,"Card purchase, Vilnius",25.50,EUR,D,record-123,CARD` +
			"\n",
	)

	source, got, err := importStatement(input)
	if err != nil {
		t.Fatalf("importStatement() returned an unexpected error: %v", err)
	}

	if source != sourceSwedbank {
		t.Errorf("importStatement() source = %q, want %q", source, sourceSwedbank)
	}

	want := []importedTransaction{
		{
			source:           sourceSwedbank,
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

	if !slices.Equal(got, want) {
		t.Errorf("importStatement() transactions = %+v, want %+v", got, want)
	}
}

func TestImportStatementRejectsUnknownFormat(t *testing.T) {
	input := strings.NewReader(
		"Date,Description,Amount\n" +
			"2026-08-05,Groceries,-25.50\n",
	)

	source, transactions, err := importStatement(input)
	if err == nil {
		t.Fatal("importStatement() error = nil, want non-nil")
	}

	if !errors.Is(err, ErrUnknownStatementFormat) {
		t.Errorf(
			"importStatement() error = %v, want it to match ErrUnknownStatementFormat",
			err,
		)
	}

	if source != "" {
		t.Errorf("importStatement() source = %q, want empty source", source)
	}

	if transactions != nil {
		t.Errorf(
			"importStatement() transactions = %+v, want nil",
			transactions,
		)
	}
}

func TestImportStatementRejectsAmbiguousFormat(t *testing.T) {
	input := strings.NewReader(
		"Type,Product,Started Date,Completed Date,Description," +
			"Amount,Fee,Currency,State," +
			"Account No,Date,Beneficiary,Details,D/K,Record ID,Code\n",
	)

	source, transactions, err := importStatement(input)
	if err == nil {
		t.Fatal("importStatement() error = nil, want non-nil")
	}

	if !errors.Is(err, ErrAmbiguousStatementFormat) {
		t.Errorf(
			"importStatement() error = %v, want it to match ErrAmbiguousStatementFormat",
			err,
		)
	}

	if source != "" {
		t.Errorf("importStatement() source = %q, want empty source", source)
	}

	if transactions != nil {
		t.Errorf(
			"importStatement() transactions = %+v, want nil",
			transactions,
		)
	}
}

func TestImportStatementDoesNotReturnPartialResults(t *testing.T) {
	input := strings.NewReader(
		"Type,Product,Started Date,Completed Date,Description,Amount,Fee,Currency,State\n" +
			"Card Payment,Current,2026-08-04 10:00:00,2026-08-04 10:01:00,SHOP,-12.34,0,EUR,COMPLETED\n" +
			`Card Payment,Current,2026-08-05 10:00:00,2026-08-05 10:01:00,"unterminated`,
	)

	source, transactions, err := importStatement(input)
	if err == nil {
		t.Fatal("importStatement() error = nil, want non-nil")
	}

	if source != "" {
		t.Errorf("importStatement() source = %q, want empty source", source)
	}

	if transactions != nil {
		t.Errorf(
			"importStatement() returned partial transactions: %+v",
			transactions,
		)
	}
}
