package main

import (
	"fmt"
	"io"
)

type importSummary struct {
	importedRows  int
	uniqueRows    int
	duplicateRows int
	conflictRows  int
}

func summarizeDeduplication(
	imported []importedTransaction,
	result deduplicationResult,
) importSummary {
	return importSummary{
		importedRows:  len(imported),
		uniqueRows:    len(result.unique),
		duplicateRows: len(result.duplicates),
		conflictRows:  len(result.conflicts),
	}
}

func (s importSummary) hasConflicts() bool {
	return s.conflictRows > 0
}

func (s importSummary) isConsistent() bool {
	classifiedRows := s.uniqueRows +
		s.duplicateRows +
		s.conflictRows

	return s.importedRows == classifiedRows
}

func writeImportSummary(
	output io.Writer,
	summary importSummary,
) error {
	_, err := fmt.Fprintf(
		output,
		"Imported rows:  %d\n"+
			"Unique rows:    %d\n"+
			"Duplicates:     %d\n"+
			"Conflicts:      %d\n",
		summary.importedRows,
		summary.uniqueRows,
		summary.duplicateRows,
		summary.conflictRows,
	)
	if err != nil {
		return fmt.Errorf("write import summary: %w", err)
	}

	return nil
}
