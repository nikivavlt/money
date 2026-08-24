package finance

import (
	"strconv"
	"strings"
	"testing"
)

func FuzzParseAmountMinorRoundTrip(f *testing.F) {
	f.Add(int64(0), uint8(2))
	f.Add(int64(1_234), uint8(2))
	f.Add(int64(-450), uint8(2))
	f.Add(int64(9_223_372_036_854_775_807), uint8(2))
	f.Add(int64(-9_223_372_036_854_775_808), uint8(2))
	f.Add(int64(1), uint8(18))

	f.Fuzz(func(
		t *testing.T,
		amount int64,
		rawFractionalDigits uint8,
	) {
		fractionalDigits := int(rawFractionalDigits % 19)

		input := formatAmountMinorForFuzz(
			amount,
			fractionalDigits,
		)

		got, err := parseAmountMinor(input, fractionalDigits)
		if err != nil {
			t.Fatalf(
				"parseAmountMinor(%q, %d) returned an unexpected error: %v",
				input,
				fractionalDigits,
				err,
			)
		}

		if got != AmountMinor(amount) {
			t.Errorf(
				"parseAmountMinor(%q, %d) = %d, want %d",
				input,
				fractionalDigits,
				got,
				amount,
			)
		}
	})
}

func formatAmountMinorForFuzz(
	amount int64,
	fractionalDigits int,
) string {
	text := strconv.FormatInt(amount, 10)

	sign := ""
	if strings.HasPrefix(text, "-") {
		sign = "-"
		text = strings.TrimPrefix(text, "-")
	}

	if fractionalDigits == 0 {
		return sign + text
	}

	if len(text) <= fractionalDigits {
		text = strings.Repeat(
			"0",
			fractionalDigits-len(text)+1,
		) + text
	}

	decimalPosition := len(text) - fractionalDigits

	return sign +
		text[:decimalPosition] +
		"." +
		text[decimalPosition:]
}
