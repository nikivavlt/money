package main

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"money/internal/finance"
)

type monthlyCashFlow struct {
	Month            time.Time
	Currency         finance.Currency
	TransactionCount int64
	Income           finance.Money
	Expenses         finance.Money
	Savings          finance.Money
}

func (s *postgresStore) summarizeMonthByUser(
	ctx context.Context,
	userID int64,
	month time.Time,
) ([]monthlyCashFlow, error) {
	if userID < 1 {
		return nil, errors.New("monthly summary user ID must be positive")
	}

	if month.IsZero() {
		return nil, errors.New("monthly summary month is zero")
	}

	monthStart := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.UTC)
	nextMonth := monthStart.AddDate(0, 1, 0)

	const query = `
		SELECT
			t.currency,
			COUNT(*),
			SUM(
				CASE
					WHEN t.amount_minor > 0
					THEN t.amount_minor::numeric
					ELSE 0
				END
			)::text AS income_minor,
			SUM(
				CASE
					WHEN t.amount_minor < 0
					THEN -(t.amount_minor::numeric)
					ELSE 0
				END
			)::text AS expenses_minor,
			SUM(t.amount_minor::numeric)::text AS savings_minor
		FROM transactions AS t
		JOIN statements AS s
			ON s.id = t.statement_id
		WHERE s.user_id = $1
		  AND t.transaction_date >= $2
		  AND t.transaction_date < $3
		GROUP BY t.currency
		ORDER BY t.currency
	`

	rows, err := s.db.QueryContext(ctx, query, userID, monthStart, nextMonth)
	if err != nil {
		return nil, fmt.Errorf("summarize month: query: %w", err)
	}
	defer rows.Close()

	var summaries []monthlyCashFlow

	for rows.Next() {
		var (
			currencyText string
			incomeText   string
			expensesText string
			savingsText  string
			summary      monthlyCashFlow
		)

		err := rows.Scan(
			&currencyText,
			&summary.TransactionCount,
			&incomeText,
			&expensesText,
			&savingsText,
		)
		if err != nil {
			return nil, fmt.Errorf("summarize month: scan row: %w", err)
		}

		summary.Month = monthStart
		summary.Currency = finance.Currency(currencyText)

		if _, err := finance.MinorDigitsForCurrency(summary.Currency); err != nil {
			return nil, fmt.Errorf("summarize month: %w", err)
		}

		income, err := parseMonthlyAmount(incomeText, summary.Currency)
		if err != nil {
			return nil, fmt.Errorf("summarize month: parse income: %w", err)
		}

		expenses, err := parseMonthlyAmount(expensesText, summary.Currency)
		if err != nil {
			return nil, fmt.Errorf("summarize month: parse expenses: %w", err)
		}

		savings, err := parseMonthlyAmount(savingsText, summary.Currency)
		if err != nil {
			return nil, fmt.Errorf("summarize month: parse savings: %w", err)
		}

		summary.Income = income
		summary.Expenses = expenses
		summary.Savings = savings

		summaries = append(summaries, summary)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("summarize month: iterate rows: %w", err)
	}

	return summaries, nil
}

func parseMonthlyAmount(
	input string,
	currency finance.Currency,
) (finance.Money, error) {
	amount, err := strconv.ParseInt(input, 10, 64)
	if err != nil {
		return finance.Money{}, fmt.Errorf("parse amount %q: %w", input, err)
	}

	return finance.Money{
		Amount:   finance.AmountMinor(amount),
		Currency: currency,
	}, nil
}
