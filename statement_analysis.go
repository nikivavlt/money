package main

import (
	"fmt"
	"io"
	"time"

	"money/internal/finance"
	"money/internal/statement"
)

type statementCashFlow struct {
	source        statement.Source
	importSummary statement.Summary
	totals        finance.CashFlowTotals
}

func analyzeStatementCashFlow(
	input io.Reader,
	location *time.Location,
	currency finance.Currency,
) (statementCashFlow, error) {
	prepared, err := statement.Prepare(
		input,
		location,
	)

	result := statementCashFlow{
		source:        prepared.Source,
		importSummary: prepared.Summary,
	}

	if err != nil {
		return result, fmt.Errorf(
			"analyze statement cash flow: %w",
			err,
		)
	}

	totals, err := finance.CalculateCashFlowTotals(
		prepared.Transactions,
		currency,
	)
	if err != nil {
		return result, fmt.Errorf(
			"analyze statement cash flow: %w",
			err,
		)
	}

	result.totals = totals

	return result, nil
}
