package main

import (
	"fmt"
	"strings"
	"time"
)

type Transaction struct {
	Date        time.Time
	Amount      Money
	Description string
}

const transactionDateLayout = "2006-01-02"

func (t Transaction) IsInflow() bool {
	return t.Amount.Amount > 0
}

func (t Transaction) IsOutflow() bool {
	return t.Amount.Amount < 0
}

func parseTransactionDate(input string, location *time.Location) (time.Time, error) {
	if location == nil {
		return time.Time{}, fmt.Errorf("parse transaction date: nil location")
	}

	normalized := strings.TrimSpace(input)
	if normalized == "" {
		return time.Time{}, fmt.Errorf(
			"parse transaction date %q: empty input",
			input,
		)
	}

	parsed, err := time.ParseInLocation(transactionDateLayout, normalized, location)
	if err != nil {
		return time.Time{}, fmt.Errorf(
			"parse transaction date %q in location %q: %w",
			input,
			location.String(),
			err,
		)
	}

	return parsed, nil
}
