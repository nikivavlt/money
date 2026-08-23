package main

import "money/finance"

type AmountMinor = finance.AmountMinor
type Currency = finance.Currency
type Money = finance.Money
type UnsupportedCurrencyError = finance.UnsupportedCurrencyError

const (
	EUR = finance.EUR
	USD = finance.USD
)

var ErrUnsupportedCurrency = finance.ErrUnsupportedCurrency

func parseMoney(
	input string,
	currency Currency,
) (Money, error) {
	return finance.ParseMoney(input, currency)
}
