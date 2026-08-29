package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"money/internal/finance"
)

type NewTransaction struct {
	StatementID int64
	Fingerprint Fingerprint
	Transaction finance.Transaction
	RawRecord   []string
}

type StoredTransaction struct {
	ID          int64
	StatementID int64
	Fingerprint Fingerprint
	Transaction finance.Transaction
	RawRecord   []string
	CreatedAt   time.Time
}

func canonicalTransactionDate(value time.Time) time.Time {
	year, month, day := value.Date()

	return time.Date(
		year,
		month,
		day,
		0,
		0,
		0,
		0,
		time.UTC,
	)
}

func validateNewTransaction(input NewTransaction) error {
	if input.StatementID < 1 {
		return errors.New("transaction statement ID must be positive")
	}

	if input.Fingerprint == (Fingerprint{}) {
		return errors.New("transaction fingerprint is empty")
	}

	if input.Transaction.Date.IsZero() {
		return errors.New("transaction date is zero")
	}

	if _, err := finance.MinorDigitsForCurrency(input.Transaction.Amount.Currency); err != nil {
		return fmt.Errorf("transaction currency: %w", err)
	}

	if strings.TrimSpace(input.Transaction.Description) == "" {
		return errors.New("transaction description is empty")
	}

	if len(input.RawRecord) == 0 {
		return errors.New("transaction raw record is empty")
	}

	return nil
}

func (s *postgresStore) createTransaction(ctx context.Context, input NewTransaction) (StoredTransaction, error) {
	return insertTransaction(ctx, s.db, input)
}

func insertTransaction(
	ctx context.Context,
	db rowQuerier,
	input NewTransaction,
) (StoredTransaction, error) {
	if err := validateNewTransaction(input); err != nil {
		return StoredTransaction{}, err
	}

	input.Transaction.Date = canonicalTransactionDate(input.Transaction.Date)

	rawRecordJSON, err := json.Marshal(input.RawRecord)
	if err != nil {
		return StoredTransaction{}, fmt.Errorf("encode transaction raw record: %w", err)
	}

	description := strings.TrimSpace(input.Transaction.Description)
	counterparty := strings.TrimSpace(input.Transaction.Counterparty)

	var counterpartyArgument any
	if counterparty != "" {
		counterpartyArgument = counterparty
	}

	const query = `
		INSERT INTO transactions (
			statement_id,
			fingerprint,
			transaction_date,
			amount_minor,
			currency,
			description,
			counterparty,
			raw_record
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8::jsonb)
		RETURNING id, created_at`

	row := db.QueryRowContext(
		ctx,
		query,
		input.StatementID,
		input.Fingerprint[:],
		input.Transaction.Date,
		int64(input.Transaction.Amount.Amount),
		string(input.Transaction.Amount.Currency),
		description,
		counterpartyArgument,
		string(rawRecordJSON),
	)

	var created StoredTransaction

	if err := row.Scan(&created.ID, &created.CreatedAt); err != nil {
		return StoredTransaction{}, fmt.Errorf("create transaction: %w", err)
	}

	created.StatementID = input.StatementID
	created.Fingerprint = input.Fingerprint
	created.Transaction = input.Transaction
	created.Transaction.Description = description
	created.Transaction.Counterparty = counterparty
	created.RawRecord = slices.Clone(input.RawRecord)

	return created, nil
}
