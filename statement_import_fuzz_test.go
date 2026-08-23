package main

import (
	"bytes"
	"testing"
)

func FuzzImportStatement(f *testing.F) {
	f.Add([]byte(
		"Type,Product,Started Date,Completed Date,Description,Amount,Fee,Currency,State\n" +
			"Card Payment,Current,2026-08-04 10:00:00,2026-08-04 10:01:00,SHOP,-12.34,0,EUR,COMPLETED\n",
	))

	f.Add([]byte(
		"Account No,Date,Beneficiary,Details,Amount,Currency,D/K,Record ID,Code\n" +
			"LT123,2026-08-05,MAXIMA,Card purchase,25.50,EUR,D,record-123,CARD\n",
	))

	f.Add([]byte(
		"Date,Description,Amount\n" +
			"2026-08-05,Groceries,-25.50\n",
	))

	f.Add([]byte(
		`Type,Product,Started Date,Completed Date,Description,Amount,Fee,Currency,State
Card Payment,Current,2026-08-04 10:00:00,2026-08-04 10:01:00,"unterminated`,
	))

	f.Add([]byte{})

	f.Fuzz(func(t *testing.T, data []byte) {
		source, transactions, err := importStatement(
			bytes.NewReader(data),
		)

		if err != nil {
			if source != "" {
				t.Errorf(
					"importStatement() source = %q with error %v, want empty source",
					source,
					err,
				)
			}

			if transactions != nil {
				t.Errorf(
					"importStatement() returned transactions with error %v: %+v",
					err,
					transactions,
				)
			}

			return
		}

		switch source {
		case sourceRevolut, sourceSwedbank:
			// Recognized source.
		default:
			t.Fatalf(
				"importStatement() succeeded with invalid source %q",
				source,
			)
		}

		for index, transaction := range transactions {
			if transaction.source != source {
				t.Errorf(
					"transaction %d source = %q, detected source = %q",
					index+1,
					transaction.source,
					source,
				)
			}
		}
	})
}
