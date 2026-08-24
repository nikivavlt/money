package statement

import (
	"errors"
	"testing"
)

func TestDetectStatementSource(t *testing.T) {
	tests := []struct {
		name   string
		header []string
		want   statementSource
	}{
		{
			name: "Revolut",
			header: []string{
				"Type",
				"Product",
				"Started Date",
				"Completed Date",
				"Description",
				"Amount",
				"Fee",
				"Currency",
				"State",
				"Balance",
			},
			want: sourceRevolut,
		},
		{
			name: "Swedbank",
			header: []string{
				"Account No",
				"",
				"Date",
				"Beneficiary",
				"Details",
				"Amount",
				"Currency",
				"D/K",
				"Record ID",
				"Code",
				"Reference No",
				"Doc. No",
				"Code in payer IS",
				"Client code",
				"Originator",
				"Beneficiary party",
				"",
			},
			want: sourceSwedbank,
		},
		{
			name: "reordered Revolut headers",
			header: []string{
				"Currency",
				"State",
				"Description",
				"Amount",
				"Completed Date",
				"Product",
				"Fee",
				"Started Date",
				"Type",
			},
			want: sourceRevolut,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := detectStatementSource(tt.header)
			if err != nil {
				t.Fatalf(
					"detectStatementSource() returned an unexpected error: %v",
					err,
				)
			}

			if got != tt.want {
				t.Errorf(
					"detectStatementSource() = %q, want %q",
					got,
					tt.want,
				)
			}
		})
	}
}

func TestDetectStatementSourceErrors(t *testing.T) {
	tests := []struct {
		name      string
		header    []string
		wantError error
	}{
		{
			name: "unknown format",
			header: []string{
				"Date",
				"Description",
				"Amount",
				"Currency",
			},
			wantError: ErrUnknownStatementFormat,
		},
		{
			name: "ambiguous format",
			header: []string{
				"Type",
				"Product",
				"Started Date",
				"Completed Date",
				"Description",
				"Fee",
				"State",
				"Account No",
				"Date",
				"Beneficiary",
				"Details",
				"D/K",
				"Record ID",
				"Code",
				"Amount",
				"Currency",
			},
			wantError: ErrAmbiguousStatementFormat,
		},
		{
			name:      "nil header",
			header:    nil,
			wantError: ErrUnknownStatementFormat,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := detectStatementSource(tt.header)
			if err == nil {
				t.Fatal(
					"detectStatementSource() error = nil, want non-nil",
				)
			}

			if got != "" {
				t.Errorf(
					"detectStatementSource() = %q with an error, want empty source",
					got,
				)
			}

			if !errors.Is(err, tt.wantError) {
				t.Errorf(
					"detectStatementSource() error = %v, want it to match %v",
					err,
					tt.wantError,
				)
			}
		})
	}
}
