package statement

import (
	"errors"
	"money/internal/finance"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestNormalizeImportedMoney(t *testing.T) {
	tests := []struct {
		name  string
		input importedTransaction
		want  finance.Money
	}{
		{
			name: "negative Revolut EUR amount",
			input: importedTransaction{
				source:       sourceRevolut,
				amountText:   "-12.34",
				feeText:      "0.00",
				currencyText: "EUR",
			},
			want: finance.Money{
				Amount:   -1_234,
				Currency: finance.EUR,
			},
		},
		{
			name: "positive Revolut EUR amount",
			input: importedTransaction{
				source:       sourceRevolut,
				amountText:   "1000.00",
				feeText:      "0.00",
				currencyText: "EUR",
			},
			want: finance.Money{
				Amount:   100_000,
				Currency: finance.EUR,
			},
		},
		{
			name: "Swedbank debit becomes negative",
			input: importedTransaction{
				source:        sourceSwedbank,
				amountText:    "25.00",
				currencyText:  "EUR",
				directionText: "D",
			},
			want: finance.Money{
				Amount:   -2_500,
				Currency: finance.EUR,
			},
		},
		{
			name: "Swedbank credit remains positive",
			input: importedTransaction{
				source:        sourceSwedbank,
				amountText:    "1000.00",
				currencyText:  "EUR",
				directionText: "K",
			},
			want: finance.Money{
				Amount:   100_000,
				Currency: finance.EUR,
			},
		},
		{
			name: "Swedbank debit zero remains zero",
			input: importedTransaction{
				source:        sourceSwedbank,
				amountText:    "0.00",
				currencyText:  "EUR",
				directionText: "D",
			},
			want: finance.Money{
				Amount:   0,
				Currency: finance.EUR,
			},
		},
		{
			name: "Swedbank USD debit",
			input: importedTransaction{
				source:        sourceSwedbank,
				amountText:    "12.34",
				currencyText:  "USD",
				directionText: "D",
			},
			want: finance.Money{
				Amount:   -1_234,
				Currency: finance.USD,
			},
		},
		{
			name: "surrounding whitespace",
			input: importedTransaction{
				source:        sourceSwedbank,
				amountText:    " 12.34 ",
				currencyText:  " EUR ",
				directionText: " D ",
			},
			want: finance.Money{
				Amount:   -1_234,
				Currency: finance.EUR,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeImportedMoney(tt.input)
			if err != nil {
				t.Fatalf(
					"normalizeImportedMoney() returned an unexpected error: %v",
					err,
				)
			}

			if got != tt.want {
				t.Errorf(
					"normalizeImportedMoney() = %+v, want %+v",
					got,
					tt.want,
				)
			}
		})
	}
}

func TestNormalizeImportedMoneyErrors(t *testing.T) {
	tests := []struct {
		name              string
		input             importedTransaction
		wantError         error
		wantErrorContains string
	}{
		{
			name: "unsupported source",
			input: importedTransaction{
				source:       statementSource("unknown"),
				amountText:   "12.34",
				currencyText: "EUR",
			},
			wantErrorContains: "unsupported statement source",
		},
		{
			name: "unsupported Revolut currency",
			input: importedTransaction{
				source:       sourceRevolut,
				amountText:   "12.34",
				feeText:      "0.00",
				currencyText: "GBP",
			},
			wantError:         finance.ErrUnsupportedCurrency,
			wantErrorContains: "unsupported currency",
		},
		{
			name: "lowercase Revolut currency",
			input: importedTransaction{
				source:       sourceRevolut,
				amountText:   "12.34",
				feeText:      "0.00",
				currencyText: "eur",
			},
			wantError:         finance.ErrUnsupportedCurrency,
			wantErrorContains: "unsupported currency",
		},
		{
			name: "unsupported Swedbank currency",
			input: importedTransaction{
				source:        sourceSwedbank,
				amountText:    "12.34",
				currencyText:  "GBP",
				directionText: "D",
			},
			wantError:         finance.ErrUnsupportedCurrency,
			wantErrorContains: "unsupported currency",
		},
		{
			name: "nonzero Revolut fee",
			input: importedTransaction{
				source:       sourceRevolut,
				amountText:   "-12.34",
				feeText:      "0.50",
				currencyText: "EUR",
			},
			wantErrorContains: "nonzero fee is not supported",
		},
		{
			name: "invalid Revolut fee",
			input: importedTransaction{
				source:       sourceRevolut,
				amountText:   "-12.34",
				feeText:      "not-money",
				currencyText: "EUR",
			},
			wantErrorContains: "normalize Revolut fee",
		},
		{
			name: "negative Swedbank raw amount",
			input: importedTransaction{
				source:        sourceSwedbank,
				amountText:    "-25.00",
				currencyText:  "EUR",
				directionText: "D",
			},
			wantErrorContains: "amount must be non-negative",
		},
		{
			name: "unknown Swedbank direction",
			input: importedTransaction{
				source:        sourceSwedbank,
				amountText:    "25.00",
				currencyText:  "EUR",
				directionText: "X",
			},
			wantErrorContains: `unsupported D/K value "X"`,
		},
		{
			name: "empty Swedbank direction",
			input: importedTransaction{
				source:        sourceSwedbank,
				amountText:    "25.00",
				currencyText:  "EUR",
				directionText: "",
			},
			wantErrorContains: `unsupported D/K value ""`,
		},
		{
			name: "Revolut amount overflow",
			input: importedTransaction{
				source:       sourceRevolut,
				amountText:   "92233720368547758.08",
				feeText:      "0.00",
				currencyText: "EUR",
			},
			wantError:         strconv.ErrRange,
			wantErrorContains: "normalize Revolut amount",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeImportedMoney(tt.input)
			if err == nil {
				t.Fatal(
					"normalizeImportedMoney() error = nil, want non-nil",
				)
			}

			if got != (finance.Money{}) {
				t.Errorf(
					"normalizeImportedMoney() returned %+v with an error, want zero Money",
					got,
				)
			}

			if tt.wantError != nil &&
				!errors.Is(err, tt.wantError) {
				t.Errorf(
					"normalizeImportedMoney() error = %v, want it to match %v",
					err,
					tt.wantError,
				)
			}

			if !strings.Contains(
				err.Error(),
				tt.wantErrorContains,
			) {
				t.Errorf(
					"normalizeImportedMoney() error = %q, want it to contain %q",
					err,
					tt.wantErrorContains,
				)
			}
		})
	}
}

