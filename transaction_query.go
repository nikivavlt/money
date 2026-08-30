package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"money/internal/finance"
	"money/internal/statement"
)

type listedTransaction struct {
	ID               int64
	StatementID      int64
	Source           statement.Source
	OriginalFilename string
	Transaction      finance.Transaction
	MerchantName     string
	CategoryName     string
	Classification   string
	CreatedAt        time.Time
}

func (s *postgresStore) listTransactionsByUser(
	ctx context.Context,
	userID int64,
) ([]listedTransaction, error) {
	if userID < 1 {
		return nil, errors.New("transaction user ID must be positive")
	}

	const query = `
		SELECT
			t.id,
			t.statement_id,
			s.source,
			s.original_filename,
			t.transaction_date,
			t.amount_minor,
			t.currency,
			t.description,
			t.counterparty,
			m.name,
			c.name,
			t.categorization_source,
			t.created_at
		FROM transactions AS t
		JOIN statements AS s
			ON s.id = t.statement_id
		LEFT JOIN merchants AS m
			ON m.id = t.merchant_id
		LEFT JOIN categories AS c
			ON c.id = t.category_id
		WHERE s.user_id = $1
		ORDER BY
			t.transaction_date DESC,
			t.id DESC
	`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("list transactions: query: %w", err)
	}
	defer rows.Close()

	var transactions []listedTransaction

	for rows.Next() {
		var (
			transaction    listedTransaction
			source         string
			amountMinor    int64
			currency       string
			counterparty   sql.NullString
			merchant       sql.NullString
			category       sql.NullString
			classification sql.NullString
		)

		err := rows.Scan(
			&transaction.ID,
			&transaction.StatementID,
			&source,
			&transaction.OriginalFilename,
			&transaction.Transaction.Date,
			&amountMinor,
			&currency,
			&transaction.Transaction.Description,
			&counterparty,
			&merchant,
			&category,
			&classification,
			&transaction.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("list transactions: scan row: %w", err)
		}

		transaction.Source = statement.Source(source)

		switch transaction.Source {
		case statement.Revolut, statement.Swedbank:
		default:
			return nil, fmt.Errorf("list transactions: transaction %d has unsupported source %q", transaction.ID, source)
		}

		transaction.Transaction.Amount = finance.Money{
			Amount:   finance.AmountMinor(amountMinor),
			Currency: finance.Currency(currency),
		}

		if _, err := finance.MinorDigitsForCurrency(transaction.Transaction.Amount.Currency); err != nil {
			return nil, fmt.Errorf("list transactions: transaction %d: %w", transaction.ID, err)
		}

		if counterparty.Valid {
			transaction.Transaction.Counterparty = counterparty.String
		}
		if merchant.Valid {
			transaction.MerchantName = merchant.String
		}
		if category.Valid {
			transaction.CategoryName = category.String
		}
		if classification.Valid {
			transaction.Classification = classification.String
		}

		transactions = append(transactions, transaction)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list transactions: iterate rows: %w", err)
	}

	return transactions, nil
}
