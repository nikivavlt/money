package main

import (
	"fmt"
	"strconv"
	"strings"
)

type AmountMinor int64
type Currency string

const (
	EUR Currency = "EUR"
	USD Currency = "USD"
)

type Money struct {
	Amount   AmountMinor
	Currency Currency
}

func parseMoney(input string, currency Currency) (Money, error) {
	minorDigits, err := minorDigitsForCurrency(currency)
	if err != nil {
		return Money{}, fmt.Errorf("parse money: %w", err)
	}

	amountMinor, err := parseAmountMinor(input, minorDigits)
	if err != nil {
		return Money{}, fmt.Errorf("parse money: %w", err)
	}

	return Money{
		Amount:   amountMinor,
		Currency: currency,
	}, nil
}

func minorDigitsForCurrency(currency Currency) (int, error) {
	switch currency {
	case EUR, USD:
		return 2, nil
	default:
		return 0, fmt.Errorf("unsupported currency %q", currency)
	}
}

func parseAmountMinor(input string, fractionalDigits int) (AmountMinor, error) {
	const maxFractionalDigits = 18

	if fractionalDigits < 0 || fractionalDigits > maxFractionalDigits {
		return 0, fmt.Errorf(
			"parse amount: fractional digits %d outside range 0..%d",
			fractionalDigits,
			maxFractionalDigits,
		)
	}

	normalized := strings.TrimSpace(input)
	if normalized == "" {
		return 0, fmt.Errorf("parse amount %q: empty input", input)
	}

	sign := normalized[:1]
	if sign == "+" || sign == "-" {
		normalized = normalized[1:]
	} else {
		sign = ""
	}

	wholePart, fractionalPart, hasDecimalPoint := strings.Cut(normalized, ".")

	if wholePart == "" || !containsOnlyDigits(wholePart) || (hasDecimalPoint && !containsOnlyDigits(fractionalPart)) {
		return 0, fmt.Errorf("parse amount %q: invalid syntax", input)
	}

	if len(fractionalPart) > fractionalDigits {
		return 0, fmt.Errorf(
			"parse amount %q: at most %d fractional digits allowed",
			input,
			fractionalDigits,
		)
	}

	fractionalPart += strings.Repeat("0", fractionalDigits-len(fractionalPart))

	result, err := strconv.ParseInt(sign+wholePart+fractionalPart, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse amount %q: %w", input, err)
	}

	return AmountMinor(result), nil
}

func containsOnlyDigits(value string) bool {
	if value == "" {
		return false
	}

	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}

	return true
}
