package main

import (
	"bufio"
	"bytes"
	"strings"
	"testing"
	"time"

	"money/internal/finance"
)

func TestReadDefaultedValue(t *testing.T) {
	var output bytes.Buffer
	scanner := bufio.NewScanner(strings.NewReader("\n"))

	got, err := readDefaultedValue(scanner, &output, "Merchant", "Maxima")
	if err != nil {
		t.Fatalf("readDefaultedValue() returned an unexpected error: %v", err)
	}
	if got != "Maxima" {
		t.Errorf("value = %q, want Maxima", got)
	}
	if output.String() != "Merchant [Maxima]: " {
		t.Errorf("prompt = %q", output.String())
	}
}

func TestReadYesNo(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{input: "\n", want: false},
		{input: "no\n", want: false},
		{input: "Y\n", want: true},
		{input: "yes\n", want: true},
	}

	for _, tt := range tests {
		t.Run(strings.TrimSpace(tt.input), func(t *testing.T) {
			got, err := readYesNo(bufio.NewScanner(strings.NewReader(tt.input)), &bytes.Buffer{}, "Continue? ")
			if err != nil {
				t.Fatalf("readYesNo() returned an unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("readYesNo() = %t, want %t", got, tt.want)
			}
		})
	}
}

func TestFinanceFormatForReview(t *testing.T) {
	got, err := financeFormatForReview(ReviewTransaction{
		ID:     17,
		Date:   time.Date(2026, time.August, 29, 0, 0, 0, 0, time.UTC),
		Amount: finance.Money{Amount: -2_550, Currency: finance.EUR},
	})
	if err != nil {
		t.Fatalf("financeFormatForReview() returned an unexpected error: %v", err)
	}
	if got != "-25.50 EUR" {
		t.Errorf("formatted amount = %q, want %q", got, "-25.50 EUR")
	}
}
