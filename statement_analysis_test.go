package main

import (
	"errors"
	"strings"
	"testing"
	"time"

	"money/internal/finance"
	"money/internal/statement"
)

func TestAnalyzeStatementCashFlow(t *testing.T) {
	location := time.FixedZone("test", 2*60*60)

	tests := []struct {
		name  string
		input string
		want  statementCashFlow
	}{
		{
			name: "Swedbank",
			input: "" +
				"Account No,Date,Beneficiary,Details,Amount,Currency,D/K,Record ID,Code\n" +
				"LT123,2026-08-05,EMPLOYER,Salary,1000.00,EUR,K,record-1,TRANSFER\n" +
				"LT123,2026-08-06,MAXIMA,Groceries,250.50,EUR,D,record-2,CARD\n",
			want: statementCashFlow{
				source: statement.Swedbank,
				importSummary: statement.Summary{
					ImportedRows: 2,
					UniqueRows:   2,
				},
				totals: finance.CashFlowTotals{
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
			},
		},
		{
			name: "Revolut",
			input: "" +
				"Type,Product,Started Date,Completed Date,Description,Amount,Fee,Currency,State\n" +
				"Deposit,Current,2026-08-04 09:00:00,2026-08-04 09:01:00,Salary,1000.00,0,EUR,COMPLETED\n" +
				"Card Payment,Current,2026-08-05 10:00:00,2026-08-05 10:01:00,Groceries,-25.50,0,EUR,COMPLETED\n",
			want: statementCashFlow{
				source: statement.Revolut,
				importSummary: statement.Summary{
					ImportedRows: 2,
					UniqueRows:   2,
				},
				totals: finance.CashFlowTotals{
					Income: finance.Money{
						Amount:   100_000,
						Currency: finance.EUR,
					},
					Expenses: finance.Money{
						Amount:   2_550,
						Currency: finance.EUR,
					},
					Savings: finance.Money{
						Amount:   97_450,
						Currency: finance.EUR,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := analyzeStatementCashFlow(
				strings.NewReader(tt.input),
				location,
				finance.EUR,
			)
			if err != nil {
				t.Fatalf(
					"analyzeStatementCashFlow() returned an unexpected error: %v",
					err,
				)
			}

			if got != tt.want {
				t.Errorf(
					"analyzeStatementCashFlow() = %+v, want %+v",
					got,
					tt.want,
				)
			}
		})
	}
}

func TestAnalyzeStatementCashFlowPreservesImportSummaryOnCurrencyError(
	t *testing.T,
) {
	input := strings.NewReader(
		"" +
			"Account No,Date,Beneficiary,Details,Amount,Currency,D/K,Record ID,Code\n" +
			"LT123,2026-08-05,EMPLOYER,Salary,1000.00,EUR,K,record-1,TRANSFER\n" +
			"LT123,2026-08-06,SHOP,Purchase,25.00,USD,D,record-2,CARD\n",
	)

	got, err := analyzeStatementCashFlow(
		input,
		time.UTC,
		finance.EUR,
	)
	if err == nil {
		t.Fatal(
			"analyzeStatementCashFlow() error = nil, want non-nil",
		)
	}

	want := statementCashFlow{
		source: statement.Swedbank,
		importSummary: statement.Summary{
			ImportedRows: 2,
			UniqueRows:   2,
		},
	}

	if got != want {
		t.Errorf(
			"analyzeStatementCashFlow() = %+v, want partial diagnostics %+v",
			got,
			want,
		)
	}

	if !strings.Contains(err.Error(), "transaction 2") {
		t.Errorf(
			"analyzeStatementCashFlow() error = %q, want transaction position 2",
			err,
		)
	}
}

func TestAnalyzeStatementCashFlowPreservesUnknownFormatError(
	t *testing.T,
) {
	input := strings.NewReader(
		"Date,Description,Amount\n" +
			"2026-08-05,Groceries,-25.50\n",
	)

	got, err := analyzeStatementCashFlow(
		input,
		time.UTC,
		finance.EUR,
	)
	if err == nil {
		t.Fatal(
			"analyzeStatementCashFlow() error = nil, want non-nil",
		)
	}

	if !errors.Is(
		err,
		statement.ErrUnknownStatementFormat,
	) {
		t.Errorf(
			"analyzeStatementCashFlow() error = %v, want ErrUnknownStatementFormat",
			err,
		)
	}

	if got != (statementCashFlow{}) {
		t.Errorf(
			"analyzeStatementCashFlow() = %+v, want zero result",
			got,
		)
	}
}

func TestAnalyzeStatementCashFlowRejectsDuplicateConflict(
	t *testing.T,
) {
	input := strings.NewReader(
		"" +
			"Account No,Date,Beneficiary,Details,Amount,Currency,D/K,Record ID,Code\n" +
			"LT123,2026-08-05,MAXIMA,Purchase,25.00,EUR,D,record-1,CARD\n" +
			"LT123,2026-08-05,MAXIMA,Purchase,999.00,EUR,D,record-1,CARD\n",
	)

	got, err := analyzeStatementCashFlow(
		input,
		time.UTC,
		finance.EUR,
	)
	if err == nil {
		t.Fatal(
			"analyzeStatementCashFlow() error = nil, want non-nil",
		)
	}

	if !errors.Is(err, statement.ErrDuplicateConflict) {
		t.Errorf(
			"analyzeStatementCashFlow() error = %v, want ErrDuplicateConflict",
			err,
		)
	}

	want := statementCashFlow{
		source: statement.Swedbank,
		importSummary: statement.Summary{
			ImportedRows: 2,
			UniqueRows:   1,
			ConflictRows: 1,
		},
	}

	if got != want {
		t.Errorf(
			"analyzeStatementCashFlow() = %+v, want diagnostics %+v",
			got,
			want,
		)
	}
}
