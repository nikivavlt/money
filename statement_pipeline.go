package main

import (
	"errors"
	"fmt"
	"io"
	"money/internal/finance"
	"time"
)

var ErrDuplicateConflict = errors.New("duplicate transaction conflict")

type duplicateConflictError struct {
	count int
}

func (e *duplicateConflictError) Error() string {
	return fmt.Sprintf(
		"%d duplicate transaction conflicts detected",
		e.count,
	)
}

func (e *duplicateConflictError) Unwrap() error {
	return ErrDuplicateConflict
}

type preparedStatementImport struct {
	source       statementSource
	transactions []finance.Transaction
	duplicates   []duplicateCandidate
	conflicts    []duplicateConflict
	summary      importSummary
}

func prepareStatementImport(
	input io.Reader,
	location *time.Location,
) (preparedStatementImport, error) {
	if location == nil {
		return preparedStatementImport{},
			fmt.Errorf("prepare statement import: nil location")
	}

	source, imported, err := importStatement(input)
	if err != nil {
		return preparedStatementImport{},
			fmt.Errorf("prepare statement import: %w", err)
	}

	deduplication := deduplicateImportedTransactions(imported)
	summary := summarizeDeduplication(imported, deduplication)

	prepared := preparedStatementImport{
		source:     source,
		duplicates: deduplication.duplicates,
		conflicts:  deduplication.conflicts,
		summary:    summary,
	}

	if !summary.isConsistent() {
		return prepared, fmt.Errorf(
			"prepare statement import: inconsistent import summary",
		)
	}

	if summary.hasConflicts() {
		return prepared, fmt.Errorf(
			"prepare statement import: %w",
			&duplicateConflictError{
				count: summary.conflictRows,
			},
		)
	}

	transactions, err := normalizeImportedTransactions(
		deduplication.unique,
		location,
	)
	if err != nil {
		return prepared, fmt.Errorf(
			"prepare statement import: %w",
			err,
		)
	}

	prepared.transactions = transactions

	return prepared, nil
}
