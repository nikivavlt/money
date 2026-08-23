package main

import (
	"fmt"
	"strings"
	"time"

	"money/finance"
)

type Transaction = finance.Transaction

const transactionDateLayout = "2006-01-02"

func parseTransactionDate(
	input string,
	location *time.Location,
) (time.Time, error) {
	if location == nil {
		return time.Time{}, fmt.Errorf(
			"parse transaction date: nil location",
		)
	}

	normalized := strings.TrimSpace(input)
	if normalized == "" {
		return time.Time{}, fmt.Errorf(
			"parse transaction date %q: empty input",
			input,
		)
	}

	parsed, err := time.ParseInLocation(
		transactionDateLayout,
		normalized,
		location,
	)
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