func TestNormalizeImportedDate(t *testing.T) {
	location, err := time.LoadLocation("Europe/Vilnius")
	if err != nil {
		t.Fatalf(
			"time.LoadLocation() returned an unexpected error: %v",
			err,
		)
	}

	tests := []struct {
		name  string
		input importedTransaction
		want  time.Time
	}{
		{
			name: "Revolut uses completed date",
			input: importedTransaction{
				source:          sourceRevolut,
				occurredAtText:  "2026-08-01 09:30:00",
				completedAtText: "2026-08-02 10:15:30",
				stateText:       "COMPLETED",
			},
			want: time.Date(
				2026,
				time.August,
				2,
				10,
				15,
				30,
				0,
				location,
			),
		},
		{
			name: "Revolut trims state and date",
			input: importedTransaction{
				source:          sourceRevolut,
				occurredAtText:  "2026-08-01 09:30:00",
				completedAtText: " 2026-08-02 10:15:30 ",
				stateText:       " COMPLETED ",
			},
			want: time.Date(
				2026,
				time.August,
				2,
				10,
				15,
				30,
				0,
				location,
			),
		},
		{
			name: "Swedbank date becomes local midnight",
			input: importedTransaction{
				source:         sourceSwedbank,
				occurredAtText: "2026-08-03",
			},
			want: time.Date(
				2026,
				time.August,
				3,
				0,
				0,
				0,
				0,
				location,
			),
		},
		{
			name: "Swedbank trims date",
			input: importedTransaction{
				source:         sourceSwedbank,
				occurredAtText: " 2026-08-04 ",
			},
			want: time.Date(
				2026,
				time.August,
				4,
				0,
				0,
				0,
				0,
				location,
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeImportedDate(
				tt.input,
				location,
			)
			if err != nil {
				t.Fatalf(
					"normalizeImportedDate() returned an unexpected error: %v",
					err,
				)
			}

			if !got.Equal(tt.want) {
				t.Errorf(
					"normalizeImportedDate() = %v, want %v",
					got,
					tt.want,
				)
			}

			if got.Location() != location {
				t.Errorf(
					"normalizeImportedDate() location = %q, want %q",
					got.Location(),
					location,
				)
			}
		})
	}
}

