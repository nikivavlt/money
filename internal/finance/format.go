package finance

import (
	"fmt"
	"strconv"
	"strings"
)

func FormatMoney(money Money) (string, error) {
	minorDigits, err := MinorDigitsForCurrency(money.Currency)
	if err != nil {
		return "", fmt.Errorf("format money: %w", err)
	}

	amount := strconv.FormatInt(int64(money.Amount), 10)
	sign := ""

	if strings.HasPrefix(amount, "-") {
		sign = "-"
		amount = strings.TrimPrefix(amount, "-")
	}

	if minorDigits == 0 {
		return sign + amount + " " + string(money.Currency), nil
	}

	if len(amount) <= minorDigits {
		amount = strings.Repeat("0", minorDigits-len(amount)+1) + amount
	}

	decimalPosition := len(amount) - minorDigits

	return sign +
		amount[:decimalPosition] +
		"." +
		amount[decimalPosition:] +
		" " +
		string(money.Currency), nil
}
