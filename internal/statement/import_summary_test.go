package statement

import (
	"bytes"
	"errors"
	"testing"
)

var errImportSummaryWrite = errors.New("summary writer failed")

type importSummaryFailingWriter struct{}

func (importSummaryFailingWriter) Write(data []byte) (int, error) {
	return 0, errImportSummaryWrite
}

func TestSummarizeDeduplication(t *testing.T) {
	imported := []importedTransaction{
		{externalID: "1"},
		{externalID: "2"},
		{externalID: "2"},
		{externalID: "3"},
	}

	result := deduplicationResult{
		unique: []importedTransaction{
			imported[0],
			imported[1],
		},
		duplicates: []duplicateCandidate{
			{
				transaction:       imported[2],
				firstPosition:     2,
				duplicatePosition: 3,
			},
		},
		conflicts: []duplicateConflict{
			{
				firstTransaction:       imported[0],
				conflictingTransaction: imported[3],
				firstPosition:          1,
				conflictingPosition:    4,
			},
		},
	}

	got := summarizeDeduplication(imported, result)

	want := importSummary{
		importedRows:  4,
		uniqueRows:    2,
		duplicateRows: 1,
		conflictRows:  1,
	}

	if got != want {
		t.Errorf(
			"summarizeDeduplication() = %+v, want %+v",
			got,
			want,
		)
	}
}

func TestImportSummaryHasConflicts(t *testing.T) {
	tests := []struct {
		name    string
		summary importSummary
		want    bool
	}{
		{
			name:    "zero summary",
			summary: importSummary{},
			want:    false,
		},
		{
			name: "duplicates are not conflicts",
			summary: importSummary{
				duplicateRows: 2,
			},
			want: false,
		},
		{
			name: "one conflict",
			summary: importSummary{
				conflictRows: 1,
			},
			want: true,
		},
		{
			name: "multiple conflicts",
			summary: importSummary{
				conflictRows: 3,
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.summary.hasConflicts()

			if got != tt.want {
				t.Errorf(
					"hasConflicts() = %t, want %t",
					got,
					tt.want,
				)
			}
		})
	}
}

func TestImportSummaryIsConsistent(t *testing.T) {
	tests := []struct {
		name    string
		summary importSummary
		want    bool
	}{
		{
			name:    "zero summary",
			summary: importSummary{},
			want:    true,
		},
		{
			name: "all rows are unique",
			summary: importSummary{
				importedRows: 3,
				uniqueRows:   3,
			},
			want: true,
		},
		{
			name: "mixed classifications",
			summary: importSummary{
				importedRows:  5,
				uniqueRows:    2,
				duplicateRows: 2,
				conflictRows:  1,
			},
			want: true,
		},
		{
			name: "one row is unaccounted for",
			summary: importSummary{
				importedRows:  5,
				uniqueRows:    2,
				duplicateRows: 1,
				conflictRows:  1,
			},
			want: false,
		},
		{
			name: "classified rows exceed imported rows",
			summary: importSummary{
				importedRows:  2,
				uniqueRows:    2,
				duplicateRows: 1,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.summary.isConsistent()

			if got != tt.want {
				t.Errorf(
					"isConsistent() = %t, want %t",
					got,
					tt.want,
				)
			}
		})
	}
}

func TestWriteImportSummary(t *testing.T) {
	summary := importSummary{
		importedRows:  143,
		uniqueRows:    138,
		duplicateRows: 4,
		conflictRows:  1,
	}

	var output bytes.Buffer

	err := writeImportSummary(&output, summary)
	if err != nil {
		t.Fatalf(
			"writeImportSummary() returned an unexpected error: %v",
			err,
		)
	}

	want := "" +
		"Imported rows:  143\n" +
		"Unique rows:    138\n" +
		"Duplicates:     4\n" +
		"Conflicts:      1\n"

	if got := output.String(); got != want {
		t.Errorf(
			"writeImportSummary() output = %q, want %q",
			got,
			want,
		)
	}
}

func TestWriteImportSummaryPropagatesWriterError(t *testing.T) {
	err := writeImportSummary(
		importSummaryFailingWriter{},
		importSummary{
			importedRows: 1,
			uniqueRows:   1,
		},
	)
	if err == nil {
		t.Fatal("writeImportSummary() error = nil, want non-nil")
	}

	if !errors.Is(err, errImportSummaryWrite) {
		t.Errorf(
			"writeImportSummary() error = %v, want it to match %v",
			err,
			errImportSummaryWrite,
		)
	}
}
