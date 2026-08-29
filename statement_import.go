package main

import (
	"context"
	"fmt"
	"io"
	"slices"
	"time"

	"money/internal/statement"
)

type statementImportResult struct {
	Stored  StoredStatementImport
	Summary statement.Summary
}

func newStatementImport(
	userID int64,
	originalFilename string,
	prepared statement.Prepared,
) NewStatementImport {
	var transactions []NewStatementTransaction

	if prepared.Transactions != nil {
		transactions = make([]NewStatementTransaction, len(prepared.Transactions))

		for index, preparedTransaction := range prepared.Transactions {
			transactions[index] = NewStatementTransaction{
				Fingerprint: Fingerprint(preparedTransaction.Fingerprint),
				Transaction: preparedTransaction.Transaction,
				RawRecord:   slices.Clone(preparedTransaction.RawRecord),
			}
		}
	}

	return NewStatementImport{
		Statement: NewStatement{
			UserID:           userID,
			Source:           prepared.Source,
			OriginalFilename: originalFilename,
			Fingerprint:      Fingerprint(prepared.Fingerprint),
			RawHeader:        slices.Clone(prepared.RawHeader),
		},
		Transactions: transactions,
	}
}

func importStatement(
	ctx context.Context,
	store *postgresStore,
	userID int64,
	originalFilename string,
	input io.Reader,
	location *time.Location,
) (statementImportResult, error) {
	prepared, err := statement.Prepare(input, location)

	result := statementImportResult{
		Summary: prepared.Summary,
	}

	if err != nil {
		return result, fmt.Errorf("prepare statement: %w", err)
	}

	newImport := newStatementImport(
		userID,
		originalFilename,
		prepared,
	)

	stored, err := store.createStatementImport(ctx, newImport)
	if err != nil {
		return result, fmt.Errorf("store statement import: %w", err)
	}

	result.Stored = stored

	return result, nil
}
