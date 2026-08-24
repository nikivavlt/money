package statement

import (
	"encoding/csv"
	"fmt"
	"io"
)

func importStatement(
	input io.Reader,
) (
	statementSource,
	[]importedTransaction,
	error,
) {
	reader := csv.NewReader(input)
	reader.FieldsPerRecord = -1

	header, err := readCSVHeaderRecord(reader)
	if err != nil {
		return "", nil, fmt.Errorf(
			"import statement: %w",
			err,
		)
	}

	source, err := detectStatementSource(header)
	if err != nil {
		return "", nil, fmt.Errorf(
			"import statement: %w",
			err,
		)
	}

	switch source {
	case sourceRevolut:
		rows, err := readRevolutRowsAfterHeader(
			reader,
			header,
		)
		if err != nil {
			return "", nil, fmt.Errorf(
				"import statement as Revolut: %w",
				err,
			)
		}

		return sourceRevolut,
			revolutRowsToImportedTransactions(rows),
			nil

	case sourceSwedbank:
		rows, err := readSwedbankRowsAfterHeader(
			reader,
			header,
		)
		if err != nil {
			return "", nil, fmt.Errorf(
				"import statement as Swedbank: %w",
				err,
			)
		}

		return sourceSwedbank,
			swedbankRowsToImportedTransactions(rows),
			nil

	default:
		return "", nil, fmt.Errorf(
			"import statement: detected unsupported source %q",
			source,
		)
	}
}
