package finance

import (
	"errors"
	"testing"
)

func TestFormatMoney(t *testing.T) {
	tests := []struct {
		name  string
		money Money
		want  string
	}{
		{
			name:  "zero",
			money: Money{Amount: 0, Currency: EUR},
			want:  "0.00 EUR",
		},
		{
			name:  "one minor unit",
			money: Money{Amount: 1, Currency: EUR},
			want:  "0.01 EUR",
		},
		{
			name:  "negative one minor unit",
			money: Money{Amount: -1, Currency: EUR},
			want:  "-0.01 EUR",
		},
		{
			name:  "positive amount",
			money: Money{Amount: 1234, Currency: EUR},
			want:  "12.34 EUR",
		},
		{
			name:  "negative amount",
			money: Money{Amount: -1234, Currency: USD},
			want:  "-12.34 USD",
		},
		{
			name: "minimum int64",
			money: Money{
				Amount:   AmountMinor(-9_223_372_036_854_775_808),
				Currency: EUR,
			},
			want: "-92233720368547758.08 EUR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FormatMoney(tt.money)
			if err != nil {
				t.Fatalf("FormatMoney() returned an unexpected error: %v", err)
			}

			if got != tt.want {
				t.Errorf("FormatMoney() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatMoneyRejectsUnsupportedCurrency(t *testing.T) {
	got, err := FormatMoney(Money{
		Amount:   100,
		Currency: Currency("GBP"),
	})
	if err == nil {
		t.Fatal("FormatMoney() error = nil, want non-nil")
	}

	if !errors.Is(err, ErrUnsupportedCurrency) {
		t.Errorf("FormatMoney() error = %v, want ErrUnsupportedCurrency", err)
	}

	if got != "" {
		t.Errorf("FormatMoney() = %q, want empty string", got)
	}
}
