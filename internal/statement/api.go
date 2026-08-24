package statement

import (
	"io"
	"time"

	"money/internal/finance"
)

type Summary struct {
	ImportedRows  int
	UniqueRows    int
	DuplicateRows int
	ConflictRows  int
}

func (s Summary) HasConflicts() bool {
	return s.ConflictRows > 0
}

type Prepared struct {
	Source       Source
	Transactions []finance.Transaction
	Summary      Summary
}

func Prepare(
	input io.Reader,
	location *time.Location,
) (Prepared, error) {
	internal, err := prepareStatementImport(
		input,
		location,
	)

	prepared := Prepared{
		Source:       internal.source,
		Transactions: internal.transactions,
		Summary: Summary{
			ImportedRows:  internal.summary.importedRows,
			UniqueRows:    internal.summary.uniqueRows,
			DuplicateRows: internal.summary.duplicateRows,
			ConflictRows:  internal.summary.conflictRows,
		},
	}

	return prepared, err
}