func TestNormalizeImportedDateErrors(t *testing.T) {
	location, err := time.LoadLocation("Europe/Vilnius")
	if err != nil {
		t.Fatalf(
			"time.LoadLocation() returned an unexpected error: %v",
			err,
		)
	}

	tests := []struct {
		name              string
		input             importedTransaction
		location          *time.Location
		wantError         error
		wantErrorContains string
	}{
		{
			name: "nil location",
			input: importedTransaction{
				source:          sourceRevolut,
				completedAtText: "2026-08-01 10:00:00",
				stateText:       "COMPLETED",
			},
			location:          nil,
			wantErrorContains: "nil location",
		},
		{
			name: "pending Revolut transaction",
			input: importedTransaction{
				source:          sourceRevolut,
				completedAtText: "",
				stateText:       "PENDING",
			},
			location:          location,
			wantError:         ErrPendingTransaction,
			wantErrorContains: "transaction is pending",
		},
		{
			name: "pending state with surrounding whitespace",
			input: importedTransaction{
				source:    sourceRevolut,
				stateText: " PENDING ",
			},
			location:          location,
			wantError:         ErrPendingTransaction,
			wantErrorContains: "transaction is pending",
		},
		{
			name: "unsupported Revolut state",
			input: importedTransaction{
				source:          sourceRevolut,
				completedAtText: "2026-08-01 10:00:00",
				stateText:       "REVERTED",
			},
			location:          location,
			wantErrorContains: `unsupported state "REVERTED"`,
		},
		{
			name: "empty Revolut state",
			input: importedTransaction{
				source:          sourceRevolut,
				completedAtText: "2026-08-01 10:00:00",
				stateText:       "",
			},
			location:          location,
			wantErrorContains: `unsupported state ""`,
		},
		{
			name: "completed Revolut transaction without completed date",
			input: importedTransaction{
				source:          sourceRevolut,
				completedAtText: "",
				stateText:       "COMPLETED",
			},
			location:          location,
			wantErrorContains: "completed transaction has empty completed date",
		},
		{
			name: "invalid Revolut completed date",
			input: importedTransaction{
				source:          sourceRevolut,
				completedAtText: "2026-02-30 10:00:00",
				stateText:       "COMPLETED",
			},
			location:          location,
			wantErrorContains: "normalize Revolut completed date",
		},
		{
			name: "invalid Swedbank date",
			input: importedTransaction{
				source:         sourceSwedbank,
				occurredAtText: "2026-02-30",
			},
			location:          location,
			wantErrorContains: "normalize Swedbank date",
		},
		{
			name: "empty Swedbank date",
			input: importedTransaction{
				source:         sourceSwedbank,
				occurredAtText: "",
			},
			location:          location,
			wantErrorContains: "empty input",
		},
		{
			name: "unsupported source",
			input: importedTransaction{
				source: statementSource("unknown"),
			},
			location:          location,
			wantErrorContains: "unsupported statement source",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeImportedDate(
				tt.input,
				tt.location,
			)
			if err == nil {
				t.Fatal(
					"normalizeImportedDate() error = nil, want non-nil",
				)
			}

			if !got.IsZero() {
				t.Errorf(
					"normalizeImportedDate() returned %v with an error, want zero time",
					got,
				)
			}

			if tt.wantError != nil &&
				!errors.Is(err, tt.wantError) {
				t.Errorf(
					"normalizeImportedDate() error = %v, want it to match %v",
					err,
					tt.wantError,
				)
			}

			if !strings.Contains(
				err.Error(),
				tt.wantErrorContains,
			) {
				t.Errorf(
					"normalizeImportedDate() error = %q, want it to contain %q",
					err,
					tt.wantErrorContains,
				)
			}
		})
	}
}

