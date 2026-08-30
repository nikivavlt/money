package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

var defaultCategoryNames = []string{
	"Income",
	"Housing",
	"Groceries",
	"Restaurants",
	"Food Delivery",
	"Transport",
	"Fuel",
	"Shopping",
	"Entertainment",
	"Subscriptions",
	"Health",
	"Insurance",
	"Travel",
	"Education",
	"Utilities",
	"Cash",
	"Transfers",
	"Fees",
	"Gifts",
	"Other",
}

type Category struct {
	ID        int64
	UserID    int64
	Name      string
	IsDefault bool
	CreatedAt time.Time
}

func seedDefaultCategories(ctx context.Context, db queryExecutor, userID int64) error {
	if userID < 1 {
		return errors.New("category user ID must be positive")
	}

	const query = `
		INSERT INTO categories (user_id, name, is_default)
		SELECT $1, name, TRUE
		FROM unnest($2::text[]) AS defaults(name)
		ON CONFLICT (user_id, (lower(name))) DO NOTHING
	`

	if _, err := db.ExecContext(ctx, query, userID, defaultCategoryNames); err != nil {
		return fmt.Errorf("seed default categories: %w", err)
	}

	return nil
}

func (s *postgresStore) createCategory(ctx context.Context, userID int64, name string) (Category, error) {
	return insertCategory(ctx, s.db, userID, name, false)
}

func insertCategory(
	ctx context.Context,
	db rowQuerier,
	userID int64,
	name string,
	isDefault bool,
) (Category, error) {
	if userID < 1 {
		return Category{}, errors.New("category user ID must be positive")
	}

	normalized := strings.TrimSpace(name)
	if normalized == "" {
		return Category{}, errors.New("category name is empty")
	}

	const query = `
		INSERT INTO categories (user_id, name, is_default)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, (lower(name)))
		DO UPDATE SET name = categories.name
		RETURNING id, user_id, name, is_default, created_at
	`

	var category Category
	err := db.QueryRowContext(ctx, query, userID, normalized, isDefault).Scan(
		&category.ID,
		&category.UserID,
		&category.Name,
		&category.IsDefault,
		&category.CreatedAt,
	)
	if err != nil {
		return Category{}, fmt.Errorf("create category: %w", err)
	}

	return category, nil
}

func findCategoryByName(
	ctx context.Context,
	db rowQuerier,
	userID int64,
	name string,
) (Category, error) {
	if userID < 1 {
		return Category{}, errors.New("category user ID must be positive")
	}

	normalized := strings.TrimSpace(name)
	if normalized == "" {
		return Category{}, errors.New("category name is empty")
	}

	const query = `
		SELECT id, user_id, name, is_default, created_at
		FROM categories
		WHERE user_id = $1
		  AND lower(name) = lower($2)
	`

	var category Category
	err := db.QueryRowContext(ctx, query, userID, normalized).Scan(
		&category.ID,
		&category.UserID,
		&category.Name,
		&category.IsDefault,
		&category.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Category{}, fmt.Errorf("category %q: %w", normalized, sql.ErrNoRows)
		}

		return Category{}, fmt.Errorf("find category: %w", err)
	}

	return category, nil
}

func (s *postgresStore) listCategories(ctx context.Context, userID int64) ([]Category, error) {
	if userID < 1 {
		return nil, errors.New("category user ID must be positive")
	}

	const query = `
		SELECT id, user_id, name, is_default, created_at
		FROM categories
		WHERE user_id = $1
		ORDER BY is_default DESC, lower(name), id
	`

	rows, err := s.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("list categories: query: %w", err)
	}
	defer rows.Close()

	var categories []Category

	for rows.Next() {
		var category Category
		if err := rows.Scan(
			&category.ID,
			&category.UserID,
			&category.Name,
			&category.IsDefault,
			&category.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("list categories: scan row: %w", err)
		}

		categories = append(categories, category)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list categories: iterate rows: %w", err)
	}

	return categories, nil
}
