package statement

import (
	"errors"
	"money/internal/finance"
	"slices"
	"strings"
	"testing"
	"time"
)

func pipelineTestLocation() *time.Location {
	return time.FixedZone("test", 2*60*60)
}

func TestPrepareStatementImportRevolut(t *testing.T) {
	input := strings.NewReader(
		"Type,Product,Started Date,Completed Date,Description,Amount,Fee,Currency,State\n" +
			`Card Payment,Current,2026-08-04 10:00:00,2026-08-04 10:01:00,"SHOP, VILNIUS",-12.34,0,EUR,COMPLETED` +
			"\n",
	)

	location := pipelineTestLocation()

	got, err := prepareStatementImport(input, location)
	if err != nil {
		t.Fatalf("prepareStatementImport() returned an unexpected error: %v", err)
	}

	if got.source != Revolut {
		t.Errorf("source = %q, want %q", got.source, Revolut)
	}

	if len(got.transactions) != 1 {
		t.Fatalf("transaction count = %d, want 1", len(got.transactions))
	}

	wantTransactions := []Transaction{
		{
			Date: time.Date(
				2026, time.August, 4,
				0, 0, 0, 0,
				location,
			),
			Amount: finance.Money{
				Amount:   -1_234,
				Currency: finance.EUR,
			},
			Description: "SHOP, VILNIUS",
		},
	}

	gotTransactions := normalizedPreparedTransactions(got.transactions)

	if !slices.Equal(gotTransactions, wantTransactions) {
		t.Errorf("transactions = %+v, want %+v", gotTransactions, wantTransactions)
	}

	wantRawRecord := []string{
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

	if !slices.Equal(got.transactions[0].rawRecord, wantRawRecord) {
		t.Errorf("transaction raw record = %q, want %q", got.transactions[0].rawRecord, wantRawRecord)
	}

	wantSummary := importSummary{
		importedRows: 1,
		uniqueRows:   1,
	}

	if got.summary != wantSummary {
		t.Errorf("summary = %+v, want %+v", got.summary, wantSummary)
	}

	if len(got.duplicates) != 0 {
		t.Errorf("duplicate count = %d, want 0", len(got.duplicates))
	}

	if len(got.conflicts) != 0 {
		t.Errorf("conflict count = %d, want 0", len(got.conflicts))
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

	if !slices.Equal(got.rawHeader, wantHeader) {
		t.Errorf("raw header = %q, want %q", got.rawHeader, wantHeader)
	}
}

func TestPrepareStatementImportSwedbank(t *testing.T) {
	input := strings.NewReader(
		"Account No,Date,Beneficiary,Details,Amount,Currency,D/K,Record ID,Code\n" +
			`LT123,2026-08-05,MAXIMA,"Card purchase, Vilnius",25.50,EUR,D,record-123,CARD` +
			"\n",
	)

	location := pipelineTestLocation()

	got, err := prepareStatementImport(input, location)
	if err != nil {
		t.Fatalf(
			"prepareStatementImport() returned an unexpected error: %v",
			err,
		)
	}

	if got.source != Swedbank {
		t.Errorf(
			"source = %q, want %q",
			got.source,
			Swedbank,
		)
	}

	wantTransactions := []Transaction{
		{
			Date: time.Date(
				2026, time.August, 5,
				0, 0, 0, 0,
				location,
			),
			Amount: finance.Money{
				Amount:   -2_550,
				Currency: finance.EUR,
			},
			Description:  "Card purchase, Vilnius",
			Counterparty: "MAXIMA",
		},
	}

	gotTransactions := normalizedPreparedTransactions(got.transactions)

	if !slices.Equal(gotTransactions, wantTransactions) {
		t.Errorf("transactions = %+v, want %+v", gotTransactions, wantTransactions)
	}

	if !slices.Equal(gotTransactions, wantTransactions) {
		t.Errorf(
			"transactions = %+v, want %+v",
			got.transactions,
			wantTransactions,
		)
	}

	wantSummary := importSummary{
		importedRows: 1,
		uniqueRows:   1,
	}

	if got.summary != wantSummary {
		t.Errorf(
			"summary = %+v, want %+v",
			got.summary,
			wantSummary,
		)
	}

	preparedTransaction := got.transactions[0]

	wantIdentity := importedTransactionIdentity(importedTransaction{
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
	})

	if preparedTransaction.identity != wantIdentity {
		t.Errorf("transaction identity = %x, want %x", preparedTransaction.identity.digest, wantIdentity.digest)
	}

	wantRawRecord := []string{
		"LT123",
		"2026-08-05",
		"MAXIMA",
		"Card purchase, Vilnius",
		"25.50",
		"EUR",
		"D",
		"record-123",
		"CARD",
	}

	if !slices.Equal(preparedTransaction.rawRecord, wantRawRecord) {
		t.Errorf("transaction raw record = %q, want %q", preparedTransaction.rawRecord, wantRawRecord)
	}
}

func TestPrepareStatementImportSkipsDuplicateRows(t *testing.T) {
	input := strings.NewReader(
		"Account No,Date,Beneficiary,Details,Amount,Currency,D/K,Record ID,Code\n" +
			"LT123,2026-08-05,MAXIMA,Card purchase,25.50,EUR,D,record-123,CARD\n" +
			"LT123,2026-08-05,MAXIMA,Card purchase,25.50,EUR,D,record-123,CARD\n",
	)

	got, err := prepareStatementImport(input, pipelineTestLocation())
	if err != nil {
		t.Fatalf(
			"prepareStatementImport() returned an unexpected error: %v",
			err,
		)
	}

	if len(got.transactions) != 1 {
		t.Errorf(
			"transaction count = %d, want 1",
			len(got.transactions),
		)
	}

	if len(got.duplicates) != 1 {
		t.Fatalf(
			"duplicate count = %d, want 1",
			len(got.duplicates),
		)
	}

	duplicate := got.duplicates[0]

	if duplicate.firstPosition != 1 {
		t.Errorf(
			"first duplicate position = %d, want 1",
			duplicate.firstPosition,
		)
	}

	if duplicate.duplicatePosition != 2 {
		t.Errorf(
			"repeated duplicate position = %d, want 2",
			duplicate.duplicatePosition,
		)
	}

	if duplicate.identityKind != identityExternalID {
		t.Errorf(
			"duplicate identity kind = %q, want %q",
			duplicate.identityKind,
			identityExternalID,
		)
	}

	wantSummary := importSummary{
		importedRows:  2,
		uniqueRows:    1,
		duplicateRows: 1,
	}

	if got.summary != wantSummary {
		t.Errorf(
			"summary = %+v, want %+v",
			got.summary,
			wantSummary,
		)
	}

	if len(got.transactions) != 1 {
		t.Errorf("normalized transaction count = %d, want 1", len(got.transactions))
	}
}

func TestPrepareStatementImportRejectsDuplicateConflict(t *testing.T) {
	input := strings.NewReader(
		"Account No,Date,Beneficiary,Details,Amount,Currency,D/K,Record ID,Code\n" +
			"LT123,2026-08-05,MAXIMA,Card purchase,25.50,EUR,D,record-123,CARD\n" +
			"LT123,2026-08-05,MAXIMA,Card purchase,999.99,EUR,D,record-123,CARD\n",
	)

	got, err := prepareStatementImport(input, pipelineTestLocation())
	if err == nil {
		t.Fatal("prepareStatementImport() error = nil, want non-nil")
	}

	if !errors.Is(err, ErrDuplicateConflict) {
		t.Errorf(
			"prepareStatementImport() error = %v, want it to match ErrDuplicateConflict",
			err,
		)
	}

	var conflictErr *duplicateConflictError
	if !errors.As(err, &conflictErr) {
		t.Fatalf(
			"prepareStatementImport() error type = %T, want *duplicateConflictError in its chain",
			err,
		)
	}

	if conflictErr.count != 1 {
		t.Errorf(
			"duplicateConflictError.count = %d, want 1",
			conflictErr.count,
		)
	}

	if got.transactions != nil {
		t.Errorf(
			"transactions = %+v, want nil on conflict",
			got.transactions,
		)
	}

	if len(got.conflicts) != 1 {
		t.Fatalf(
			"conflict count = %d, want 1",
			len(got.conflicts),
		)
	}

	conflict := got.conflicts[0]

	if conflict.firstPosition != 1 {
		t.Errorf(
			"conflict first position = %d, want 1",
			conflict.firstPosition,
		)
	}

	if conflict.conflictingPosition != 2 {
		t.Errorf(
			"conflicting position = %d, want 2",
			conflict.conflictingPosition,
		)
	}

	wantSummary := importSummary{
		importedRows: 2,
		uniqueRows:   1,
		conflictRows: 1,
	}

	if got.summary != wantSummary {
		t.Errorf(
			"summary = %+v, want %+v",
			got.summary,
			wantSummary,
		)
	}
}

func TestPrepareStatementImportRejectsInvalidAmount(t *testing.T) {
	input := strings.NewReader(
		"Account No,Date,Beneficiary,Details,Amount,Currency,D/K,Record ID,Code\n" +
			"LT123,2026-08-05,MAXIMA,Card purchase,not-money,EUR,D,record-123,CARD\n",
	)

	got, err := prepareStatementImport(input, pipelineTestLocation())
	if err == nil {
		t.Fatal("prepareStatementImport() error = nil, want non-nil")
	}

	if !strings.Contains(err.Error(), "normalize imported transaction 1") {
		t.Errorf(
			"prepareStatementImport() error = %q, want transaction position 1",
			err,
		)
	}

	if got.transactions != nil {
		t.Errorf(
			"transactions = %+v, want nil after normalization failure",
			got.transactions,
		)
	}

	wantSummary := importSummary{
		importedRows: 1,
		uniqueRows:   1,
	}

	if got.summary != wantSummary {
		t.Errorf(
			"summary = %+v, want %+v",
			got.summary,
			wantSummary,
		)
	}
}

func TestPrepareStatementImportPreservesPendingError(t *testing.T) {
	input := strings.NewReader(
		"Type,Product,Started Date,Completed Date,Description,Amount,Fee,Currency,State\n" +
			"Card Payment,Current,2026-08-04 10:00:00,,Pending payment,-10.00,0,EUR,PENDING\n",
	)

	got, err := prepareStatementImport(input, pipelineTestLocation())
	if err == nil {
		t.Fatal("prepareStatementImport() error = nil, want non-nil")
	}

	if !errors.Is(err, ErrPendingTransaction) {
		t.Errorf(
			"prepareStatementImport() error = %v, want it to match ErrPendingTransaction",
			err,
		)
	}

	if got.transactions != nil {
		t.Errorf(
			"transactions = %+v, want nil for a pending transaction",
			got.transactions,
		)
	}
}

func TestPrepareStatementImportPreservesUnknownFormatError(t *testing.T) {
	input := strings.NewReader(
		"Date,Description,Amount\n" +
			"2026-08-05,Groceries,-25.50\n",
	)

	got, err := prepareStatementImport(input, pipelineTestLocation())
	if err == nil {
		t.Fatal("prepareStatementImport() error = nil, want non-nil")
	}

	if !errors.Is(err, ErrUnknownStatementFormat) {
		t.Errorf(
			"prepareStatementImport() error = %v, want it to match ErrUnknownStatementFormat",
			err,
		)
	}

	if !preparedImportIsZero(got) {
		t.Errorf(
			"prepareStatementImport() result = %+v, want zero result",
			got,
		)
	}
}

func TestPrepareStatementImportRejectsNilLocation(t *testing.T) {
	input := strings.NewReader(
		"Account No,Date,Beneficiary,Details,Amount,Currency,D/K,Record ID,Code\n",
	)

	got, err := prepareStatementImport(input, nil)
	if err == nil {
		t.Fatal("prepareStatementImport() error = nil, want non-nil")
	}

	if !strings.Contains(err.Error(), "nil location") {
		t.Errorf(
			"prepareStatementImport() error = %q, want nil-location context",
			err,
		)
	}

	if !preparedImportIsZero(got) {
		t.Errorf(
			"prepareStatementImport() result = %+v, want zero result",
			got,
		)
	}
}

func preparedImportIsZero(value preparedStatementImport) bool {
	return value.source == "" &&
		value.rawHeader == nil &&
		value.transactions == nil &&
		value.duplicates == nil &&
		value.conflicts == nil &&
		value.summary == (importSummary{})
}

func normalizedPreparedTransactions(input []preparedTransaction) []Transaction {
	if input == nil {
		return nil
	}

	result := make([]Transaction, len(input))

	for index, transaction := range input {
		result[index] = transaction.normalized
	}

	return result
}
