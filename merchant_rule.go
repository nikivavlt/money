package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"money/internal/categorization"
	"money/internal/statement"
)

var ErrMerchantRuleAlreadyExists = errors.New("merchant rule already exists")

type NewMerchantRule struct {
	UserID       int64
	Source       statement.Source
	MatchType    categorization.MatchType
	Pattern      string
	MerchantName string
	CategoryName string
	Priority     int
}

type MerchantRule struct {
	ID                int64
	UserID            int64
	Source            statement.Source
	MatchType         categorization.MatchType
	Pattern           string
	NormalizedPattern string
	Merchant          Merchant
	Category          Category
	Priority          int
	Enabled           bool
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func validateNewMerchantRule(input NewMerchantRule) (string, error) {
	if input.UserID < 1 {
		return "", errors.New("merchant rule user ID must be positive")
	}

	if input.Source != "" && input.Source != statement.Revolut && input.Source != statement.Swedbank {
		return "", fmt.Errorf("unsupported merchant rule source %q", input.Source)
	}

	if strings.TrimSpace(input.MerchantName) == "" {
		return "", errors.New("merchant rule merchant name is empty")
	}

	if strings.TrimSpace(input.CategoryName) == "" {
		return "", errors.New("merchant rule category name is empty")
	}

	normalizedPattern, err := categorization.NormalizePattern(input.MatchType, input.Pattern)
	if err != nil {
		return "", err
	}

	return normalizedPattern, nil
}

func (s *postgresStore) createMerchantRule(
	ctx context.Context,
	input NewMerchantRule,
) (MerchantRule, error) {
	normalizedPattern, err := validateNewMerchantRule(input)
	if err != nil {
		return MerchantRule{}, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return MerchantRule{}, fmt.Errorf("begin create merchant rule: %w", err)
	}
	defer tx.Rollback()

	category, err := findCategoryByName(ctx, tx, input.UserID, input.CategoryName)
	if err != nil {
		return MerchantRule{}, fmt.Errorf("create merchant rule: %w", err)
	}

	merchant, err := upsertMerchant(ctx, tx, input.UserID, input.MerchantName)
	if err != nil {
		return MerchantRule{}, fmt.Errorf("create merchant rule: %w", err)
	}

	var sourceArgument any
	if input.Source != "" {
		sourceArgument = string(input.Source)
	}

	const query = `
		INSERT INTO merchant_rules (
			user_id,
			source,
			match_type,
			pattern,
			normalized_pattern,
			merchant_id,
			category_id,
			priority
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, enabled, created_at, updated_at
	`

	var rule MerchantRule
	err = tx.QueryRowContext(
		ctx,
		query,
		input.UserID,
		sourceArgument,
		string(input.MatchType),
		strings.TrimSpace(input.Pattern),
		normalizedPattern,
		merchant.ID,
		category.ID,
		input.Priority,
	).Scan(&rule.ID, &rule.Enabled, &rule.CreatedAt, &rule.UpdatedAt)
	if err != nil {
		if isUniqueViolation(err) {
			return MerchantRule{}, fmt.Errorf(
				"%w: %s rule %q",
				ErrMerchantRuleAlreadyExists,
				input.MatchType,
				normalizedPattern,
			)
		}

		return MerchantRule{}, fmt.Errorf("create merchant rule: insert: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return MerchantRule{}, fmt.Errorf("commit create merchant rule: %w", err)
	}

	rule.UserID = input.UserID
	rule.Source = input.Source
	rule.MatchType = input.MatchType
	rule.Pattern = strings.TrimSpace(input.Pattern)
	rule.NormalizedPattern = normalizedPattern
	rule.Merchant = merchant
	rule.Category = category
	rule.Priority = input.Priority

	return rule, nil
}

func (s *postgresStore) learnMerchantRule(
	ctx context.Context,
	input NewMerchantRule,
) (MerchantRule, error) {
	normalizedPattern, err := validateNewMerchantRule(input)
	if err != nil {
		return MerchantRule{}, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return MerchantRule{}, fmt.Errorf("begin learn merchant rule: %w", err)
	}
	defer tx.Rollback()

	category, err := findCategoryByName(ctx, tx, input.UserID, input.CategoryName)
	if err != nil {
		return MerchantRule{}, fmt.Errorf("learn merchant rule: %w", err)
	}

	merchant, err := upsertMerchant(ctx, tx, input.UserID, input.MerchantName)
	if err != nil {
		return MerchantRule{}, fmt.Errorf("learn merchant rule: %w", err)
	}

	var sourceArgument any
	if input.Source != "" {
		sourceArgument = string(input.Source)
	}

	const query = `
		INSERT INTO merchant_rules (
			user_id,
			source,
			match_type,
			pattern,
			normalized_pattern,
			merchant_id,
			category_id,
			priority
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (
			user_id,
			(COALESCE(source, '')),
			match_type,
			normalized_pattern
		)
		DO UPDATE SET
			merchant_id = EXCLUDED.merchant_id,
			category_id = EXCLUDED.category_id,
			priority = EXCLUDED.priority,
			enabled = TRUE,
			updated_at = CURRENT_TIMESTAMP
		RETURNING id, enabled, created_at, updated_at
	`

	var rule MerchantRule
	err = tx.QueryRowContext(
		ctx,
		query,
		input.UserID,
		sourceArgument,
		string(input.MatchType),
		strings.TrimSpace(input.Pattern),
		normalizedPattern,
		merchant.ID,
		category.ID,
		input.Priority,
	).Scan(&rule.ID, &rule.Enabled, &rule.CreatedAt, &rule.UpdatedAt)
	if err != nil {
		return MerchantRule{}, fmt.Errorf("learn merchant rule: upsert: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return MerchantRule{}, fmt.Errorf("commit learn merchant rule: %w", err)
	}

	rule.UserID = input.UserID
	rule.Source = input.Source
	rule.MatchType = input.MatchType
	rule.Pattern = strings.TrimSpace(input.Pattern)
	rule.NormalizedPattern = normalizedPattern
	rule.Merchant = merchant
	rule.Category = category
	rule.Priority = input.Priority

	return rule, nil
}

func (s *postgresStore) listMerchantRules(
	ctx context.Context,
	userID int64,
) ([]MerchantRule, error) {
	return queryMerchantRules(ctx, s.db, userID, false)
}

func queryMerchantRules(
	ctx context.Context,
	db queryExecutor,
	userID int64,
	enabledOnly bool,
) ([]MerchantRule, error) {
	if userID < 1 {
		return nil, errors.New("merchant rule user ID must be positive")
	}

	const query = `
		SELECT
			r.id,
			r.user_id,
			COALESCE(r.source, ''),
			r.match_type,
			r.pattern,
			r.normalized_pattern,
			r.priority,
			r.enabled,
			r.created_at,
			r.updated_at,
			m.id,
			m.user_id,
			m.name,
			m.normalized_name,
			m.created_at,
			m.updated_at,
			c.id,
			c.user_id,
			c.name,
			c.is_default,
			c.created_at
		FROM merchant_rules AS r
		JOIN merchants AS m
			ON m.id = r.merchant_id
		JOIN categories AS c
			ON c.id = r.category_id
		WHERE r.user_id = $1
		  AND (NOT $2 OR r.enabled)
		ORDER BY
			r.priority DESC,
			CASE r.match_type
				WHEN 'exact' THEN 4
				WHEN 'prefix' THEN 3
				WHEN 'contains' THEN 2
				WHEN 'regex' THEN 1
			END DESC,
			length(r.normalized_pattern) DESC,
			r.id
	`

	rows, err := db.QueryContext(ctx, query, userID, enabledOnly)
	if err != nil {
		return nil, fmt.Errorf("list merchant rules: query: %w", err)
	}
	defer rows.Close()

	var rules []MerchantRule

	for rows.Next() {
		var (
			rule      MerchantRule
			source    string
			matchType string
		)

		if err := rows.Scan(
			&rule.ID,
			&rule.UserID,
			&source,
			&matchType,
			&rule.Pattern,
			&rule.NormalizedPattern,
			&rule.Priority,
			&rule.Enabled,
			&rule.CreatedAt,
			&rule.UpdatedAt,
			&rule.Merchant.ID,
			&rule.Merchant.UserID,
			&rule.Merchant.Name,
			&rule.Merchant.NormalizedName,
			&rule.Merchant.CreatedAt,
			&rule.Merchant.UpdatedAt,
			&rule.Category.ID,
			&rule.Category.UserID,
			&rule.Category.Name,
			&rule.Category.IsDefault,
			&rule.Category.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("list merchant rules: scan row: %w", err)
		}

		rule.Source = statement.Source(source)
		rule.MatchType = categorization.MatchType(matchType)

		if _, err := categorization.NormalizePattern(rule.MatchType, rule.Pattern); err != nil {
			return nil, fmt.Errorf("merchant rule %d is invalid: %w", rule.ID, err)
		}

		rules = append(rules, rule)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list merchant rules: iterate rows: %w", err)
	}

	return rules, nil
}

func categorizationRules(rules []MerchantRule) []categorization.Rule {
	result := make([]categorization.Rule, len(rules))

	for index, rule := range rules {
		result[index] = categorization.Rule{
			ID:                rule.ID,
			Source:            string(rule.Source),
			MatchType:         rule.MatchType,
			Pattern:           rule.Pattern,
			NormalizedPattern: rule.NormalizedPattern,
			MerchantID:        rule.Merchant.ID,
			MerchantName:      rule.Merchant.Name,
			CategoryID:        rule.Category.ID,
			CategoryName:      rule.Category.Name,
			Priority:          rule.Priority,
			Enabled:           rule.Enabled,
		}
	}

	return result
}

func (s *postgresStore) setMerchantRuleEnabled(
	ctx context.Context,
	userID int64,
	ruleID int64,
	enabled bool,
) error {
	if userID < 1 {
		return errors.New("merchant rule user ID must be positive")
	}
	if ruleID < 1 {
		return errors.New("merchant rule ID must be positive")
	}

	const query = `
		UPDATE merchant_rules
		SET enabled = $3,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
		  AND user_id = $2
	`

	result, err := s.db.ExecContext(ctx, query, ruleID, userID, enabled)
	if err != nil {
		return fmt.Errorf("set merchant rule enabled: %w", err)
	}

	updated, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("set merchant rule enabled: rows affected: %w", err)
	}
	if updated != 1 {
		return fmt.Errorf("merchant rule %d was not found for user", ruleID)
	}

	return nil
}

func (s *postgresStore) setMerchantRulePriority(
	ctx context.Context,
	userID int64,
	ruleID int64,
	priority int,
) error {
	if userID < 1 {
		return errors.New("merchant rule user ID must be positive")
	}
	if ruleID < 1 {
		return errors.New("merchant rule ID must be positive")
	}

	const query = `
		UPDATE merchant_rules
		SET priority = $3,
			updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
		  AND user_id = $2
	`

	result, err := s.db.ExecContext(ctx, query, ruleID, userID, priority)
	if err != nil {
		return fmt.Errorf("set merchant rule priority: %w", err)
	}

	updated, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("set merchant rule priority: rows affected: %w", err)
	}
	if updated != 1 {
		return fmt.Errorf("merchant rule %d was not found for user", ruleID)
	}

	return nil
}
