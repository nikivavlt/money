package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func runImportCommand(
	ctx context.Context,
	path string,
	output io.Writer,
	getenv func(string) string,
) error {
	userID, err := parseImportUserID(getenv("MONEY_USER_ID"))
	if err != nil {
		return err
	}

	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open statement file %q: %w", path, err)
	}
	defer file.Close()

	store, err := openCommandStore(ctx, getenv)
	if err != nil {
		return err
	}
	defer store.db.Close()

	result, err := importStatement(
		ctx,
		store,
		userID,
		filepath.Base(path),
		file,
		time.Local,
	)
	if err != nil {
		return fmt.Errorf("import statement file %q: %w", path, err)
	}

	if err := writeStatementImportResult(output, result); err != nil {
		return err
	}

	return nil
}

func parseImportUserID(input string) (int64, error) {
	normalized := strings.TrimSpace(input)
	if normalized == "" {
		return 0, fmt.Errorf("MONEY_USER_ID is empty")
	}

	userID, err := strconv.ParseInt(normalized, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse MONEY_USER_ID %q: %w", input, err)
	}

	if userID < 1 {
		return 0, fmt.Errorf("MONEY_USER_ID must be positive")
	}

	return userID, nil
}

func writeStatementImportResult(
	output io.Writer,
	result statementImportResult,
) error {
	_, err := fmt.Fprintf(
		output,
		"Statement ID:        %d\n"+
			"Stored transactions: %d\n"+
			"Imported rows:       %d\n"+
			"Unique rows:         %d\n"+
			"Duplicates:          %d\n"+
			"Conflicts:           %d\n",
		result.Stored.Statement.ID,
		len(result.Stored.Transactions),
		result.Summary.ImportedRows,
		result.Summary.UniqueRows,
		result.Summary.DuplicateRows,
		result.Summary.ConflictRows,
	)
	if err != nil {
		return fmt.Errorf("write import result: %w", err)
	}

	return nil
}
