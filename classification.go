package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"money/internal/categorization"
	"money/internal/finance"
	"money/internal/statement"
)

type ReviewTransaction struct {
	ID                    int64
	Source                statement.Source
	Date                  time.Time
	Amount                finance.Money
	Description           string
	Counterparty          string
	NormalizedDescription string
}

type ManualCorrection struct {
	TransactionID int64
	CategoryID    int64
	MerchantName  string
}

type RuleConflict struct {
	TransactionID int64
	RuleIDs       []int64
}

type RuleApplicationSummary struct {
	Considered int
	Classified int
	Conflicts  []RuleConflict
}

func matchStoredTransaction(
	source statement.Source,
	transaction finance.Transaction,
	rules []categorization.Rule,
) (categorization.MatchResult, error) {
	return categorization.MatchAny(
		string(source),
		[]string{transaction.Counterparty, transaction.Description},
		rules,
	)
}

func persistRuleClassification(
	ctx context.Context,
	db queryExecutor,
	transactionID int64,
	rule categorization.Rule,
) (bool, error) {
	const updateQuery = `
		UPDATE transactions
		SET merchant_id = $2,
			category_id = $3,
			categorization_source = 'rule',
			applied_rule_id = $4,
			review_status = 'resolved'
		WHERE id = $1
		  AND categorization_source IS DISTINCT FROM 'manual'
		RETURNING id
	`

	var updatedID int64
	err := db.QueryRowContext(
		ctx,
		updateQuery,
		transactionID,
		rule.MerchantID,
		rule.CategoryID,
		rule.ID,
	).Scan(&updatedID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("apply merchant rule %d to transaction %d: %w", rule.ID, transactionID, err)
	}

	const historyQuery = `
		INSERT INTO transaction_classifications (
			transaction_id,
			merchant_id,
			category_id,
			source,
			rule_id
		)
		VALUES ($1, $2, $3, 'rule', $4)
	`

	if _, err := db.ExecContext(
		ctx,
		historyQuery,
		transactionID,
		rule.MerchantID,
		rule.CategoryID,
		rule.ID,
	); err != nil {
		return false, fmt.Errorf("record merchant rule classification: %w", err)
	}

	return true, nil
}

func classifyStoredTransaction(
	ctx context.Context,
	db queryExecutor,
	source statement.Source,
	transaction StoredTransaction,
	rules []categorization.Rule,
) (classified bool, conflict RuleConflict, err error) {
	result, err := matchStoredTransaction(source, transaction.Transaction, rules)
	if errors.Is(err, categorization.ErrRuleConflict) {
		conflict.TransactionID = transaction.ID
		conflict.RuleIDs = make([]int64, len(result.Conflicts))
		for index, rule := range result.Conflicts {
			conflict.RuleIDs[index] = rule.ID
		}

		return false, conflict, nil
	}
	if err != nil {
		return false, RuleConflict{}, err
	}

	if result.Rule.ID == 0 {
		return false, RuleConflict{}, nil
	}

	classified, err = persistRuleClassification(ctx, db, transaction.ID, result.Rule)
	if err != nil {
		return false, RuleConflict{}, err
	}

	return classified, RuleConflict{}, nil
}

