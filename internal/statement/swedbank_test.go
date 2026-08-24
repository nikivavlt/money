package statement

import (
	"maps"
	"slices"
	"strings"
	"testing"
)

func TestSwedbankColumnIndexes(t *testing.T) {
	tests := []struct {
		name   string
		header []string
		want   map[string]int
	}{
		{
			name: "actual Swedbank header",
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
			want: map[string]int{
				"Account No":        0,
				"Date":              2,
				"Beneficiary":       3,
				"Details":           4,
				"Amount":            5,
				"Currency":          6,
				"D/K":               7,
				"Record ID":         8,
				"Code":              9,
				"Reference No":      10,
				"Doc. No":           11,
				"Code in payer IS":  12,
				"Client code":       13,
				"Originator":        14,
				"Beneficiary party": 15,
			},
		},
		{
			name: "required columns reordered",
			header: []string{
				"Currency",
				"Code",
				"Details",
				"Date",
				"Amount",
				"Account No",
				"Record ID",
				"Beneficiary",
				"D/K",
			},
			want: map[string]int{
				"Currency":    0,
				"Code":        1,
				"Details":     2,
				"Date":        3,
				"Amount":      4,
				"Account No":  5,
				"Record ID":   6,
				"Beneficiary": 7,
				"D/K":         8,
			},
		},
		{
			name: "optional trailing columns absent",
			header: []string{
				"Account No",
				"Date",
				"Beneficiary",
				"Details",
				"Amount",
				"Currency",
				"D/K",
				"Record ID",
				"Code",
			},
			want: map[string]int{
				"Account No":  0,
				"Date":        1,
				"Beneficiary": 2,
				"Details":     3,
				"Amount":      4,
				"Currency":    5,
				"D/K":         6,
				"Record ID":   7,
				"Code":        8,
			},
		},
		{
			name: "multiple unnamed columns ignored",
			header: []string{
				"",
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
				"",
			},
			want: map[string]int{
				"Account No":  1,
				"Date":        3,
				"Beneficiary": 4,
				"Details":     5,
				"Amount":      6,
				"Currency":    7,
				"D/K":         8,
				"Record ID":   9,
				"Code":        10,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := swedbankColumnIndexes(tt.header)
			if err != nil {
				t.Fatalf(
					"swedbankColumnIndexes() returned an unexpected error: %v",
					err,
				)
			}

			if !maps.Equal(got, tt.want) {
				t.Errorf(
					"swedbankColumnIndexes() = %v, want %v",
					got,
					tt.want,
				)
			}
		})
	}
}

