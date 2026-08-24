package main

import (
	"money/internal/finance"
	"slices"
	"testing"
)

func TestFilterOutflows(t *testing.T) {
	input := []finance.Transaction{
		{
			Description: "Salary",
			Amount: finance.Money{
				Amount:   100_000,
				Currency: finance.EUR,
			},
		},
		{
			Description: "Groceries",
			Amount: finance.Money{
				Amount:   -2_500,
				Currency: finance.EUR,
			},
		},
		{
			Description: "Zero",
			Amount: finance.Money{
				Amount:   0,
				Currency: finance.EUR,
			},
		},
		{
			Description: "Transport",
			Amount: finance.Money{
				Amount:   -750,
				Currency: finance.USD,
			},
		},
	}

	originalInput := slices.Clone(input)

	want := []finance.Transaction{
		input[1],
		input[3],
	}

	got := filterOutflows(input)

	if !slices.Equal(got, want) {
		t.Fatalf(
			"filterOutflows() = %+v, want %+v",
			got,
			want,
		)
	}

	if !slices.Equal(input, originalInput) {
		t.Fatalf(
			"filterOutflows() modified input: got %+v, original %+v",
			input,
			originalInput,
		)
	}

	got[0].Description = "changed result"

	if !slices.Equal(input, originalInput) {
		t.Errorf(
			"changing result modified input: got %+v, original %+v",
			input,
			originalInput,
		)
	}
}

func TestFilterOutflowsBoundaryCases(t *testing.T) {
	euroOutflow := finance.Transaction{
		Description: "EUR outflow",
		Amount: finance.Money{
			Amount:   -100,
			Currency: finance.EUR,
		},
	}

	usdOutflow := finance.Transaction{
		Description: "USD outflow",
		Amount: finance.Money{
			Amount:   -200,
			Currency: finance.USD,
		},
	}

	tests := []struct {
		name  string
		input []finance.Transaction
		want  []finance.Transaction
	}{
		{
			name:  "nil input",
			input: nil,
			want:  nil,
		},
		{
			name:  "empty input",
			input: []finance.Transaction{},
			want:  []finance.Transaction{},
		},
		{
			name: "no matching outflows",
			input: []finance.Transaction{
				{
					Description: "Income",
					Amount: finance.Money{
						Amount:   1_000,
						Currency: finance.EUR,
					},
				},
				{
					Description: "Zero",
					Amount: finance.Money{
						Amount:   0,
						Currency: finance.EUR,
					},
				},
			},
			want: nil,
		},
		{
			name: "all finance.transactions are outflows",
			input: []finance.Transaction{
				euroOutflow,
				usdOutflow,
			},
			want: []finance.Transaction{
				euroOutflow,
				usdOutflow,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterOutflows(tt.input)

			if !slices.Equal(got, tt.want) {
				t.Errorf(
					"filterOutflows() = %+v, want %+v",
					got,
					tt.want,
				)
			}
		})
	}
}

func TestGroupTransactionsByDescription(t *testing.T) {
	input := []finance.Transaction{
		{
			Description: "MAXIMA",
			Amount: finance.Money{
				Amount:   -1_000,
				Currency: finance.EUR,
			},
		},
		{
			Description: "SPOTIFY",
			Amount: finance.Money{
				Amount:   -999,
				Currency: finance.EUR,
			},
		},
		{
			Description: "MAXIMA",
			Amount: finance.Money{
				Amount:   -2_000,
				Currency: finance.EUR,
			},
		},
		{
			Description: "",
			Amount: finance.Money{
				Amount:   -500,
				Currency: finance.EUR,
			},
		},
		{
			Description: "maxima",
			Amount: finance.Money{
				Amount:   -300,
				Currency: finance.EUR,
			},
		},
	}

	originalInput := slices.Clone(input)

	want := map[string][]finance.Transaction{
		"MAXIMA": []finance.Transaction{
			input[0],
			input[2],
		},
		"SPOTIFY": []finance.Transaction{
			input[1],
		},
		"": []finance.Transaction{
			input[3],
		},
		"maxima": []finance.Transaction{
			input[4],
		},
	}

	got := groupTransactionsByDescription(input)

	if got == nil {
		t.Fatal("groupfinance.TransactionsByDescription() returned a nil map")
	}

	if len(got) != len(want) {
		t.Fatalf(
			"groupfinance.TransactionsByDescription() returned %d groups, want %d",
			len(got),
			len(want),
		)
	}

	for description, wantTransactions := range want {
		gotTransactions, exists := got[description]
		if !exists {
			t.Errorf("group %q is missing", description)
			continue
		}

		if !slices.Equal(gotTransactions, wantTransactions) {
			t.Errorf(
				"group %q = %+v, want %+v",
				description,
				gotTransactions,
				wantTransactions,
			)
		}
	}

	if !slices.Equal(input, originalInput) {
		t.Errorf(
			"groupfinance.TransactionsByDescription() modified input: got %+v, original %+v",
			input,
			originalInput,
		)
	}

	maxima, exists := got["MAXIMA"]
	if !exists || len(maxima) == 0 {
		t.Fatal(`cannot test result ownership: group "MAXIMA" is missing or empty`)
	}

	maxima[0].Description = "changed result"

	if input[0].Description != "MAXIMA" {
		t.Errorf(
			"changing grouped result modified input description to %q",
			input[0].Description,
		)
	}
}

func TestGroupTransactionsByDescriptionEmptyInput(t *testing.T) {
	tests := []struct {
		name  string
		input []finance.Transaction
	}{
		{
			name:  "nil input",
			input: nil,
		},
		{
			name:  "empty input",
			input: []finance.Transaction{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := groupTransactionsByDescription(tt.input)

			if got == nil {
				t.Fatal("groupfinance.TransactionsByDescription() returned a nil map")
			}

			if len(got) != 0 {
				t.Errorf(
					"groupfinance.TransactionsByDescription() returned %d groups, want 0",
					len(got),
				)
			}
		})
	}
}

func TestSortedDescriptions(t *testing.T) {
	tests := []struct {
		name  string
		input map[string][]finance.Transaction
		want  []string
	}{
		{
			name:  "nil map",
			input: nil,
			want:  make([]string, 0),
		},
		{
			name:  "empty map",
			input: make(map[string][]finance.Transaction),
			want:  make([]string, 0),
		},

		{
			name: "unsorted keys",
			input: map[string][]finance.Transaction{
				"C": nil,
				"B": nil,
				"A": nil,
			},
			want: []string{"A", "B", "C"},
		},
		{
			name: "empty uppercase and lowercase keys",
			input: map[string][]finance.Transaction{
				"":        nil,
				"maxima":  nil,
				"SPOTIFY": nil,
				"MAXIMA":  nil,
			},
			want: []string{
				"",
				"MAXIMA",
				"SPOTIFY",
				"maxima",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sortedDescriptions(tt.input)

			if !slices.Equal(got, tt.want) {
				t.Errorf("sortedDescriptions() = %q, want %q", got, tt.want)
			}
		})
	}
}