func TestNormalizeImportedTransaction(t *testing.T) {
	location, err := time.LoadLocation("Europe/Vilnius")
	if err != nil {
		t.Fatalf(
			"time.LoadLocation() returned an unexpected error: %v",
			err,
		)
	}

	tests := []struct {
		name  string
		input importedTransaction
		want  Transaction
	}{
		{
			name: "completed Revolut transaction",
			input: importedTransaction{
				source:           sourceRevolut,
				accountText:      "Current",
				occurredAtText:   "2026-08-01 09:30:00",
				completedAtText:  "2026-08-02 10:15:30",
				amountText:       "-12.34",
				feeText:          "0.00",
				currencyText:     "EUR",
				rawDescription:   "  Shop, Vilnius  ",
				counterpartyText: "",
				typeText:         "Card Payment",
				stateText:        "COMPLETED",
			},
			want: Transaction{
				Date: time.Date(
					2026,
					time.August,
					2,
					10,
					15,
					30,
					0,
					location,
				),
				Amount: finance.Money{
					Amount:   -1_234,
					Currency: finance.EUR,
				},
				Description:  "  Shop, Vilnius  ",
				Counterparty: "",
			},
		},
		{
			name: "Swedbank debit transaction",
			input: importedTransaction{
				source:           sourceSwedbank,
				accountText:      "LT00-TEST-ACCOUNT",
				occurredAtText:   "2026-08-03",
				amountText:       "25.00",
				currencyText:     "EUR",
				directionText:    "D",
				rawDescription:   "Card payment, Vilnius",
				counterpartyText: "Example Shop",
				externalID:       "record-1",
				typeText:         "K",
			},
			want: Transaction{
				Date: time.Date(
					2026,
					time.August,
					3,
					0,
					0,
					0,
					0,
					location,
				),
				Amount: finance.Money{
					Amount:   -2_500,
					Currency: finance.EUR,
				},
				Description:  "Card payment, Vilnius",
				Counterparty: "Example Shop",
			},
		},
		{
			name: "Swedbank credit transaction",
			input: importedTransaction{
				source:           sourceSwedbank,
				occurredAtText:   "2026-08-04",
				amountText:       "1000.00",
				currencyText:     "EUR",
				directionText:    "K",
				rawDescription:   "Salary",
				counterpartyText: "Example Employer",
			},
			want: Transaction{
				Date: time.Date(
					2026,
					time.August,
					4,
					0,
					0,
					0,
					0,
					location,
				),
				Amount: finance.Money{
					Amount:   100_000,
					Currency: finance.EUR,
				},
				Description:  "Salary",
				Counterparty: "Example Employer",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeImportedTransaction(
				tt.input,
				location,
			)
			if err != nil {
				t.Fatalf(
					"normalizeImportedTransaction() returned an unexpected error: %v",
					err,
				)
			}

			if !got.Date.Equal(tt.want.Date) {
				t.Errorf(
					"normalized date = %v, want %v",
					got.Date,
					tt.want.Date,
				)
			}

			if got.Amount != tt.want.Amount {
				t.Errorf(
					"normalized amount = %+v, want %+v",
					got.Amount,
					tt.want.Amount,
				)
			}

			if got.Description != tt.want.Description {
				t.Errorf(
					"normalized description = %q, want %q",
					got.Description,
					tt.want.Description,
				)
			}

			if got.Counterparty != tt.want.Counterparty {
				t.Errorf(
					"normalized counterparty = %q, want %q",
					got.Counterparty,
					tt.want.Counterparty,
				)
			}
		})
	}
}

