package main

import "slices"

type app struct {
	transactions []Transaction
}

func newApp(
	transactions []Transaction,
) *app {
	return &app{
		transactions: slices.Clone(transactions),
	}
}

func (a *app) addTransaction(
	transaction Transaction,
) {
	a.transactions = append(
		a.transactions,
		transaction,
	)
}

func (a *app) transactionsSnapshot() []Transaction {
	return slices.Clone(a.transactions)
}
