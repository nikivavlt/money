package statement

import (
	"maps"
	"slices"
	"strings"
	"testing"
)

func TestRevolutColumnIndexes(t *testing.T) {
	tests := []struct {
		name   string
		header []string
		want   map[string]int
	}{
		{
			name: "actual Revolut header",
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
			want: map[string]int{
				"Type":           0,
				"Product":        1,
				"Started Date":   2,
				"Completed Date": 3,
				"Description":    4,
				"Amount":         5,
				"Fee":            6,
				"Currency":       7,
				"State":          8,
				"Balance":        9,
			},
		},
		{
			name: "reordered header with extra column",
			header: []string{
				"Extra",
				"Currency",
				"Amount",
				"Description",
				"State",
				"Fee",
				"Completed Date",
				"Started Date",
				"Product",
				"Type",
			},
			want: map[string]int{
				"Extra":          0,
				"Currency":       1,
				"Amount":         2,
				"Description":    3,
				"State":          4,
				"Fee":            5,
				"Completed Date": 6,
				"Started Date":   7,
				"Product":        8,
				"Type":           9,
			},
		},
		{
			name: "optional balance missing",
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
			},
			want: map[string]int{
				"Type":           0,
				"Product":        1,
				"Started Date":   2,
				"Completed Date": 3,
				"Description":    4,
				"Amount":         5,
				"Fee":            6,
				"Currency":       7,
				"State":          8,
			},
		},
		{
			name: "unnamed columns ignored",
			header: []string{
				"",
				"Type",
				"Product",
				"Started Date",
				"Completed Date",
				"Description",
				"Amount",
				"Fee",
				"Currency",
				"State",
				"",
			},
			want: map[string]int{
				"Type":           1,
				"Product":        2,
				"Started Date":   3,
				"Completed Date": 4,
				"Description":    5,
				"Amount":         6,
				"Fee":            7,
				"Currency":       8,
				"State":          9,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := revolutColumnIndexes(tt.header)
			if err != nil {
				t.Fatalf(
					"revolutColumnIndexes() returned an unexpected error: %v",
					err,
				)
			}

			if !maps.Equal(got, tt.want) {
				t.Errorf(
					"revolutColumnIndexes() = %v, want %v",
					got,
					tt.want,
				)
			}
		})
	}
}