func TestSwedbankColumnIndexesErrors(t *testing.T) {
	tests := []struct {
		name              string
		header            []string
		wantErrorContains string
	}{
		{
			name: "missing debit credit column",
			header: []string{
				"Account No",
				"Date",
				"Beneficiary",
				"Details",
				"Amount",
				"Currency",
				"Record ID",
				"Code",
			},
			wantErrorContains: `missing required column "D/K"`,
		},
		{
			name: "duplicate amount column",
			header: []string{
				"Account No",
				"Date",
				"Beneficiary",
				"Details",
				"Amount",
				"Currency",
				"D/K",
				"Record ID",
				"Code",
				"Amount",
			},
			wantErrorContains: `duplicate column "Amount"`,
		},
		{
			name:              "nil header",
			header:            nil,
			wantErrorContains: `missing required column "Account No"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := swedbankColumnIndexes(tt.header)
			if err == nil {
				t.Fatal("swedbankColumnIndexes() error = nil, want non-nil")
			}

			if got != nil {
				t.Errorf(
					"swedbankColumnIndexes() returned %v with an error, want nil map",
					got,
				)
			}

			if !strings.Contains(err.Error(), tt.wantErrorContains) {
				t.Errorf(
					"swedbankColumnIndexes() error = %q, want it to contain %q",
					err,
					tt.wantErrorContains,
				)
			}
		})
	}
}

func TestSwedbankRowFromRecord(t *testing.T) {
	header := []string{
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
	}

	indexes, err := swedbankColumnIndexes(header)
	if err != nil {
		t.Fatalf(
			"swedbankColumnIndexes() returned an unexpected error: %v",
			err,
		)
	}

	record := []string{
		"LT00-TEST-ACCOUNT",
		"Current account",
		"2026-08-01",
		"Example Shop",
		"Card payment, Vilnius",
		"25.00",
		"EUR",
		"D",
		"record-1",
		"CARD",
		"reference-1",
		"document-1",
		"payer-code-1",
		"client-code-1",
		"Example Originator",
		"Example Beneficiary",
		"",
	}

	got, err := swedbankRowFromRecord(record, indexes)
	if err != nil {
		t.Fatalf(
			"swedbankRowFromRecord() returned an unexpected error: %v",
			err,
		)
	}

	want := swedbankRow{
		accountNumberText:    "LT00-TEST-ACCOUNT",
		dateText:             "2026-08-01",
		beneficiaryText:      "Example Shop",
		detailsText:          "Card payment, Vilnius",
		amountText:           "25.00",
		currencyText:         "EUR",
		directionText:        "D",
		externalID:           "record-1",
		codeText:             "CARD",
		referenceNumberText:  "reference-1",
		documentNumberText:   "document-1",
		payerCodeText:        "payer-code-1",
		clientCodeText:       "client-code-1",
		originatorText:       "Example Originator",
		beneficiaryPartyText: "Example Beneficiary",
	}

	if got != want {
		t.Errorf(
			"swedbankRowFromRecord() = %+v, want %+v",
			got,
			want,
		)
	}
}

func TestReadSwedbankRowsWithVariableWidths(t *testing.T) {
	input := strings.NewReader(
		`Account No,,Date,Beneficiary,Details,Amount,Currency,D/K,Record ID,Code,Reference No,Doc. No,Code in payer IS,Client code,Originator,Beneficiary party,
LT00-TEST-ACCOUNT,Current account,2026-08-01,Example Shop,"Card payment, Vilnius",25.00,EUR,D,record-1,CARD,reference-1,document-1,payer-code-1,client-code-1
LT00-TEST-ACCOUNT,Current account,2026-08-02,Example Employer,Salary,1000.00,EUR,K,record-2,TRANSFER,reference-2,document-2,payer-code-2
`,
	)

	got, err := readSwedbankRows(input)
	if err != nil {
		t.Fatalf(
			"readSwedbankRows() returned an unexpected error: %v",
			err,
		)
	}

	want := []swedbankRow{
		{
			accountNumberText:    "LT00-TEST-ACCOUNT",
			dateText:             "2026-08-01",
			beneficiaryText:      "Example Shop",
			detailsText:          "Card payment, Vilnius",
			amountText:           "25.00",
			currencyText:         "EUR",
			directionText:        "D",
			externalID:           "record-1",
			codeText:             "CARD",
			referenceNumberText:  "reference-1",
			documentNumberText:   "document-1",
			payerCodeText:        "payer-code-1",
			clientCodeText:       "client-code-1",
			originatorText:       "",
			beneficiaryPartyText: "",
		},
		{
			accountNumberText:    "LT00-TEST-ACCOUNT",
			dateText:             "2026-08-02",
			beneficiaryText:      "Example Employer",
			detailsText:          "Salary",
			amountText:           "1000.00",
			currencyText:         "EUR",
			directionText:        "K",
			externalID:           "record-2",
			codeText:             "TRANSFER",
			referenceNumberText:  "reference-2",
			documentNumberText:   "document-2",
			payerCodeText:        "payer-code-2",
			clientCodeText:       "",
			originatorText:       "",
			beneficiaryPartyText: "",
		},
	}

	if !slices.Equal(got, want) {
		t.Errorf(
			"readSwedbankRows() = %+v, want %+v",
			got,
			want,
		)
	}
}

func TestReadSwedbankRowsRequiredColumnsOnly(t *testing.T) {
	input := strings.NewReader(
		`Account No,Date,Beneficiary,Details,Amount,Currency,D/K,Record ID,Code
LT00-TEST-ACCOUNT,2026-08-01,Example Shop,Card payment,25.00,EUR,D,record-1,CARD
`,
	)

	got, err := readSwedbankRows(input)
	if err != nil {
		t.Fatalf(
			"readSwedbankRows() returned an unexpected error: %v",
			err,
		)
	}

	want := []swedbankRow{
		{
			accountNumberText: "LT00-TEST-ACCOUNT",
			dateText:          "2026-08-01",
			beneficiaryText:   "Example Shop",
			detailsText:       "Card payment",
			amountText:        "25.00",
			currencyText:      "EUR",
			directionText:     "D",
			externalID:        "record-1",
			codeText:          "CARD",
		},
	}

	if !slices.Equal(got, want) {
		t.Errorf(
			"readSwedbankRows() = %+v, want %+v",
			got,
			want,
		)
	}
}

func TestReadSwedbankRowsHeaderOnly(t *testing.T) {
	input := strings.NewReader(
		"Account No,Date,Beneficiary,Details,Amount,Currency,D/K,Record ID,Code\n",
	)

	got, err := readSwedbankRows(input)
	if err != nil {
		t.Fatalf(
			"readSwedbankRows() returned an unexpected error: %v",
			err,
		)
	}

	if len(got) != 0 {
		t.Errorf(
			"readSwedbankRows() returned %d rows, want 0",
			len(got),
		)
	}
}

func TestReadSwedbankRowsErrors(t *testing.T) {
	tests := []struct {
		name              string
		input             string
		wantErrorContains string
	}{
		{
			name: "required field absent from short record",
			input: `Account No,,Date,Beneficiary,Details,Amount,Currency,D/K,Record ID,Code
LT00-TEST-ACCOUNT,Current account,2026-08-01,Example Shop,Card payment,25.00,EUR,D,record-1
`,
			wantErrorContains: `required column "Code" is absent`,
		},
		{
			name: "missing required header",
			input: `Account No,Date,Beneficiary,Details,Amount,Currency,Record ID,Code
LT00-TEST-ACCOUNT,2026-08-01,Example Shop,Card payment,25.00,EUR,record-1,CARD
`,
			wantErrorContains: `missing required column "D/K"`,
		},
		{
			name: "malformed record discards previously parsed rows",
			input: `Account No,Date,Beneficiary,Details,Amount,Currency,D/K,Record ID,Code
LT00-TEST-ACCOUNT,2026-08-01,Example Shop,Card payment,25.00,EUR,D,record-1,CARD
LT00-TEST-ACCOUNT,2026-08-02,Example Shop,"unterminated
`,
			wantErrorContains: "read Swedbank CSV record 3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := readSwedbankRows(strings.NewReader(tt.input))
			if err == nil {
				t.Fatal("readSwedbankRows() error = nil, want non-nil")
			}

			if got != nil {
				t.Errorf(
					"readSwedbankRows() returned partial rows %+v with an error, want nil",
					got,
				)
			}

			if !strings.Contains(err.Error(), tt.wantErrorContains) {
				t.Errorf(
					"readSwedbankRows() error = %q, want it to contain %q",
					err,
					tt.wantErrorContains,
				)
			}
		})
	}
}

func TestImportSwedbankTransactions(t *testing.T) {
	input := strings.NewReader(
		`Account No,Date,Beneficiary,Details,Amount,Currency,D/K,Record ID,Code
LT00-TEST-ACCOUNT,2026-08-01,Example Shop,"Card payment, Vilnius",25.00,EUR,D,,K
`,
	)

	got, err := importSwedbankTransactions(input)
	if err != nil {
		t.Fatalf(
			"importSwedbankTransactions() returned an unexpected error: %v",
			err,
		)
	}

	want := []importedTransaction{
		{
			source:           Swedbank,
			accountText:      "LT00-TEST-ACCOUNT",
			occurredAtText:   "2026-08-01",
			completedAtText:  "",
			amountText:       "25.00",
			feeText:          "",
			currencyText:     "EUR",
			directionText:    "D",
			rawDescription:   "Card payment, Vilnius",
			counterpartyText: "Example Shop",
			externalID:       "",
			typeText:         "K",
			stateText:        "",
		},
	}

	if !slices.Equal(got, want) {
		t.Errorf(
			"importSwedbankTransactions() = %+v, want %+v",
			got,
			want,
		)
	}
}
