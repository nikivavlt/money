package main

import "testing"

func TestTransactionFlowDirection(t *testing.T) {
	tests := []struct {
		name        string
		amount      AmountMinor
		wantInflow  bool
		wantOutflow bool
	}{
		{
			name:        "positive amount is inflow",
			amount:      100,
			wantInflow:  true,
			wantOutflow: false,
		},
		{
			name:        "negative amount is outflow",
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
			tx := Transaction{
				Amount: Money{
					Amount:   tt.amount,
					Currency: EUR,
				},
			}

			gotInflow := tx.IsInflow()
			if gotInflow != tt.wantInflow {
				t.Errorf(
					"Transaction.IsInflow() with amount %d = %t, want %t",
					tt.amount,
					gotInflow,
					tt.wantInflow,
				)
			}

			gotOutflow := tx.IsOutflow()
			if gotOutflow != tt.wantOutflow {
				t.Errorf(
					"Transaction.IsOutflow() with amount %d = %t, want %t",
					tt.amount,
					gotOutflow,
					tt.wantOutflow,
				)
			}
		})
	}
}