func TestNormalizeImportedTransactionErrors(t *testing.T) {
	location, err := time.LoadLocation("Europe/Vilnius")
	if err != nil {
		t.Fatalf(
			"time.LoadLocation() returned an unexpected error: %v",
			err,
		)
	}

	tests := []struct {
		name              string
		input             importedTransaction
		location          *time.Location
		wantError         error
		wantErrorContains string
	}{
		{
			name: "pending transaction is classified before invalid money",
			input: importedTransaction{
				source:          sourceRevolut,
				stateText:       "PENDING",
				completedAtText: "",
				amountText:      "not-money",
				feeText:         "also-invalid",
				currencyText:    "GBP",
			},
			location:          location,
			wantError:         ErrPendingTransaction,
			wantErrorContains: "transaction is pending",
		},
		{
			name: "unsupported currency remains inspectable",
			input: importedTransaction{
				source:          sourceRevolut,
				stateText:       "COMPLETED",
				completedAtText: "2026-08-01 10:00:00",
				amountText:      "12.34",
				feeText:         "0.00",
				currencyText:    "GBP",
			},
			location:          location,
			wantError:         finance.ErrUnsupportedCurrency,
			wantErrorContains: "unsupported currency",
		},
		{
			name: "nonzero Revolut fee",
			input: importedTransaction{
				source:          sourceRevolut,
				stateText:       "COMPLETED",
				completedAtText: "2026-08-01 10:00:00",
				amountText:      "-12.34",
				feeText:         "0.50",
				currencyText:    "EUR",
			},
			location:          location,
			wantErrorContains: "nonzero fee is not supported",
		},
		{
			name: "nil location",
			input: importedTransaction{
				source:          sourceRevolut,
				stateText:       "COMPLETED",
				completedAtText: "2026-08-01 10:00:00",
				amountText:      "-12.34",
				feeText:         "0.00",
				currencyText:    "EUR",
			},
			location:          nil,
			wantErrorContains: "nil location",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeImportedTransaction(
				tt.input,
				tt.location,
			)
			if err == nil {
				t.Fatal(
					"normalizeImportedTransaction() error = nil, want non-nil",
				)
			}

			if got != (Transaction{}) {
				t.Errorf(
					"normalizeImportedTransaction() returned %+v with an error, want zero Transaction",
					got,
				)
			}

			if tt.wantError != nil &&
				!errors.Is(err, tt.wantError) {
				t.Errorf(
					"normalizeImportedTransaction() error = %v, want it to match %v",
					err,
					tt.wantError,
				)
			}

			if !strings.Contains(
				err.Error(),
				tt.wantErrorContains,
			) {
				t.Errorf(
					"normalizeImportedTransaction() error = %q, want it to contain %q",
					err,
					tt.wantErrorContains,
				)
			}
		})
	}
}

func TestNormalizeImportedTransactions(t *testing.T) {
	location := time.FixedZone("test", 2*60*60)

	imported := []importedTransaction{
		{
			source:           sourceSwedbank,
			occurredAtText:   "2026-08-04",
			amountText:       "25.50",
			currencyText:     "EUR",
			directionText:    "D",
			rawDescription:   "Card purchase",
			counterpartyText: "MAXIMA",
		},
		{
			source:          sourceRevolut,
			completedAtText: "2026-08-05 14:30:00",
			amountText:      "100.00",
			feeText:         "0",
			currencyText:    "EUR",
			rawDescription:  "Salary",
			stateText:       "COMPLETED",
		},
	}

	got, err := normalizeImportedTransactions(imported, location)
	if err != nil {
		t.Fatalf(
			"normalizeImportedTransactions() returned an unexpected error: %v",
			err,
		)
	}

	want := []Transaction{
		{
			Date: time.Date(
				2026, time.August, 4,
				0, 0, 0, 0,
				location,
			),
			Amount: finance.Money{
				Amount:   -2_550,
				Currency: finance.EUR,
			},
			Description:  "Card purchase",
			Counterparty: "MAXIMA",
		},
		{
			Date: time.Date(
				2026, time.August, 5,
				14, 30, 0, 0,
				location,
			),
			Amount: finance.Money{
				Amount:   10_000,
				Currency: finance.EUR,
			},
			Description: "Salary",
		},
	}

	if !slices.Equal(got, want) {
		t.Errorf(
			"normalizeImportedTransactions() = %+v, want %+v",
			got,
			want,
		)
	}
}

