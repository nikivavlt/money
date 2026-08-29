package statement

import (
	"bytes"
	"testing"
)

func FuzzReadImportedStatement(f *testing.F) {
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
		imported, err := readImportedStatement(
			bytes.NewReader(data),
		)

		if err != nil {
			if !importedStatementIsZero(imported) {
				t.Errorf("readImportedStatement() returned partial result with error %v: %+v", err, imported)
			}

			return
		}

		switch imported.source {
		case Revolut, Swedbank:
		default:
			t.Fatalf("readImportedStatement() succeeded with invalid source %q", imported.source)
		}

		if len(imported.transactions) != len(imported.rawRecords) {
			t.Fatalf("transaction count = %d, raw record count = %d", len(imported.transactions), len(imported.rawRecords))
		}

		for index, transaction := range imported.transactions {
			if transaction.source != imported.source {
				t.Errorf("transaction %d source = %q, detected source = %q", index+1, transaction.source, imported.source)
			}
		}
	})
}
