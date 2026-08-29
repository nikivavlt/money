package statement

import (
	"errors"
	"fmt"
	"money/internal/finance"
	"strings"
	"time"
)

const revolutDateTimeLayout = "2006-01-02 15:04:05"

var ErrPendingTransaction = errors.New("transaction is pending")

func normalizeImportedDate(
	transaction importedTransaction,
	location *time.Location,
) (time.Time, error) {
	if location == nil {
		return time.Time{}, fmt.Errorf(
			"normalize imported date: nil location",
		)
	}

	switch transaction.source {
	case Revolut:
		return normalizeRevolutDate(transaction, location)

	case Swedbank:
		return normalizeSwedbankDate(transaction, location)

	default:
		return time.Time{}, fmt.Errorf(
			"normalize imported date: unsupported statement source %q",
			transaction.source,
		)
	}
}

func normalizeRevolutDate(
	transaction importedTransaction,
	location *time.Location,
) (time.Time, error) {
	state := strings.TrimSpace(transaction.stateText)

	switch state {
	case "COMPLETED":
		// Continue with completed-date parsing.

	case "PENDING":
		return time.Time{}, fmt.Errorf(
			"normalize Revolut date: %w",
			ErrPendingTransaction,
		)

	default:
		return time.Time{}, fmt.Errorf(
			"normalize Revolut date: unsupported state %q",
			transaction.stateText,
		)
	}

	completedAt := strings.TrimSpace(transaction.completedAtText)
	if completedAt == "" {
		return time.Time{}, fmt.Errorf(
			"normalize Revolut date: completed transaction has empty completed date",
		)
	}

	parsed, err := time.ParseInLocation(
		revolutDateTimeLayout,
		completedAt,
		location,
	)
	if err != nil {
		return time.Time{}, fmt.Errorf(
			"normalize Revolut completed date %q: %w",
			transaction.completedAtText,
			err,
		)
	}

	return time.Date(
		parsed.Year(),
		parsed.Month(),
		parsed.Day(),
		0,
		0,
		0,
		0,
		location,
	), nil
}

func normalizeSwedbankDate(
	transaction importedTransaction,
	location *time.Location,
) (time.Time, error) {
	parsed, err := parseTransactionDate(
		transaction.occurredAtText,
		location,
	)
	if err != nil {
		return time.Time{}, fmt.Errorf(
			"normalize Swedbank date: %w",
			err,
		)
	}

	return parsed, nil
}

func normalizeImportedMoney(
	transaction importedTransaction,
) (finance.Money, error) {
	switch transaction.source {
	case Revolut:
		return normalizeRevolutMoney(transaction)

	case Swedbank:
		return normalizeSwedbankMoney(transaction)

	default:
		return finance.Money{}, fmt.Errorf(
			"normalize imported money: unsupported statement source %q",
			transaction.source,
		)
	}
}

func normalizeRevolutMoney(
	transaction importedTransaction,
) (finance.Money, error) {
	currency := finance.Currency(
		strings.TrimSpace(transaction.currencyText),
	)

	amount, err := finance.ParseMoney(transaction.amountText, currency)
	if err != nil {
		return finance.Money{}, fmt.Errorf(
			"normalize Revolut amount: %w",
			err,
		)
	}

	fee, err := finance.ParseMoney(transaction.feeText, currency)
	if err != nil {
		return finance.Money{}, fmt.Errorf(
			"normalize Revolut fee: %w",
			err,
		)
	}

	if fee.Amount != 0 {
		return finance.Money{}, fmt.Errorf(
			"normalize Revolut money: nonzero fee is not supported yet",
		)
	}

	return amount, nil
}

func normalizeSwedbankMoney(
	transaction importedTransaction,
) (finance.Money, error) {
	currency := finance.Currency(
		strings.TrimSpace(transaction.currencyText),
	)

	amount, err := finance.ParseMoney(transaction.amountText, currency)
	if err != nil {
		return finance.Money{}, fmt.Errorf(
			"normalize Swedbank amount: %w",
			err,
		)
	}

	if amount.Amount < 0 {
		return finance.Money{}, fmt.Errorf(
			"normalize Swedbank money: amount must be non-negative before applying D/K",
		)
	}

	direction := strings.TrimSpace(transaction.directionText)

	switch direction {
	case "D":
		amount.Amount = -amount.Amount
		return amount, nil

	case "K":
		return amount, nil

	default:
		return finance.Money{}, fmt.Errorf(
			"normalize Swedbank money: unsupported D/K value %q",
			transaction.directionText,
		)
	}
}

func normalizeImportedTransaction(
	imported importedTransaction,
	location *time.Location,
) (finance.Transaction, error) {
	date, err := normalizeImportedDate(imported, location)
	if err != nil {
		return finance.Transaction{}, fmt.Errorf(
			"normalize transaction date: %w",
			err,
		)
	}

	amount, err := normalizeImportedMoney(imported)
	if err != nil {
		return finance.Transaction{}, fmt.Errorf(
			"normalize transaction money: %w",
			err,
		)
	}

	return finance.Transaction{
		Date:         date,
		Amount:       amount,
		Description:  imported.rawDescription,
		Counterparty: imported.counterpartyText,
	}, nil
}

func normalizeImportedTransactions(
	imported []importedTransaction,
	location *time.Location,
) ([]finance.Transaction, error) {
	normalized := make([]finance.Transaction, 0, len(imported))

	for index, transaction := range imported {
		result, err := normalizeImportedTransaction(transaction, location)
		if err != nil {
			return nil, fmt.Errorf(
				"normalize imported transaction %d: %w",
				index+1,
				err,
			)
		}

		normalized = append(normalized, result)
	}

	return normalized, nil
}
