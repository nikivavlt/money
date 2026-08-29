package main

import (
	"context"
	"fmt"
	"money/internal/finance"
)

type NewStatementTransaction struct {
	Fingerprint Fingerprint
	Transaction finance.Transaction
	RawRecord   []string
}

type NewStatementImport struct {
	Statement    NewStatement
	Transactions []NewStatementTransaction
}

type StoredStatementImport struct {
	Statement    Statement
	Transactions []StoredTransaction
}

func (s *postgresStore) createStatementImport(ctx context.Context, input NewStatementImport) (StoredStatementImport, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return StoredStatementImport{}, fmt.Errorf("begin statement import: %w", err)
	}
	defer tx.Rollback()

	createdStatement, err := insertStatement(ctx, tx, input.Statement)
	if err != nil {
		return StoredStatementImport{}, fmt.Errorf("import statement: %w", err)
	}

	createdTransactions := make([]StoredTransaction, 0, len(input.Transactions))

	for index, transaction := range input.Transactions {
		created, err := insertTransaction(
			ctx,
			tx,
			NewTransaction{
				StatementID: createdStatement.ID,
				Fingerprint: transaction.Fingerprint,
				Transaction: transaction.Transaction,
				RawRecord:   transaction.RawRecord,
			},
		)
		if err != nil {
			return StoredStatementImport{}, fmt.Errorf("import transaction %d: %w", index+1, err)
		}

		createdTransactions = append(createdTransactions, created)
	}

	if err := tx.Commit(); err != nil {
		return StoredStatementImport{}, fmt.Errorf("commit statement import: %w", err)
	}

	return StoredStatementImport{
		Statement:    createdStatement,
		Transactions: createdTransactions,
	}, nil
}