func (s *postgresStore) applyMerchantRules(
	ctx context.Context,
	userID int64,
) (RuleApplicationSummary, error) {
	if userID < 1 {
		return RuleApplicationSummary{}, errors.New("apply merchant rules user ID must be positive")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return RuleApplicationSummary{}, fmt.Errorf("begin apply merchant rules: %w", err)
	}
	defer tx.Rollback()

	storedRules, err := queryMerchantRules(ctx, tx, userID, true)
	if err != nil {
		return RuleApplicationSummary{}, err
	}
	rules := categorizationRules(storedRules)

	const query = `
		SELECT
			t.id,
			t.statement_id,
			s.source,
			t.transaction_date,
			t.amount_minor,
			t.currency,
			t.description,
			t.counterparty,
			t.normalized_description,
			t.created_at
		FROM transactions AS t
		JOIN statements AS s
			ON s.id = t.statement_id
		WHERE s.user_id = $1
		  AND t.category_id IS NULL
		  AND t.review_status = 'pending'
		ORDER BY t.transaction_date, t.id
	`

	rows, err := tx.QueryContext(ctx, query, userID)
	if err != nil {
		return RuleApplicationSummary{}, fmt.Errorf("load transactions for merchant rules: %w", err)
	}

	type candidate struct {
		transaction StoredTransaction
		source      statement.Source
	}

	var candidates []candidate

	for rows.Next() {
		var (
			item         candidate
			amountMinor  int64
			currency     string
			counterparty sql.NullString
		)

		if err := rows.Scan(
			&item.transaction.ID,
			&item.transaction.StatementID,
			&item.source,
			&item.transaction.Transaction.Date,
			&amountMinor,
			&currency,
			&item.transaction.Transaction.Description,
			&counterparty,
			&item.transaction.NormalizedDescription,
			&item.transaction.CreatedAt,
		); err != nil {
			rows.Close()
			return RuleApplicationSummary{}, fmt.Errorf("load transactions for merchant rules: scan row: %w", err)
		}

		item.transaction.Transaction.Amount = finance.Money{
			Amount:   finance.AmountMinor(amountMinor),
			Currency: finance.Currency(currency),
		}
		if counterparty.Valid {
			item.transaction.Transaction.Counterparty = counterparty.String
		}

		candidates = append(candidates, item)
	}

	if err := rows.Err(); err != nil {
		rows.Close()
		return RuleApplicationSummary{}, fmt.Errorf("load transactions for merchant rules: iterate rows: %w", err)
	}
	if err := rows.Close(); err != nil {
		return RuleApplicationSummary{}, fmt.Errorf("load transactions for merchant rules: close rows: %w", err)
	}

	summary := RuleApplicationSummary{Considered: len(candidates)}

	for _, candidate := range candidates {
		classified, conflict, err := classifyStoredTransaction(
			ctx,
			tx,
			candidate.source,
			candidate.transaction,
			rules,
		)
		if err != nil {
			return RuleApplicationSummary{}, err
		}

		if classified {
			summary.Classified++
		}
		if conflict.TransactionID != 0 {
			summary.Conflicts = append(summary.Conflicts, conflict)
		}
	}

	if err := tx.Commit(); err != nil {
		return RuleApplicationSummary{}, fmt.Errorf("commit merchant rule application: %w", err)
	}

	return summary, nil
}

func (s *postgresStore) listPendingReviews(
	ctx context.Context,
	userID int64,
) ([]ReviewTransaction, error) {
	if userID < 1 {
		return nil, errors.New("review user ID must be positive")
	}

	const query = `
		SELECT
			t.id,
			s.source,
			t.transaction_date,
			t.amount_minor,
			t.currency,
			t.description,
			t.counterparty,
			t.normalized_description
		FROM transactions AS t
		JOIN statements AS s
			ON s.id = t.statement_id
		WHERE s.user_id = $1
		  AND t.amount_minor < 0
		  AND t.category_id IS NULL
		  AND t.review_status = 'pending'
		ORDER BY t.transaction_date, t.id
	`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("list pending reviews: query: %w", err)
	}
	defer rows.Close()

	var transactions []ReviewTransaction

	for rows.Next() {
		var (
			transaction  ReviewTransaction
			amountMinor  int64
			currency     string
			counterparty sql.NullString
		)

		if err := rows.Scan(
			&transaction.ID,
			&transaction.Source,
			&transaction.Date,
			&amountMinor,
			&currency,
			&transaction.Description,
			&counterparty,
			&transaction.NormalizedDescription,
		); err != nil {
			return nil, fmt.Errorf("list pending reviews: scan row: %w", err)
		}

		transaction.Amount = finance.Money{
			Amount:   finance.AmountMinor(amountMinor),
			Currency: finance.Currency(currency),
		}
		if counterparty.Valid {
			transaction.Counterparty = counterparty.String
		}

		transactions = append(transactions, transaction)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list pending reviews: iterate rows: %w", err)
	}

	return transactions, nil
}

func (s *postgresStore) findReviewTransaction(
	ctx context.Context,
	userID int64,
	transactionID int64,
) (ReviewTransaction, error) {
	if userID < 1 {
		return ReviewTransaction{}, errors.New("review user ID must be positive")
	}
	if transactionID < 1 {
		return ReviewTransaction{}, errors.New("review transaction ID must be positive")
	}

	const query = `
		SELECT
			t.id,
			s.source,
			t.transaction_date,
			t.amount_minor,
			t.currency,
			t.description,
			t.counterparty,
			t.normalized_description
		FROM transactions AS t
		JOIN statements AS s
			ON s.id = t.statement_id
		WHERE s.user_id = $1
		  AND t.id = $2
	`

	var (
		transaction  ReviewTransaction
		amountMinor  int64
		currency     string
		counterparty sql.NullString
	)

	err := s.db.QueryRowContext(ctx, query, userID, transactionID).Scan(
		&transaction.ID,
		&transaction.Source,
		&transaction.Date,
		&amountMinor,
		&currency,
		&transaction.Description,
		&counterparty,
		&transaction.NormalizedDescription,
	)
	if err != nil {
		return ReviewTransaction{}, fmt.Errorf("find review transaction: %w", err)
	}

	transaction.Amount = finance.Money{
		Amount:   finance.AmountMinor(amountMinor),
		Currency: finance.Currency(currency),
	}
	if counterparty.Valid {
		transaction.Counterparty = counterparty.String
	}

	return transaction, nil
}

