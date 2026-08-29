package statement

import (
	"errors"
	"fmt"
	"io"
	"slices"
	"time"

	"money/internal/finance"
)

var ErrDuplicateConflict = errors.New("duplicate transaction conflict")

type duplicateConflictError struct {
	count int
}

func (e *duplicateConflictError) Error() string {
	return fmt.Sprintf("%d duplicate transaction conflicts detected", e.count)
}

func (e *duplicateConflictError) Unwrap() error {
	return ErrDuplicateConflict
}

type preparedTransaction struct {
	identity   transactionIdentity
	normalized finance.Transaction
	rawRecord  []string
}
type preparedStatementImport struct {
	source       Source
	rawHeader    []string
	transactions []preparedTransaction
	duplicates   []duplicateCandidate
	conflicts    []duplicateConflict
	summary      importSummary
}

func prepareStatementImport(input io.Reader, location *time.Location) (preparedStatementImport, error) {
	if location == nil {
		return preparedStatementImport{}, fmt.Errorf("prepare statement import: nil location")
	}

	imported, err := readImportedStatement(input)
	if err != nil {
		return preparedStatementImport{}, fmt.Errorf("prepare statement import: %w", err)
	}

	deduplication := deduplicateImportedTransactions(imported.transactions)
	summary := summarizeDeduplication(imported.transactions, deduplication)

	prepared := preparedStatementImport{
		source:     imported.source,
		rawHeader:  imported.rawHeader,
		duplicates: deduplication.duplicates,
		conflicts:  deduplication.conflicts,
		summary:    summary,
	}

	if !summary.isConsistent() {
		return prepared, fmt.Errorf("prepare statement import: inconsistent import summary")
	}

	if summary.hasConflicts() {
		return prepared, fmt.Errorf("prepare statement import: %w", &duplicateConflictError{count: summary.conflictRows})
	}

	normalized, err := normalizeImportedTransactions(deduplication.unique, location)
	if err != nil {
		return prepared, fmt.Errorf("prepare statement import: %w", err)
	}

	transactions, err := pairPreparedTransactions(imported, deduplication.unique, normalized)
	if err != nil {
		return prepared, fmt.Errorf("prepare statement import: %w", err)
	}

	prepared.transactions = transactions

	return prepared, nil
}

func pairPreparedTransactions(
	imported importedStatement,
	unique []importedTransaction,
	normalized []finance.Transaction,
) ([]preparedTransaction, error) {
	if len(imported.transactions) != len(imported.rawRecords) {
		return nil, fmt.Errorf("imported transaction count %d does not match raw record count %d", len(imported.transactions), len(imported.rawRecords))
	}

	if len(unique) != len(normalized) {
		return nil, fmt.Errorf("unique transaction count %d does not match normalized transaction count %d", len(unique), len(normalized))
	}

	firstRawRecord := make(map[transactionIdentity][]string, len(imported.transactions))

	for index, transaction := range imported.transactions {
		identity := importedTransactionIdentity(transaction)

		if _, exists := firstRawRecord[identity]; exists {
			continue
		}

		firstRawRecord[identity] = imported.rawRecords[index]
	}

	result := make([]preparedTransaction, len(unique))

	for index, transaction := range unique {
		identity := importedTransactionIdentity(transaction)

		rawRecord, exists := firstRawRecord[identity]
		if !exists {
			return nil, fmt.Errorf("raw record not found for unique transaction %d", index+1)
		}

		result[index] = preparedTransaction{
			identity:   identity,
			normalized: normalized[index],
			rawRecord:  slices.Clone(rawRecord),
		}
	}

	return result, nil
}
