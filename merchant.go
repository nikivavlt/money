package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"money/internal/categorization"
)

type Merchant struct {
	ID             int64
	UserID         int64
	Name           string
	NormalizedName string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func upsertMerchant(
	ctx context.Context,
	db rowQuerier,
	userID int64,
	name string,
) (Merchant, error) {
	if userID < 1 {
		return Merchant{}, errors.New("merchant user ID must be positive")
	}

	displayName := strings.TrimSpace(name)
	if displayName == "" {
		return Merchant{}, errors.New("merchant name is empty")
	}

	normalizedName := categorization.NormalizeText(displayName)
	if normalizedName == "" {
		return Merchant{}, errors.New("merchant normalized name is empty")
	}

	const query = `
		INSERT INTO merchants (user_id, name, normalized_name)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, normalized_name)
		DO UPDATE SET updated_at = merchants.updated_at
		RETURNING id, user_id, name, normalized_name, created_at, updated_at
	`

	var merchant Merchant
	err := db.QueryRowContext(ctx, query, userID, displayName, normalizedName).Scan(
		&merchant.ID,
		&merchant.UserID,
		&merchant.Name,
		&merchant.NormalizedName,
		&merchant.CreatedAt,
		&merchant.UpdatedAt,
	)
	if err != nil {
		return Merchant{}, fmt.Errorf("create merchant: %w", err)
	}

	return merchant, nil
}

func (s *postgresStore) listMerchants(ctx context.Context, userID int64) ([]Merchant, error) {
	if userID < 1 {
		return nil, errors.New("merchant user ID must be positive")
	}

	const query = `
		SELECT id, user_id, name, normalized_name, created_at, updated_at
		FROM merchants
		WHERE user_id = $1
		ORDER BY lower(name), id
	`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("list merchants: query: %w", err)
	}
	defer rows.Close()

	var merchants []Merchant

	for rows.Next() {
		var merchant Merchant
		if err := rows.Scan(
			&merchant.ID,
			&merchant.UserID,
			&merchant.Name,
			&merchant.NormalizedName,
			&merchant.CreatedAt,
			&merchant.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("list merchants: scan row: %w", err)
		}

		merchants = append(merchants, merchant)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list merchants: iterate rows: %w", err)
	}

	return merchants, nil
}