func TestNormalizeImportedTransactionsEmptyInput(t *testing.T) {
	tests := []struct {
		name  string
		input []importedTransaction
	}{
		{
			name:  "nil input",
			input: nil,
		},
		{
			name:  "empty input",
			input: []importedTransaction{},
		},
	}

	location := time.FixedZone("test", 2*60*60)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeImportedTransactions(tt.input, location)
			if err != nil {
				t.Fatalf(
					"normalizeImportedTransactions() returned an unexpected error: %v",
					err,
				)
			}

			if len(got) != 0 {
				t.Errorf(
					"normalizeImportedTransactions() returned %d transactions, want 0",
					len(got),
				)
			}
		})
	}
}

func TestNormalizeImportedTransactionsIdentifiesFailingTransaction(t *testing.T) {
	location := time.FixedZone("test", 2*60*60)

	imported := []importedTransaction{
		{
			source:           sourceSwedbank,
			occurredAtText:   "2026-08-04",
			amountText:       "10.00",
			currencyText:     "EUR",
			directionText:    "D",
			rawDescription:   "Valid transaction",
			counterpartyText: "MAXIMA",
		},
		{
			source:           sourceSwedbank,
			occurredAtText:   "2026-08-05",
			amountText:       "invalid amount",
			currencyText:     "EUR",
			directionText:    "D",
			rawDescription:   "Invalid transaction",
			counterpartyText: "UNKNOWN",
		},
	}

	got, err := normalizeImportedTransactions(imported, location)
	if err == nil {
		t.Fatal(
			"normalizeImportedTransactions() error = nil, want non-nil",
		)
	}

	if !strings.Contains(err.Error(), "transaction 2") {
		t.Errorf(
			"normalizeImportedTransactions() error = %q, want transaction number 2",
			err,
		)
	}

	if got != nil {
		t.Errorf(
			"normalizeImportedTransactions() returned partial results: %+v",
			got,
		)
	}
}

func TestNormalizeImportedTransactionsPreservesPendingError(t *testing.T) {
	location := time.FixedZone("test", 2*60*60)

	imported := []importedTransaction{
		{
			source:          sourceRevolut,
			completedAtText: "",
			amountText:      "-10.00",
			feeText:         "0",
			currencyText:    "EUR",
			rawDescription:  "Pending card payment",
			stateText:       "PENDING",
		},
	}

	got, err := normalizeImportedTransactions(imported, location)
	if err == nil {
		t.Fatal(
			"normalizeImportedTransactions() error = nil, want non-nil",
		)
	}

	if !errors.Is(err, ErrPendingTransaction) {
		t.Errorf(
			"normalizeImportedTransactions() error = %v, want it to match ErrPendingTransaction",
			err,
		)
	}

	if got != nil {
		t.Errorf(
			"normalizeImportedTransactions() returned transactions with an error: %+v",
			got,
		)
	}
}

func TestNormalizeImportedTransactionsRejectsNilLocation(t *testing.T) {
	imported := []importedTransaction{
		{
			source:           sourceSwedbank,
			occurredAtText:   "2026-08-04",
			amountText:       "10.00",
			currencyText:     "EUR",
			directionText:    "D",
			rawDescription:   "Card purchase",
			counterpartyText: "MAXIMA",
		},
	}

	got, err := normalizeImportedTransactions(imported, nil)
	if err == nil {
		t.Fatal(
			"normalizeImportedTransactions() error = nil, want non-nil",
		)
	}

	if !strings.Contains(err.Error(), "transaction 1") {
		t.Errorf(
			"normalizeImportedTransactions() error = %q, want transaction number 1",
			err,
		)
	}

	if got != nil {
		t.Errorf(
			"normalizeImportedTransactions() returned transactions with an error: %+v",
			got,
		)
	}
}
