package main

import (
	"money/internal/finance"
	"slices"
)

type app struct {
	transactions []finance.Transaction
}

func newApp(
	transactions []finance.Transaction,
) *app {
	return &app{
		transactions: slices.Clone(transactions),
	}
}

func (a *app) addTransaction(
	transaction finance.Transaction,
) {
	a.transactions = append(
		a.transactions,
		transaction,
	)
}

func (a *app) transactionsSnapshot() []finance.Transaction {
	return slices.Clone(a.transactions)
}
