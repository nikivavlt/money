package finance

import "testing"

func TestTransactionFlow(t *testing.T) {
	tests := []struct {
		name        string
		amount      AmountMinor
		wantInflow  bool
		wantOutflow bool
	}{
		{
			name:        "positive is inflow",
			amount:      100,
			wantInflow:  true,
			wantOutflow: false,
		},
		{
			name:        "negative is outflow",
			amount:      -100,
			wantInflow:  false,
			wantOutflow: true,
		},
		{
			name:        "zero is neither",
			amount:      0,
			wantInflow:  false,
			wantOutflow: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transaction := Transaction{
				Amount: Money{
					Amount:   tt.amount,
					Currency: EUR,
				},
			}

			if got := transaction.IsInflow(); got != tt.wantInflow {
				t.Errorf(
					"IsInflow() = %t, want %t",
					got,
					tt.wantInflow,
				)
			}

			if got := transaction.IsOutflow(); got != tt.wantOutflow {
				t.Errorf(
					"IsOutflow() = %t, want %t",
					got,
					tt.wantOutflow,
				)
			}
		})
	}
}
