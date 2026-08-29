package statement

import (
	"encoding/csv"
	"fmt"
	"io"
	"slices"
)

type importedStatement struct {
	source       Source
	rawHeader    []string
	transactions []importedTransaction
	rawRecords   [][]string
}

func readImportedStatement(input io.Reader) (importedStatement, error) {
	reader := csv.NewReader(input)
	reader.FieldsPerRecord = -1

	header, err := readCSVHeaderRecord(reader)
	if err != nil {
		return importedStatement{}, fmt.Errorf("import statement: %w", err)
	}

	source, err := detectStatementSource(header)
	if err != nil {
		return importedStatement{}, fmt.Errorf("import statement: %w", err)
	}

	switch source {
	case Revolut:
		rows, rawRecords, err := readRevolutRowsAfterHeader(reader, header)
		if err != nil {
			return importedStatement{}, fmt.Errorf("import statement as Revolut: %w", err)
		}

		return importedStatement{
			source:       Revolut,
			rawHeader:    slices.Clone(header),
			transactions: revolutRowsToImportedTransactions(rows),
			rawRecords:   rawRecords,
		}, nil

	case Swedbank:
		rows, rawRecords, err := readSwedbankRowsAfterHeader(reader, header)
		if err != nil {
			return importedStatement{}, fmt.Errorf("import statement as Swedbank: %w", err)
		}

		return importedStatement{
			source:       Swedbank,
			rawHeader:    slices.Clone(header),
			transactions: swedbankRowsToImportedTransactions(rows),
			rawRecords:   rawRecords,
		}, nil

	default:
		return importedStatement{}, fmt.Errorf("import statement: detected unsupported source %q", source)
	}
}

func importStatement(input io.Reader) (Source, []importedTransaction, error) {
	imported, err := readImportedStatement(input)
	if err != nil {
		return "", nil, err
	}

	return imported.source, imported.transactions, nil
}
