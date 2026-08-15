package main

import "testing"

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
