package main

import (
	"errors"
	"strconv"
	"testing"
)

func TestParseMoney(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		currency Currency
		want     Money
		wantErr  bool
	}{
		{
			name:     "valid EUR",
			input:    "12.34",
			currency: EUR,
			want: Money{
				Amount:   1234,
				Currency: EUR,
			},
		},
		{
			name:     "valid negative EUR",
			input:    "-4.50",
			currency: EUR,
			want: Money{
				Amount:   -450,
				Currency: EUR,
			},
		},
		{
			name:     "valid USD",
			input:    "12.34",
			currency: USD,
			want: Money{
				Amount:   1234,
				Currency: USD,
			},
		},
		{
			name:     "USD pads one fractional digit",
			input:    "12.3",
			currency: USD,
			want: Money{
				Amount:   1230,
				Currency: USD,
			},
		},
		{
			name:     "valid negative USD",
			input:    "-19.99",
			currency: USD,
			want: Money{
				Amount:   -1999,
				Currency: USD,
			},
		},
		{
			name:     "zero USD",
			input:    "0.00",
			currency: USD,
			want: Money{
				Amount:   0,
				Currency: USD,
			},
		},
		{
			name:     "EUR rejects more than two fractional digits",
			input:    "12.345",
			currency: EUR,
			wantErr:  true,
		},
		{
			name:     "USD rejects more than two fractional digits",
			input:    "12.345",
			currency: USD,
			wantErr:  true,
		},
		{
			name:     "USD amount rejects currency symbol",
			input:    "$12.34",
			currency: USD,
			wantErr:  true,
		},
		{
			name:     "unsupported currency",
			input:    "12.34",
			currency: Currency("GBP"),
			wantErr:  true,
		},
		{
			name:     "lowercase EUR is not canonical",
			input:    "12.34",
			currency: Currency("eur"),
			wantErr:  true,
		},
		{
			name:     "lowercase USD is not canonical",
			input:    "12.34",
			currency: Currency("usd"),
			wantErr:  true,
		},
		{
			name:     "empty currency",
			input:    "12.34",
			currency: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseMoney(tt.input, tt.currency)

			if tt.wantErr {
				if err == nil {
					t.Fatalf(
						"parseMoney(%q, %q) error = nil, want non-nil error; result = %+v",
						tt.input,
						tt.currency,
						got,
					)
				}

				if got != (Money{}) {
					t.Errorf(
						"parseMoney(%q, %q) returned %+v with an error, want zero Money",
						tt.input,
						tt.currency,
						got,
					)
				}

				return
			}

			if err != nil {
				t.Fatalf(
					"parseMoney(%q, %q) returned unexpected error: %v",
					tt.input,
					tt.currency,
					err,
				)
			}

			if got != tt.want {
				t.Errorf(
					"parseMoney(%q, %q) = %+v, want %+v",
					tt.input,
					tt.currency,
					got,
					tt.want,
				)
			}
		})
	}
}