func (s *postgresStore) applyManualCorrection(
	ctx context.Context,
	userID int64,
	correction ManualCorrection,
) (Merchant, Category, error) {
	if userID < 1 {
		return Merchant{}, Category{}, errors.New("manual correction user ID must be positive")
	}
	if correction.TransactionID < 1 {
		return Merchant{}, Category{}, errors.New("manual correction transaction ID must be positive")
	}
	if correction.CategoryID < 1 {
		return Merchant{}, Category{}, errors.New("manual correction category ID must be positive")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Merchant{}, Category{}, fmt.Errorf("begin manual correction: %w", err)
	}
	defer tx.Rollback()

	const transactionQuery = `
		SELECT t.id
		FROM transactions AS t
		JOIN statements AS s
			ON s.id = t.statement_id
		WHERE t.id = $1
		  AND s.user_id = $2
		FOR UPDATE OF t
	`

	var transactionID int64
	if err := tx.QueryRowContext(ctx, transactionQuery, correction.TransactionID, userID).Scan(&transactionID); err != nil {
		return Merchant{}, Category{}, fmt.Errorf("find transaction for manual correction: %w", err)
	}

	const categoryQuery = `
		SELECT id, user_id, name, is_default, created_at
		FROM categories
		WHERE id = $1
		  AND user_id = $2
	`

	var category Category
	if err := tx.QueryRowContext(ctx, categoryQuery, correction.CategoryID, userID).Scan(
		&category.ID,
		&category.UserID,
		&category.Name,
		&category.IsDefault,
		&category.CreatedAt,
	); err != nil {
		return Merchant{}, Category{}, fmt.Errorf("find category for manual correction: %w", err)
	}

	merchant, err := upsertMerchant(ctx, tx, userID, correction.MerchantName)
	if err != nil {
		return Merchant{}, Category{}, fmt.Errorf("manual correction: %w", err)
	}

	const updateQuery = `
		UPDATE transactions
		SET merchant_id = $2,
			category_id = $3,
			categorization_source = 'manual',
			applied_rule_id = NULL,
			review_status = 'resolved'
		WHERE id = $1
	`

	if _, err := tx.ExecContext(ctx, updateQuery, transactionID, merchant.ID, category.ID); err != nil {
		return Merchant{}, Category{}, fmt.Errorf("update manual correction: %w", err)
	}

	const historyQuery = `
		INSERT INTO transaction_classifications (
			transaction_id,
			merchant_id,
			category_id,
			source,
			rule_id
		)
		VALUES ($1, $2, $3, 'manual', NULL)
	`

	if _, err := tx.ExecContext(ctx, historyQuery, transactionID, merchant.ID, category.ID); err != nil {
		return Merchant{}, Category{}, fmt.Errorf("record manual correction: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return Merchant{}, Category{}, fmt.Errorf("commit manual correction: %w", err)
	}

	return merchant, category, nil
}

func (s *postgresStore) skipReview(ctx context.Context, userID int64, transactionID int64) error {
	if userID < 1 {
		return errors.New("review user ID must be positive")
	}
	if transactionID < 1 {
		return errors.New("review transaction ID must be positive")
	}

	const query = `
		UPDATE transactions AS t
		SET review_status = 'skipped'
		FROM statements AS s
		WHERE t.id = $1
		  AND t.statement_id = s.id
		  AND s.user_id = $2
		  AND t.category_id IS NULL
	`

	result, err := s.db.ExecContext(ctx, query, transactionID, userID)
	if err != nil {
		return fmt.Errorf("skip transaction review: %w", err)
	}

	updated, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("skip transaction review: rows affected: %w", err)
	}
	if updated != 1 {
		return fmt.Errorf("skip transaction review: transaction %d was not pending for user", transactionID)
	}

	return nil
}
