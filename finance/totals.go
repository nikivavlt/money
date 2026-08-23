package finance

import (
	"errors"
	"fmt"
)

const (
	maxAmountMinor AmountMinor = 1<<63 - 1
	minAmountMinor AmountMinor = -1 << 63
)

var ErrAmountOverflow = errors.New("amount overflow")

type CashFlowTotals struct {
	Income   Money
	Expenses Money
	Savings  Money
}

func CalculateCashFlowTotals(
	transactions []Transaction,
	currency Currency,
) (CashFlowTotals, error) {
	if _, err := MinorDigitsForCurrency(currency); err != nil {
		return CashFlowTotals{},
			fmt.Errorf("calculate cash-flow totals: %w", err)
	}

	totals := CashFlowTotals{
		Income: Money{
			Currency: currency,
		},
		Expenses: Money{
			Currency: currency,
		},
		Savings: Money{
			Currency: currency,
		},
	}

	for index, transaction := range transactions {
		if transaction.Amount.Currency != currency {
			return CashFlowTotals{}, fmt.Errorf(
				"calculate cash-flow totals: transaction %d has currency %q, want %q",
				index+1,
				transaction.Amount.Currency,
				currency,
			)
		}

		amount := transaction.Amount.Amount

		switch {
		case amount > 0:
			income, err := checkedAddAmountMinor(
				totals.Income.Amount,
				amount,
			)
			if err != nil {
				return CashFlowTotals{}, fmt.Errorf(
					"calculate cash-flow totals: transaction %d income: %w",
					index+1,
					err,
				)
			}

			totals.Income.Amount = income

		case amount < 0:
			expense, err := checkedNegateAmountMinor(amount)
			if err != nil {
				return CashFlowTotals{}, fmt.Errorf(
					"calculate cash-flow totals: transaction %d expense: %w",
					index+1,
					err,
				)
			}

			expenses, err := checkedAddAmountMinor(
				totals.Expenses.Amount,
				expense,
			)
			if err != nil {
				return CashFlowTotals{}, fmt.Errorf(
					"calculate cash-flow totals: transaction %d expenses: %w",
					index+1,
					err,
				)
			}

			totals.Expenses.Amount = expenses
		}
	}

	savings, err := checkedSubtractAmountMinor(
		totals.Income.Amount,
		totals.Expenses.Amount,
	)
	if err != nil {
		return CashFlowTotals{},
			fmt.Errorf("calculate cash-flow totals: savings: %w", err)
	}

	totals.Savings.Amount = savings

	return totals, nil
}

func checkedAddAmountMinor(
	left AmountMinor,
	right AmountMinor,
) (AmountMinor, error) {
	if right > 0 && left > maxAmountMinor-right {
		return 0, fmt.Errorf(
			"%w: %d + %d",
			ErrAmountOverflow,
			left,
			right,
		)
	}

	if right < 0 && left < minAmountMinor-right {
		return 0, fmt.Errorf(
			"%w: %d + %d",
			ErrAmountOverflow,
			left,
			right,
		)
	}

	return left + right, nil
}

func checkedNegateAmountMinor(
	amount AmountMinor,
) (AmountMinor, error) {
	if amount == minAmountMinor {
		return 0, fmt.Errorf(
			"%w: cannot negate %d",
			ErrAmountOverflow,
			amount,
		)
	}

	return -amount, nil
}

func checkedSubtractAmountMinor(
	left AmountMinor,
	right AmountMinor,
) (AmountMinor, error) {
	if right > 0 && left < minAmountMinor+right {
		return 0, fmt.Errorf(
			"%w: %d - %d",
			ErrAmountOverflow,
			left,
			right,
		)
	}

	if right < 0 && left > maxAmountMinor+right {
		return 0, fmt.Errorf(
			"%w: %d - %d",
			ErrAmountOverflow,
			left,
			right,
		)
	}

	return left - right, nil
}
