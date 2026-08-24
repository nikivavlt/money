package statement

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestReadCSVHeader(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "ordinary header",
			input: "Date,Description,Amount\n",
			want:  []string{"Date", "Description", "Amount"},
		},
		{
			name:  "quoted comma in header",
			input: `Date,"Original, Description",Amount` + "\n",
			want:  []string{"Date", "Original, Description", "Amount"},
		},
		{
			name:  "UTF-8 BOM",
			input: "\uFEFFDate,Description,Amount\n",
			want:  []string{"Date", "Description", "Amount"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := readCSVHeader(strings.NewReader(tt.input))
			if err != nil {
				t.Fatalf("readCSVHeader() returned an unexpected error: %v", err)
			}

			if !slices.Equal(got, tt.want) {
				t.Errorf(
					"readCSVHeader() = %q, want %q",
					got,
					tt.want,
				)
			}
		})
	}
}

func TestReadCSVHeaderErrors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "empty input",
			input: "",
		},
		{
			name:  "malformed quoted header",
			input: `Date,"Description,Amount`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := readCSVHeader(strings.NewReader(tt.input))
			if err == nil {
				t.Fatal("readCSVHeader() error = nil, want non-nil")
			}
		})
	}
}

func TestReadCSVHeaderFromFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "statement.csv")

	content := []byte("Date,Description,Amount\n")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("os.WriteFile() returned an unexpected error: %v", err)
	}

	got, err := readCSVHeaderFromFile(path)
	if err != nil {
		t.Fatalf(
			"readCSVHeaderFromFile() returned an unexpected error: %v",
			err,
		)
	}

	want := []string{"Date", "Description", "Amount"}

	if !slices.Equal(got, want) {
		t.Errorf(
			"readCSVHeaderFromFile() = %q, want %q",
			got,
			want,
		)
	}
}

func TestReadCSVHeaderFromFileMissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.csv")

	_, err := readCSVHeaderFromFile(path)
	if err == nil {
		t.Fatal("readCSVHeaderFromFile() error = nil, want non-nil")
	}

	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf(
			"readCSVHeaderFromFile() error = %v, want it to match os.ErrNotExist",
			err,
		)
	}
}