func TestParseAmountMinor(t *testing.T) {
	tests := []struct {
		name             string
		input            string
		fractionalDigits int
		want             AmountMinor
		wantErr          bool
	}{
		{
			name:             "two fractional digits",
			input:            "12.34",
			fractionalDigits: 2,
			want:             1234,
		},
		{
			name:             "pads one fractional digit",
			input:            "12.3",
			fractionalDigits: 2,
			want:             1230,
		},
		{
			name:             "pads missing fractional part",
			input:            "12",
			fractionalDigits: 2,
			want:             1200,
		},
		{
			name:             "negative amount",
			input:            "-4.50",
			fractionalDigits: 2,
			want:             -450,
		},
		{
			name:             "explicit positive sign",
			input:            "+4.50",
			fractionalDigits: 2,
			want:             450,
		},
		{
			name:             "trims surrounding whitespace",
			input:            " \t12.34\n",
			fractionalDigits: 2,
			want:             1234,
		},
		{
			name:             "zero fractional digits",
			input:            "123",
			fractionalDigits: 0,
			want:             123,
		},
		{
			name:             "three fractional digits",
			input:            "1.234",
			fractionalDigits: 3,
			want:             1234,
		},
		{
			name:             "negative zero becomes zero",
			input:            "-0.00",
			fractionalDigits: 2,
			want:             0,
		},
		{
			name:             "leading zeroes",
			input:            "00012.30",
			fractionalDigits: 2,
			want:             1230,
		},
		{
			name:             "maximum int64",
			input:            "92233720368547758.07",
			fractionalDigits: 2,
			want:             AmountMinor(9_223_372_036_854_775_807),
		},
		{
			name:             "minimum int64",
			input:            "-92233720368547758.08",
			fractionalDigits: 2,
			want:             AmountMinor(-9_223_372_036_854_775_808),
		},
		{
			name:             "maximum supported fractional digits",
			input:            "0.000000000000000001",
			fractionalDigits: 18,
			want:             1,
		},
		{
			name:             "empty input",
			input:            "",
			fractionalDigits: 2,
			wantErr:          true,
		},
		{
			name:             "whitespace only",
			input:            " \t\n",
			fractionalDigits: 2,
			wantErr:          true,
		},
		{
			name:             "positive sign without amount",
			input:            "+",
			fractionalDigits: 2,
			wantErr:          true,
		},
		{
			name:             "negative sign without amount",
			input:            "-",
			fractionalDigits: 2,
			wantErr:          true,
		},
		{
			name:             "missing whole part",
			input:            ".50",
			fractionalDigits: 2,
			wantErr:          true,
		},
		{
			name:             "missing fractional part",
			input:            "12.",
			fractionalDigits: 2,
			wantErr:          true,
		},
		{
			name:             "too many fractional digits",
			input:            "1.234",
			fractionalDigits: 2,
			wantErr:          true,
		},
		{
			name:             "fraction forbidden for zero scale",
			input:            "12.0",
			fractionalDigits: 0,
			wantErr:          true,
		},
		{
			name:             "multiple decimal points",
			input:            "1.2.3",
			fractionalDigits: 2,
			wantErr:          true,
		},
		{
			name:             "comma decimal separator",
			input:            "1,23",
			fractionalDigits: 2,
			wantErr:          true,
		},
		{
			name:             "currency symbol",
			input:            "€1.23",
			fractionalDigits: 2,
			wantErr:          true,
		},
		{
			name:             "exponent notation",
			input:            "1e3",
			fractionalDigits: 2,
			wantErr:          true,
		},
		{
			name:             "numeric underscore",
			input:            "1_000.00",
			fractionalDigits: 2,
			wantErr:          true,
		},
		{
			name:             "non ASCII digits",
			input:            "１２.３４",
			fractionalDigits: 2,
			wantErr:          true,
		},
		{
			name:             "negative fractional digit count",
			input:            "1",
			fractionalDigits: -1,
			wantErr:          true,
		},
		{
			name:             "fractional digit count exceeds limit",
			input:            "1",
			fractionalDigits: 19,
			wantErr:          true,
		},
		{
			name:             "positive overflow",
			input:            "92233720368547758.08",
			fractionalDigits: 2,
			wantErr:          true,
		},
		{
			name:             "negative overflow",
			input:            "-92233720368547758.09",
			fractionalDigits: 2,
			wantErr:          true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseAmountMinor(tt.input, tt.fractionalDigits)

			if tt.wantErr {
				if err == nil {
					t.Fatalf(
						"parseAmountMinor(%q, %d) error = nil, want non-nil error; result = %d",
						tt.input,
						tt.fractionalDigits,
						got,
					)
				}

				return
			}

			if err != nil {
				t.Fatalf(
					"parseAmountMinor(%q, %d) returned unexpected error: %v",
					tt.input,
					tt.fractionalDigits,
					err,
				)
			}

			if got != tt.want {
				t.Errorf(
					"parseAmountMinor(%q, %d) = %d, want %d",
					tt.input,
					tt.fractionalDigits,
					got,
					tt.want,
				)
			}
		})
	}
}

func TestParseMoneyUnsupportedCurrency(t *testing.T) {
	const unsupported Currency = "GBP"

	_, err := parseMoney("12.34", unsupported)
	if err == nil {
		t.Fatal("parseMoney() returned nil error, want an error")
	}

	if !errors.Is(err, ErrUnsupportedCurrency) {
		t.Errorf(
			"parseMoney() error = %v, want it to match ErrUnsupportedCurrency",
			err,
		)
	}

	var currencyErr *UnsupportedCurrencyError
	if !errors.As(err, &currencyErr) {
		t.Fatalf(
			"parseMoney() error type = %T, want *UnsupportedCurrencyError",
			err,
		)
	}

	if currencyErr.Currency != unsupported {
		t.Errorf(
			"UnsupportedCurrencyError.Currency = %q, want %q",
			currencyErr.Currency,
			unsupported,
		)
	}

	wantMessage := `parse money: unsupported currency "GBP"`
	if err.Error() != wantMessage {
		t.Errorf(
			"parseMoney() error = %q, want %q",
			err,
			wantMessage,
		)
	}
}

func TestParseMoneyRejectsOverflow(t *testing.T) {
	_, err := parseMoney("92233720368547758.08", EUR)
	if err == nil {
		t.Fatal("parseMoney() returned nil error for an overflowing amount")
	}

	if !errors.Is(err, strconv.ErrRange) {
		t.Errorf(
			"parseMoney() error = %v, want it to match strconv.ErrRange",
			err,
		)
	}
}
