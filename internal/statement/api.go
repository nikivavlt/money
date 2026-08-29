package statement

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"money/internal/finance"
	"slices"
	"time"
)

type Fingerprint [sha256.Size]byte

type Summary struct {
	ImportedRows  int
	UniqueRows    int
	DuplicateRows int
	ConflictRows  int
}

func (s Summary) HasConflicts() bool {
	return s.ConflictRows > 0
}

type PreparedTransaction struct {
	Fingerprint Fingerprint
	Transaction finance.Transaction
	RawRecord   []string
}

type Prepared struct {
	Source       Source
	Fingerprint  Fingerprint
	RawHeader    []string
	Transactions []PreparedTransaction
	Summary      Summary
}

func Prepare(input io.Reader, location *time.Location) (Prepared, error) {
	if input == nil {
		return Prepared{}, fmt.Errorf("prepare statement: nil input")
	}

	rawFile, err := io.ReadAll(input)
	if err != nil {
		return Prepared{}, fmt.Errorf("prepare statement: read input: %w", err)
	}

	internal, prepareErr := prepareStatementImport(bytes.NewReader(rawFile), location)
	if prepareErr != nil && internal.source == "" {
		return Prepared{}, prepareErr
	}

	var transactions []PreparedTransaction

	if internal.transactions != nil {
		transactions = make(
			[]PreparedTransaction,
			len(internal.transactions),
		)

		for index, transaction := range internal.transactions {
			transactions[index] = PreparedTransaction{
				Fingerprint: Fingerprint(
					transaction.identity.digest,
				),
				Transaction: transaction.normalized,
				RawRecord: slices.Clone(
					transaction.rawRecord,
				),
			}
		}
	}

	prepared := Prepared{
		Source:       internal.source,
		Fingerprint:  Fingerprint(sha256.Sum256(rawFile)),
		RawHeader:    slices.Clone(internal.rawHeader),
		Transactions: transactions,
		Summary: Summary{
			ImportedRows:  internal.summary.importedRows,
			UniqueRows:    internal.summary.uniqueRows,
			DuplicateRows: internal.summary.duplicateRows,
			ConflictRows:  internal.summary.conflictRows,
		},
	}

	return prepared, prepareErr
}