func TestRevolutColumnIndexesErrors(t *testing.T) {
	tests := []struct {
		name              string
		header            []string
		wantErrorContains string
	}{
		{
			name: "missing required amount column",
			header: []string{
				"Type",
				"Product",
				"Started Date",
				"Completed Date",
				"Description",
				"Fee",
				"Currency",
				"State",
			},
			wantErrorContains: `missing required column "Amount"`,
		},
		{
			name: "duplicate named column",
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
				"Amount",
			},
			wantErrorContains: `duplicate column "Amount"`,
		},
		{
			name:              "nil header",
			header:            nil,
			wantErrorContains: `missing required column "Type"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := revolutColumnIndexes(tt.header)
			if err == nil {
				t.Fatal("revolutColumnIndexes() error = nil, want non-nil")
			}

			if got != nil {
				t.Errorf(
					"revolutColumnIndexes() returned %v with an error, want nil map",
					got,
				)
			}

			if !strings.Contains(err.Error(), tt.wantErrorContains) {
				t.Errorf(
					"revolutColumnIndexes() error = %q, want it to contain %q",
					err,
					tt.wantErrorContains,
				)
			}
		})
	}
}

func TestRevolutRowFromRecord(t *testing.T) {
	header := []string{
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
	}

	indexes, err := revolutColumnIndexes(header)
	if err != nil {
		t.Fatalf(
			"revolutColumnIndexes() returned an unexpected error: %v",
			err,
		)
	}

	record := []string{
		"CARD_PAYMENT",
		"Current",
		"2026-08-01 10:00:00",
		"2026-08-01 10:01:00",
		"Shop, Vilnius",
		"-12.34",
		"0.00",
		"EUR",
		"COMPLETED",
		"987.66",
	}

	got, err := revolutRowFromRecord(record, indexes)
	if err != nil {
		t.Fatalf(
			"revolutRowFromRecord() returned an unexpected error: %v",
			err,
		)
	}

	want := revolutRow{
		transactionType:   "CARD_PAYMENT",
		product:           "Current",
		startedDateText:   "2026-08-01 10:00:00",
		completedDateText: "2026-08-01 10:01:00",
		rawDescription:    "Shop, Vilnius",
		amountText:        "-12.34",
		feeText:           "0.00",
		currencyText:      "EUR",
		stateText:         "COMPLETED",
		balanceText:       "987.66",
	}

	if got != want {
		t.Errorf(
			"revolutRowFromRecord() = %+v, want %+v",
			got,
			want,
		)
	}
}

func TestReadRevolutRows(t *testing.T) {
	input := strings.NewReader(
		`Type,Product,Started Date,Completed Date,Description,Amount,Fee,Currency,State,Balance
CARD_PAYMENT,Current,2026-08-01 10:00:00,2026-08-01 10:01:00,"Shop, Vilnius",-12.34,0.00,EUR,COMPLETED,987.66
TRANSFER,Current,2026-08-02 08:00:00,,Salary,1000.00,0.00,EUR,PENDING,1987.66
`,
	)

	got, err := readRevolutRows(input)
	if err != nil {
		t.Fatalf("readRevolutRows() returned an unexpected error: %v", err)
	}

	want := []revolutRow{
		{
			transactionType:   "CARD_PAYMENT",
			product:           "Current",
			startedDateText:   "2026-08-01 10:00:00",
			completedDateText: "2026-08-01 10:01:00",
			rawDescription:    "Shop, Vilnius",
			amountText:        "-12.34",
			feeText:           "0.00",
			currencyText:      "EUR",
			stateText:         "COMPLETED",
			balanceText:       "987.66",
		},
		{
			transactionType:   "TRANSFER",
			product:           "Current",
			startedDateText:   "2026-08-02 08:00:00",
			completedDateText: "",
			rawDescription:    "Salary",
			amountText:        "1000.00",
			feeText:           "0.00",
			currencyText:      "EUR",
			stateText:         "PENDING",
			balanceText:       "1987.66",
		},
	}

	if !slices.Equal(got, want) {
		t.Errorf(
			"readRevolutRows() = %+v, want %+v",
			got,
			want,
		)
	}
}

func TestReadRevolutRowsWithoutOptionalBalance(t *testing.T) {
	input := strings.NewReader(
		`Type,Product,Started Date,Completed Date,Description,Amount,Fee,Currency,State
CARD_PAYMENT,Current,2026-08-01 10:00:00,2026-08-01 10:01:00,Groceries,-25.00,0.00,EUR,COMPLETED
`,
	)

	got, err := readRevolutRows(input)
	if err != nil {
		t.Fatalf("readRevolutRows() returned an unexpected error: %v", err)
	}

	want := []revolutRow{
		{
			transactionType:   "CARD_PAYMENT",
			product:           "Current",
			startedDateText:   "2026-08-01 10:00:00",
			completedDateText: "2026-08-01 10:01:00",
			rawDescription:    "Groceries",
			amountText:        "-25.00",
			feeText:           "0.00",
			currencyText:      "EUR",
			stateText:         "COMPLETED",
			balanceText:       "",
		},
	}

	if !slices.Equal(got, want) {
		t.Errorf(
			"readRevolutRows() = %+v, want %+v",
			got,
			want,
		)
	}
}

func TestReadRevolutRowsHeaderOnly(t *testing.T) {
	input := strings.NewReader(
		"Type,Product,Started Date,Completed Date,Description,Amount,Fee,Currency,State\n",
	)

	got, err := readRevolutRows(input)
	if err != nil {
		t.Fatalf("readRevolutRows() returned an unexpected error: %v", err)
	}

	if len(got) != 0 {
		t.Errorf("readRevolutRows() returned %d rows, want 0", len(got))
	}
}

func TestReadRevolutRowsErrors(t *testing.T) {
	tests := []struct {
		name              string
		input             string
		wantErrorContains string
	}{
		{
			name: "missing required header",
			input: `Type,Product,Started Date,Completed Date,Description,Fee,Currency,State
CARD_PAYMENT,Current,2026-08-01 10:00:00,2026-08-01 10:01:00,Groceries,0.00,EUR,COMPLETED
`,
			wantErrorContains: `missing required column "Amount"`,
		},
		{
			name: "required field absent from record",
			input: `Type,Product,Started Date,Completed Date,Description,Amount,Fee,Currency,State
CARD_PAYMENT,Current,2026-08-01 10:00:00,2026-08-01 10:01:00,Groceries,-25.00,0.00,EUR
`,
			wantErrorContains: `required column "State" is absent`,
		},
		{
			name: "malformed quoted record",
			input: `Type,Product,Started Date,Completed Date,Description,Amount,Fee,Currency,State
CARD_PAYMENT,Current,2026-08-01 10:00:00,2026-08-01 10:01:00,"unterminated
`,
			wantErrorContains: "read Revolut CSV record 2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := readRevolutRows(strings.NewReader(tt.input))
			if err == nil {
				t.Fatal("readRevolutRows() error = nil, want non-nil")
			}

			if got != nil {
				t.Errorf(
					"readRevolutRows() returned partial rows %+v with an error, want nil",
					got,
				)
			}

			if !strings.Contains(err.Error(), tt.wantErrorContains) {
				t.Errorf(
					"readRevolutRows() error = %q, want it to contain %q",
					err,
					tt.wantErrorContains,
				)
			}
		})
	}
}

func TestImportRevolutTransactions(t *testing.T) {
	input := strings.NewReader(
		`Type,Product,Started Date,Completed Date,Description,Amount,Fee,Currency,State
Card Payment,Current,2026-08-01 10:00:00,,"Shop, Vilnius",-12.34,1.00,EUR,PENDING
`,
	)

	got, err := importRevolutTransactions(input)
	if err != nil {
		t.Fatalf(
			"importRevolutTransactions() returned an unexpected error: %v",
			err,
		)
	}

	want := []importedTransaction{
		{
			source:           sourceRevolut,
			accountText:      "Current",
			occurredAtText:   "2026-08-01 10:00:00",
			completedAtText:  "",
			amountText:       "-12.34",
			feeText:          "1.00",
			currencyText:     "EUR",
			directionText:    "",
			rawDescription:   "Shop, Vilnius",
			counterpartyText: "",
			externalID:       "",
			typeText:         "Card Payment",
			stateText:        "PENDING",
		},
	}

	if !slices.Equal(got, want) {
		t.Errorf(
			"importRevolutTransactions() = %+v, want %+v",
			got,
			want,
		)
	}
}
