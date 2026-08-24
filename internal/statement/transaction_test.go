package statement

import (
	"testing"
	"time"
)

func TestParseTransactionDate(t *testing.T) {
	vilnius, err := time.LoadLocation("Europe/Vilnius")
	if err != nil {
		t.Fatalf("load Europe/Vilnius location: %v", err)
	}

	tests := []struct {
		name     string
		input    string
		location *time.Location
		want     time.Time
		wantErr  bool
	}{
		{
			name:     "valid date in UTC",
			input:    "2026-08-04",
			location: time.UTC,
			want: time.Date(
				2026,
				time.August,
				4,
				0,
				0,
				0,
				0,
				time.UTC,
			),
		},
		{
			name:     "valid date in Europe Vilnius",
			input:    "2026-08-04",
			location: vilnius,
			want: time.Date(
				2026,
				time.August,
				4,
				0,
				0,
				0,
				0,
				vilnius,
			),
		},
		{
			name:     "trims surrounding whitespace",
			input:    " \t2026-08-04\n",
			location: time.UTC,
			want: time.Date(
				2026,
				time.August,
				4,
				0,
				0,
				0,
				0,
				time.UTC,
			),
		},
		{
			name:     "empty input",
			input:    "",
			location: time.UTC,
			wantErr:  true,
		},
		{
			name:     "whitespace-only input",
			input:    " \t\n",
			location: time.UTC,
			wantErr:  true,
		},
		{
			name:     "nil location",
			input:    "2026-08-04",
			location: nil,
			wantErr:  true,
		},
		{
			name:     "invalid calendar date",
			input:    "2026-02-30",
			location: time.UTC,
			wantErr:  true,
		},
		{
			name:     "non-padded month and day",
			input:    "2026-8-4",
			location: time.UTC,
			wantErr:  true,
		},
		{
			name:     "wrong component order",
			input:    "04-08-2026",
			location: time.UTC,
			wantErr:  true,
		},
		{
			name:     "timestamp is not a date",
			input:    "2026-08-04T12:30:00",
			location: time.UTC,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseTransactionDate(
				tt.input,
				tt.location,
			)

			if tt.wantErr {
				if err == nil {
					t.Fatalf(
						"parseTransactionDate(%q, %v) error = nil, want non-nil error",
						tt.input,
						tt.location,
					)
				}

				if !got.IsZero() {
					t.Errorf(
						"parseTransactionDate(%q, %v) returned %v with an error, want zero time",
						tt.input,
						tt.location,
						got,
					)
				}

				return
			}

			if err != nil {
				t.Fatalf(
					"parseTransactionDate(%q, %v) returned unexpected error: %v",
					tt.input,
					tt.location,
					err,
				)
			}

			if !got.Equal(tt.want) {
				t.Errorf(
					"parseTransactionDate(%q, %v) = %v, want %v",
					tt.input,
					tt.location,
					got,
					tt.want,
				)
			}

			if got.Location() != tt.location {
				t.Errorf(
					"parsed location = %v, want %v",
					got.Location(),
					tt.location,
				)
			}

			if got.Hour() != 0 ||
				got.Minute() != 0 ||
				got.Second() != 0 ||
				got.Nanosecond() != 0 {
				t.Errorf(
					"parsed time of day = %02d:%02d:%02d.%09d, want midnight",
					got.Hour(),
					got.Minute(),
					got.Second(),
					got.Nanosecond(),
				)
			}
		})
	}
}
