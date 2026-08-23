package finance

import "time"

type Transaction struct {
	Date         time.Time
	Amount       Money
	Description  string
	Counterparty string
}

func (t Transaction) IsInflow() bool {
	return t.Amount.Amount > 0
}

func (t Transaction) IsOutflow() bool {
	return t.Amount.Amount < 0
}
