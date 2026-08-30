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
	Statement               Statement
	Transactions            []StoredTransaction
	RuleClassified          int
	CategorizationConflicts []RuleConflict
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

	storedRules, err := queryMerchantRules(ctx, tx, input.Statement.UserID, true)
	if err != nil {
		return StoredStatementImport{}, fmt.Errorf("load import merchant rules: %w", err)
	}
	rules := categorizationRules(storedRules)

	createdTransactions := make([]StoredTransaction, 0, len(input.Transactions))
	var (
		ruleClassified          int
		categorizationConflicts []RuleConflict
	)

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

		classified, conflict, err := classifyStoredTransaction(
			ctx,
			tx,
			input.Statement.Source,
			created,
			rules,
		)
		if err != nil {
			return StoredStatementImport{}, fmt.Errorf("categorize transaction %d: %w", index+1, err)
		}
		if classified {
			ruleClassified++
		}
		if conflict.TransactionID != 0 {
			categorizationConflicts = append(categorizationConflicts, conflict)
		}
	}

	if err := tx.Commit(); err != nil {
		return StoredStatementImport{}, fmt.Errorf("commit statement import: %w", err)
	}

	return StoredStatementImport{
		Statement:               createdStatement,
		Transactions:            createdTransactions,
		RuleClassified:          ruleClassified,
		CategorizationConflicts: categorizationConflicts,
	}, nil
}
