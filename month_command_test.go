package main

import (
	"bytes"
	"testing"
	"time"

	"money/internal/finance"
)

func TestParseMonth(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		want      time.Time
		wantError bool
	}{
		{
			name:  "valid month",
			input: "2026-08",
			want:  time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:      "single digit month",
			input:     "2026-8",
			wantError: true,
		},
		{
			name:      "invalid month",
			input:     "2026-13",
			wantError: true,
		},
		{
			name:      "complete date",
			input:     "2026-08-01",
			wantError: true,
		},
		{
			name:      "empty",
			input:     "",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseMonth(tt.input)

			if tt.wantError {
				if err == nil {
					t.Fatal("parseMonth() error = nil, want non-nil")
				}

				if !got.IsZero() {
					t.Errorf("parseMonth() = %v, want zero time", got)
				}

				return
			}

			if err != nil {
				t.Fatalf("parseMonth() returned an unexpected error: %v", err)
			}

			if !got.Equal(tt.want) {
				t.Errorf("parseMonth() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWriteMonthlyCashFlow(t *testing.T) {
	var output bytes.Buffer

	month := time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC)

	summaries := []monthlyCashFlow{
		{
			Month:            month,
			Currency:         finance.EUR,
			TransactionCount: 3,
			Income: finance.Money{
				Amount:   100_000,
				Currency: finance.EUR,
			},
			Expenses: finance.Money{
				Amount:   25_050,
				Currency: finance.EUR,
			},
			Savings: finance.Money{
				Amount:   74_950,
				Currency: finance.EUR,
			},
		},
		{
			Month:            month,
			Currency:         finance.USD,
			TransactionCount: 1,
			Income: finance.Money{
				Amount:   0,
				Currency: finance.USD,
			},
			Expenses: finance.Money{
				Amount:   1_000,
				Currency: finance.USD,
			},
			Savings: finance.Money{
				Amount:   -1_000,
				Currency: finance.USD,
			},
		},
	}

	err := writeMonthlyCashFlow(&output, month, summaries)
	if err != nil {
		t.Fatalf("writeMonthlyCashFlow() returned an unexpected error: %v", err)
	}

	want := "" +
		"MONTH\tCURRENCY\tTRANSACTIONS\tINCOME\tEXPENSES\tSAVINGS\n" +
		"2026-08\tEUR\t3\t1000.00 EUR\t250.50 EUR\t749.50 EUR\n" +
		"2026-08\tUSD\t1\t0.00 USD\t10.00 USD\t-10.00 USD\n"

	if output.String() != want {
		t.Errorf("writeMonthlyCashFlow() output = %q, want %q", output.String(), want)
	}
}

func TestWriteMonthlyCashFlowEmpty(t *testing.T) {
	var output bytes.Buffer

	month := time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC)

	err := writeMonthlyCashFlow(&output, month, nil)
	if err != nil {
		t.Fatalf("writeMonthlyCashFlow() returned an unexpected error: %v", err)
	}

	want := "No transactions for 2026-08.\n"

	if output.String() != want {
		t.Errorf("writeMonthlyCashFlow() output = %q, want %q", output.String(), want)
	}
}
