package main

import (
	"slices"
	"testing"
)

func TestFilterOutflows(t *testing.T) {
	input := []Transaction{
		{
			Description: "Salary",
			Amount: Money{
				Amount:   100_000,
				Currency: EUR,
			},
		},
		{
			Description: "Groceries",
			Amount: Money{
				Amount:   -2_500,
				Currency: EUR,
			},
		},
		{
			Description: "Zero",
			Amount: Money{
				Amount:   0,
				Currency: EUR,
			},
		},
		{
			Description: "Transport",
			Amount: Money{
				Amount:   -750,
				Currency: USD,
			},
		},
	}

	originalInput := slices.Clone(input)

	want := []Transaction{
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
	euroOutflow := Transaction{
		Description: "EUR outflow",
		Amount: Money{
			Amount:   -100,
			Currency: EUR,
		},
	}

	usdOutflow := Transaction{
		Description: "USD outflow",
		Amount: Money{
			Amount:   -200,
			Currency: USD,
		},
	}

	tests := []struct {
		name  string
		input []Transaction
		want  []Transaction
	}{
		{
			name:  "nil input",
			input: nil,
			want:  nil,
		},
		{
			name:  "empty input",
			input: []Transaction{},
			want:  []Transaction{},
		},
		{
			name: "no matching outflows",
			input: []Transaction{
				{
					Description: "Income",
					Amount: Money{
						Amount:   1_000,
						Currency: EUR,
					},
				},
				{
					Description: "Zero",
					Amount: Money{
						Amount:   0,
						Currency: EUR,
					},
				},
			},
			want: nil,
		},
		{
			name: "all transactions are outflows",
			input: []Transaction{
				euroOutflow,
				usdOutflow,
			},
			want: []Transaction{
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
	input := []Transaction{
		{
			Description: "MAXIMA",
			Amount: Money{
				Amount:   -1_000,
				Currency: EUR,
			},
		},
		{
			Description: "SPOTIFY",
			Amount: Money{
				Amount:   -999,
				Currency: EUR,
			},
		},
		{
			Description: "MAXIMA",
			Amount: Money{
				Amount:   -2_000,
				Currency: EUR,
			},
		},
		{
			Description: "",
			Amount: Money{
				Amount:   -500,
				Currency: EUR,
			},
		},
		{
			Description: "maxima",
			Amount: Money{
				Amount:   -300,
				Currency: EUR,
			},
		},
	}

	originalInput := slices.Clone(input)

	want := map[string][]Transaction{
		"MAXIMA": []Transaction{
			input[0],
			input[2],
		},
		"SPOTIFY": []Transaction{
			input[1],
		},
		"": []Transaction{
			input[3],
		},
		"maxima": []Transaction{
			input[4],
		},
	}

	got := groupTransactionsByDescription(input)

	if got == nil {
		t.Fatal("groupTransactionsByDescription() returned a nil map")
	}

	if len(got) != len(want) {
		t.Fatalf(
			"groupTransactionsByDescription() returned %d groups, want %d",
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
			"groupTransactionsByDescription() modified input: got %+v, original %+v",
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
		input []Transaction
	}{
		{
			name:  "nil input",
			input: nil,
		},
		{
			name:  "empty input",
			input: []Transaction{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := groupTransactionsByDescription(tt.input)

			if got == nil {
				t.Fatal("groupTransactionsByDescription() returned a nil map")
			}

			if len(got) != 0 {
				t.Errorf(
					"groupTransactionsByDescription() returned %d groups, want 0",
					len(got),
				)
			}
		})
	}
}

func TestSortedDescriptions(t *testing.T) {
	tests := []struct {
		name  string
		input map[string][]Transaction
		want  []string
	}{
		{
			name:  "nil map",
			input: nil,
			want:  make([]string, 0),
		},
		{
			name:  "empty map",
			input: make(map[string][]Transaction),
			want:  make([]string, 0),
		},

		{
			name: "unsorted keys",
			input: map[string][]Transaction{
				"C": nil,
				"B": nil,
				"A": nil,
			},
			want: []string{"A", "B", "C"},
		},
		{
			name: "empty uppercase and lowercase keys",
			input: map[string][]Transaction{
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
