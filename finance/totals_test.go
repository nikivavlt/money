package finance

import (
	"errors"
	"strings"
	"testing"
)

func TestCalculateCashFlowTotals(t *testing.T) {
	tests := []struct {
		name         string
		transactions []Transaction
		want         CashFlowTotals
	}{
		{
			name: "income expenses and zero",
			transactions: []Transaction{
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
					Description: "Transport",
					Amount: Money{
						Amount:   -750,
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
			want: CashFlowTotals{
				Income: Money{
					Amount:   100_000,
					Currency: EUR,
				},
				Expenses: Money{
					Amount:   3_250,
					Currency: EUR,
				},
				Savings: Money{
					Amount:   96_750,
					Currency: EUR,
				},
			},
		},
		{
			name: "expenses exceed income",
			transactions: []Transaction{
				{
					Amount: Money{
						Amount:   1_000,
						Currency: EUR,
					},
				},
				{
					Amount: Money{
						Amount:   -2_500,
						Currency: EUR,
					},
				},
			},
			want: CashFlowTotals{
				Income: Money{
					Amount:   1_000,
					Currency: EUR,
				},
				Expenses: Money{
					Amount:   2_500,
					Currency: EUR,
				},
				Savings: Money{
					Amount:   -1_500,
					Currency: EUR,
				},
			},
		},
		{
			name:         "nil transactions",
			transactions: nil,
			want: CashFlowTotals{
				Income:   Money{Currency: EUR},
				Expenses: Money{Currency: EUR},
				Savings:  Money{Currency: EUR},
			},
		},
		{
			name:         "empty transactions",
			transactions: []Transaction{},
			want: CashFlowTotals{
				Income:   Money{Currency: EUR},
				Expenses: Money{Currency: EUR},
				Savings:  Money{Currency: EUR},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CalculateCashFlowTotals(
				tt.transactions,
				EUR,
			)
			if err != nil {
				t.Fatalf(
					"calculateCashFlowTotals() returned an unexpected error: %v",
					err,
				)
			}

			if got != tt.want {
				t.Errorf(
					"calculateCashFlowTotals() = %+v, want %+v",
					got,
					tt.want,
				)
			}
		})
	}
}

func TestCalculateCashFlowTotalsRejectsMixedCurrencies(t *testing.T) {
	transactions := []Transaction{
		{
			Amount: Money{
				Amount:   1_000,
				Currency: EUR,
			},
		},
		{
			Amount: Money{
				Amount:   -500,
				Currency: USD,
			},
		},
	}

	got, err := CalculateCashFlowTotals(transactions, EUR)
	if err == nil {
		t.Fatal("calculateCashFlowTotals() error = nil, want non-nil")
	}

	if !strings.Contains(err.Error(), "transaction 2") {
		t.Errorf(
			"calculateCashFlowTotals() error = %q, want transaction position 2",
			err,
		)
	}

	if got != (CashFlowTotals{}) {
		t.Errorf(
			"calculateCashFlowTotals() result = %+v, want zero result",
			got,
		)
	}
}

func TestCalculateCashFlowTotalsRejectsUnsupportedCurrency(t *testing.T) {
	got, err := CalculateCashFlowTotals(nil, Currency("GBP"))
	if err == nil {
		t.Fatal("calculateCashFlowTotals() error = nil, want non-nil")
	}

	if !errors.Is(err, ErrUnsupportedCurrency) {
		t.Errorf(
			"calculateCashFlowTotals() error = %v, want it to match ErrUnsupportedCurrency",
			err,
		)
	}

	if got != (CashFlowTotals{}) {
		t.Errorf(
			"calculateCashFlowTotals() result = %+v, want zero result",
			got,
		)
	}
}

func TestCalculateCashFlowTotalsDetectsOverflow(t *testing.T) {
	tests := []struct {
		name         string
		transactions []Transaction
	}{
		{
			name: "income accumulation",
			transactions: []Transaction{
				{
					Amount: Money{
						Amount:   maxAmountMinor,
						Currency: EUR,
					},
				},
				{
					Amount: Money{
						Amount:   1,
						Currency: EUR,
					},
				},
			},
		},
		{
			name: "minimum amount cannot become positive expense",
			transactions: []Transaction{
				{
					Amount: Money{
						Amount:   minAmountMinor,
						Currency: EUR,
					},
				},
			},
		},
		{
			name: "expense accumulation",
			transactions: []Transaction{
				{
					Amount: Money{
						Amount:   -maxAmountMinor,
						Currency: EUR,
					},
				},
				{
					Amount: Money{
						Amount:   -1,
						Currency: EUR,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CalculateCashFlowTotals(
				tt.transactions,
				EUR,
			)
			if err == nil {
				t.Fatal(
					"calculateCashFlowTotals() error = nil, want non-nil",
				)
			}

			if !errors.Is(err, ErrAmountOverflow) {
				t.Errorf(
					"calculateCashFlowTotals() error = %v, want it to match ErrAmountOverflow",
					err,
				)
			}

			if got != (CashFlowTotals{}) {
				t.Errorf(
					"calculateCashFlowTotals() result = %+v, want zero result",
					got,
				)
			}
		})
	}
}

func TestCheckedAddAmountMinor(t *testing.T) {
	tests := []struct {
		name    string
		left    AmountMinor
		right   AmountMinor
		want    AmountMinor
		wantErr bool
	}{
		{
			name:  "zero plus zero",
			left:  0,
			right: 0,
			want:  0,
		},
		{
			name:  "ordinary addition",
			left:  10,
			right: 20,
			want:  30,
		},
		{
			name:  "maximum plus zero",
			left:  maxAmountMinor,
			right: 0,
			want:  maxAmountMinor,
		},
		{
			name:  "minimum plus zero",
			left:  minAmountMinor,
			right: 0,
			want:  minAmountMinor,
		},
		{
			name:  "maximum plus negative",
			left:  maxAmountMinor,
			right: -1,
			want:  maxAmountMinor - 1,
		},
		{
			name:  "minimum plus positive",
			left:  minAmountMinor,
			right: 1,
			want:  minAmountMinor + 1,
		},
		{
			name:    "positive overflow",
			left:    maxAmountMinor,
			right:   1,
			wantErr: true,
		},
		{
			name:    "negative overflow",
			left:    minAmountMinor,
			right:   -1,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := checkedAddAmountMinor(tt.left, tt.right)

			if tt.wantErr {
				if err == nil {
					t.Fatal("checkedAddAmountMinor() error = nil, want non-nil")
				}

				if !errors.Is(err, ErrAmountOverflow) {
					t.Errorf(
						"checkedAddAmountMinor() error = %v, want ErrAmountOverflow",
						err,
					)
				}

				return
			}

			if err != nil {
				t.Fatalf(
					"checkedAddAmountMinor() returned an unexpected error: %v",
					err,
				)
			}

			if got != tt.want {
				t.Errorf(
					"checkedAddAmountMinor(%d, %d) = %d, want %d",
					tt.left,
					tt.right,
					got,
					tt.want,
				)
			}
		})
	}
}

func TestCheckedNegateAmountMinor(t *testing.T) {
	tests := []struct {
		name    string
		input   AmountMinor
		want    AmountMinor
		wantErr bool
	}{
		{
			name:  "zero",
			input: 0,
			want:  0,
		},
		{
			name:  "positive",
			input: 250,
			want:  -250,
		},
		{
			name:  "negative",
			input: -250,
			want:  250,
		},
		{
			name:  "maximum",
			input: maxAmountMinor,
			want:  -maxAmountMinor,
		},
		{
			name:    "minimum cannot be negated",
			input:   minAmountMinor,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := checkedNegateAmountMinor(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Fatal(
						"checkedNegateAmountMinor() error = nil, want non-nil",
					)
				}

				if !errors.Is(err, ErrAmountOverflow) {
					t.Errorf(
						"checkedNegateAmountMinor() error = %v, want ErrAmountOverflow",
						err,
					)
				}

				return
			}

			if err != nil {
				t.Fatalf(
					"checkedNegateAmountMinor() returned an unexpected error: %v",
					err,
				)
			}

			if got != tt.want {
				t.Errorf(
					"checkedNegateAmountMinor(%d) = %d, want %d",
					tt.input,
					got,
					tt.want,
				)
			}
		})
	}
}

func TestCheckedSubtractAmountMinor(t *testing.T) {
	tests := []struct {
		name    string
		left    AmountMinor
		right   AmountMinor
		want    AmountMinor
		wantErr bool
	}{
		{
			name:  "ordinary subtraction",
			left:  10,
			right: 3,
			want:  7,
		},
		{
			name:  "negative result",
			left:  3,
			right: 10,
			want:  -7,
		},
		{
			name:  "minimum minus minimum",
			left:  minAmountMinor,
			right: minAmountMinor,
			want:  0,
		},
		{
			name:  "negative one minus minimum",
			left:  -1,
			right: minAmountMinor,
			want:  maxAmountMinor,
		},
		{
			name:    "positive overflow",
			left:    maxAmountMinor,
			right:   -1,
			wantErr: true,
		},
		{
			name:    "negative overflow",
			left:    minAmountMinor,
			right:   1,
			wantErr: true,
		},
		{
			name:    "zero minus minimum overflows",
			left:    0,
			right:   minAmountMinor,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := checkedSubtractAmountMinor(tt.left, tt.right)

			if tt.wantErr {
				if err == nil {
					t.Fatal(
						"checkedSubtractAmountMinor() error = nil, want non-nil",
					)
				}

				if !errors.Is(err, ErrAmountOverflow) {
					t.Errorf(
						"checkedSubtractAmountMinor() error = %v, want ErrAmountOverflow",
						err,
					)
				}

				return
			}

			if err != nil {
				t.Fatalf(
					"checkedSubtractAmountMinor() returned an unexpected error: %v",
					err,
				)
			}

			if got != tt.want {
				t.Errorf(
					"checkedSubtractAmountMinor(%d, %d) = %d, want %d",
					tt.left,
					tt.right,
					got,
					tt.want,
				)
			}
		})
	}
}
