package main

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"money/internal/statement"
)

func TestParseImportUserID(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		want      int64
		wantError string
	}{
		{
			name:  "positive ID",
			input: "42",
			want:  42,
		},
		{
			name:  "surrounding whitespace",
			input: " 42 ",
			want:  42,
		},
		{
			name:      "empty",
			input:     "",
			wantError: "MONEY_USER_ID is empty",
		},
		{
			name:      "whitespace",
			input:     " \t ",
			wantError: "MONEY_USER_ID is empty",
		},
		{
			name:      "zero",
			input:     "0",
			wantError: "MONEY_USER_ID must be positive",
		},
		{
			name:      "negative",
			input:     "-1",
			wantError: "MONEY_USER_ID must be positive",
		},
		{
			name:      "not an integer",
			input:     "abc",
			wantError: `parse MONEY_USER_ID "abc": strconv.ParseInt: parsing "abc": invalid syntax`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseImportUserID(tt.input)

			if tt.wantError == "" {
				if err != nil {
					t.Fatalf("parseImportUserID() returned an unexpected error: %v", err)
				}

				if got != tt.want {
					t.Errorf("parseImportUserID() = %d, want %d", got, tt.want)
				}

				return
			}

			if err == nil {
				t.Fatal("parseImportUserID() error = nil, want non-nil")
			}

			if err.Error() != tt.wantError {
				t.Errorf("parseImportUserID() error = %q, want %q", err, tt.wantError)
			}

			if got != 0 {
				t.Errorf("parseImportUserID() = %d, want 0", got)
			}
		})
	}
}

func TestRunImportRequiresFilePath(t *testing.T) {
	var (
		stdout bytes.Buffer
		stderr bytes.Buffer
	)

	exitCode := run(
		context.Background(),
		[]string{"import"},
		&stdout,
		&stderr,
		func(string) string {
			return ""
		},
	)

	if exitCode != 2 {
		t.Errorf("run() exit code = %d, want 2", exitCode)
	}

	if stdout.Len() != 0 {
		t.Errorf("run() stdout = %q, want empty", stdout.String())
	}

	wantError := "money: import: usage: money import <path>\n"

	if stderr.String() != wantError {
		t.Errorf("run() stderr = %q, want %q", stderr.String(), wantError)
	}
}

func TestRunImportRequiresUserID(t *testing.T) {
	var (
		stdout bytes.Buffer
		stderr bytes.Buffer
	)

	exitCode := run(
		context.Background(),
		[]string{"import", "statement.csv"},
		&stdout,
		&stderr,
		func(string) string {
			return ""
		},
	)

	if exitCode != 1 {
		t.Errorf("run() exit code = %d, want 1", exitCode)
	}

	if stdout.Len() != 0 {
		t.Errorf("run() stdout = %q, want empty", stdout.String())
	}

	if !strings.Contains(stderr.String(), "MONEY_USER_ID is empty") {
		t.Errorf("run() stderr = %q, want MONEY_USER_ID error", stderr.String())
	}
}

func TestWriteStatementImportResult(t *testing.T) {
	var output bytes.Buffer

	result := statementImportResult{
		Stored: StoredStatementImport{
			Statement: Statement{
				ID: 17,
			},
			Transactions: []StoredTransaction{
				{ID: 101},
				{ID: 102},
			},
		},
		Summary: statement.Summary{
			ImportedRows:  3,
			UniqueRows:    2,
			DuplicateRows: 1,
		},
	}

	err := writeStatementImportResult(&output, result)
	if err != nil {
		t.Fatalf("writeStatementImportResult() returned an unexpected error: %v", err)
	}

	want := "" +
		"Statement ID:        17\n" +
		"Stored transactions: 2\n" +
		"Imported rows:       3\n" +
		"Unique rows:         2\n" +
		"Duplicates:          1\n" +
		"Conflicts:           0\n"

	if output.String() != want {
		t.Errorf("writeStatementImportResult() output = %q, want %q", output.String(), want)
	}
}
