package main

import (
	"slices"
	"testing"
)

func TestImportedTransactionIdentityUsesExternalID(t *testing.T) {
	original := importedTransaction{
		source:           sourceSwedbank,
		accountText:      "account-1",
		occurredAtText:   "2026-08-04",
		amountText:       "10.00",
		currencyText:     "EUR",
		directionText:    "D",
		rawDescription:   "Original description",
		counterpartyText: "MAXIMA",
		externalID:       "record-123",
	}

	changed := original
	changed.occurredAtText = "2026-09-20"
	changed.amountText = "999.99"
	changed.rawDescription = "Changed description"

	first := importedTransactionIdentity(original)
	second := importedTransactionIdentity(changed)

	if first.kind != identityExternalID {
		t.Errorf(
			"identity kind = %q, want %q",
			first.kind,
			identityExternalID,
		)
	}

	if first != second {
		t.Error(
			"the same source, account, and external ID produced different identities",
		)
	}
}

func TestImportedTransactionExternalIDIsScoped(t *testing.T) {
	original := importedTransaction{
		source:      sourceSwedbank,
		accountText: "account-1",
		externalID:  "record-123",
	}

	tests := []struct {
		name   string
		change func(*importedTransaction)
	}{
		{
			name: "different source",
			change: func(transaction *importedTransaction) {
				transaction.source = sourceRevolut
			},
		},
		{
			name: "different account",
			change: func(transaction *importedTransaction) {
				transaction.accountText = "account-2"
			},
		},
		{
			name: "different external ID",
			change: func(transaction *importedTransaction) {
				transaction.externalID = "record-456"
			},
		},
	}

	originalIdentity := importedTransactionIdentity(original)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			changed := original
			tt.change(&changed)

			if importedTransactionIdentity(changed) == originalIdentity {
				t.Error("scoped external-ID identity did not change")
			}
		})
	}
}

func TestImportedTransactionIdentityUsesFingerprintWithoutExternalID(
	t *testing.T,
) {
	transaction := importedTransaction{
		source:          sourceRevolut,
		accountText:     "Current",
		occurredAtText:  "2026-08-04 10:00:00",
		completedAtText: "2026-08-04 10:01:00",
		amountText:      "-12.34",
		feeText:         "0",
		currencyText:    "EUR",
		rawDescription:  "Card purchase",
		typeText:        "Card Payment",
		stateText:       "COMPLETED",
	}

	first := importedTransactionIdentity(transaction)
	second := importedTransactionIdentity(transaction)

	if first.kind != identityFingerprint {
		t.Errorf(
			"identity kind = %q, want %q",
			first.kind,
			identityFingerprint,
		)
	}

	if first != second {
		t.Error("identical transactions produced different fingerprints")
	}
}

func TestImportedTransactionFingerprintUsesMaterialFields(t *testing.T) {
	original := importedTransaction{
		source:           sourceRevolut,
		accountText:      "Current",
		occurredAtText:   "2026-08-04 10:00:00",
		completedAtText:  "2026-08-04 10:01:00",
		amountText:       "-12.34",
		feeText:          "0",
		currencyText:     "EUR",
		rawDescription:   "Card purchase",
		counterpartyText: "Merchant",
		typeText:         "Card Payment",
		stateText:        "COMPLETED",
	}

	tests := []struct {
		name   string
		change func(*importedTransaction)
	}{
		{
			name: "source",
			change: func(transaction *importedTransaction) {
				transaction.source = sourceSwedbank
			},
		},
		{
			name: "account",
			change: func(transaction *importedTransaction) {
				transaction.accountText = "Savings"
			},
		},
		{
			name: "occurred time",
			change: func(transaction *importedTransaction) {
				transaction.occurredAtText = "2026-08-04 11:00:00"
			},
		},
		{
			name: "amount",
			change: func(transaction *importedTransaction) {
				transaction.amountText = "-12.35"
			},
		},
		{
			name: "currency",
			change: func(transaction *importedTransaction) {
				transaction.currencyText = "USD"
			},
		},
		{
			name: "description",
			change: func(transaction *importedTransaction) {
				transaction.rawDescription = "Different purchase"
			},
		},
		{
			name: "counterparty",
			change: func(transaction *importedTransaction) {
				transaction.counterpartyText = "Different merchant"
			},
		},
		{
			name: "type",
			change: func(transaction *importedTransaction) {
				transaction.typeText = "Transfer"
			},
		},
		{
			name: "state",
			change: func(transaction *importedTransaction) {
				transaction.stateText = "PENDING"
			},
		},
	}

	originalIdentity := importedTransactionIdentity(original)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			changed := original
			tt.change(&changed)

			if importedTransactionIdentity(changed) == originalIdentity {
				t.Error("materially different transaction has the same fingerprint")
			}
		})
	}
}

func TestImportedTransactionIdentityTrimsSurroundingWhitespace(t *testing.T) {
	original := importedTransaction{
		source:           sourceSwedbank,
		accountText:      "account-1",
		occurredAtText:   "2026-08-04",
		amountText:       "25.50",
		currencyText:     "EUR",
		directionText:    "D",
		rawDescription:   "Card purchase",
		counterpartyText: "MAXIMA",
		typeText:         "CARD",
	}

	withWhitespace := original
	withWhitespace.accountText = " account-1 "
	withWhitespace.occurredAtText = "\t2026-08-04\n"
	withWhitespace.amountText = " 25.50 "
	withWhitespace.currencyText = " EUR "
	withWhitespace.directionText = " D "
	withWhitespace.rawDescription = " Card purchase "
	withWhitespace.counterpartyText = " MAXIMA "
	withWhitespace.typeText = " CARD "

	first := importedTransactionIdentity(original)
	second := importedTransactionIdentity(withWhitespace)

	if first != second {
		t.Error("surrounding whitespace changed the fingerprint")
	}
}

func TestWhitespaceExternalIDUsesFingerprint(t *testing.T) {
	transaction := importedTransaction{
		source:        sourceSwedbank,
		accountText:   "account-1",
		externalID:    " \t\n ",
		amountText:    "10.00",
		currencyText:  "EUR",
		directionText: "D",
	}

	identity := importedTransactionIdentity(transaction)

	if identity.kind != identityFingerprint {
		t.Errorf(
			"identity kind = %q, want %q",
			identity.kind,
			identityFingerprint,
		)
	}
}

func TestHashIdentityPartsPreservesFieldBoundaries(t *testing.T) {
	first := hashIdentityParts("A|B", "C")
	second := hashIdentityParts("A", "B|C")

	if first == second {
		t.Error("different field sequences produced the same digest")
	}
}

func TestTransactionIdentityCanBeUsedAsMapKey(t *testing.T) {
	transaction := importedTransaction{
		source:      sourceSwedbank,
		accountText: "account-1",
		externalID:  "record-123",
	}

	identity := importedTransactionIdentity(transaction)

	seen := map[transactionIdentity]struct{}{
		identity: {},
	}

	if _, exists := seen[identity]; !exists {
		t.Fatal("stored transaction identity is missing from map")
	}
}

func TestDeduplicateImportedTransactionsWithoutDuplicates(t *testing.T) {
	input := []importedTransaction{
		{
			source:      sourceSwedbank,
			accountText: "account-1",
			externalID:  "record-1",
		},
		{
			source:      sourceSwedbank,
			accountText: "account-1",
			externalID:  "record-2",
		},
	}

	result := deduplicateImportedTransactions(input)

	if !slices.Equal(result.unique, input) {
		t.Errorf(
			"unique transactions = %+v, want %+v",
			result.unique,
			input,
		)
	}

	if len(result.duplicates) != 0 {
		t.Errorf("duplicate count = %d, want 0", len(result.duplicates))
	}

	if len(result.conflicts) != 0 {
		t.Errorf("conflict count = %d, want 0", len(result.conflicts))
	}
}

func TestDeduplicateImportedTransactionsByExternalID(t *testing.T) {
	first := importedTransaction{
		source:           sourceSwedbank,
		accountText:      "account-1",
		occurredAtText:   "2026-08-04",
		amountText:       "10.00",
		currencyText:     "EUR",
		directionText:    "D",
		rawDescription:   "Card purchase",
		counterpartyText: "MAXIMA",
		externalID:       "record-123",
	}

	duplicate := first

	result := deduplicateImportedTransactions(
		[]importedTransaction{first, duplicate},
	)

	wantUnique := []importedTransaction{first}
	if !slices.Equal(result.unique, wantUnique) {
		t.Errorf(
			"unique transactions = %+v, want %+v",
			result.unique,
			wantUnique,
		)
	}

	wantDuplicates := []duplicateCandidate{
		{
			transaction:       duplicate,
			firstPosition:     1,
			duplicatePosition: 2,
			identityKind:      identityExternalID,
		},
	}

	if !slices.Equal(result.duplicates, wantDuplicates) {
		t.Errorf(
			"duplicates = %+v, want %+v",
			result.duplicates,
			wantDuplicates,
		)
	}

	if len(result.conflicts) != 0 {
		t.Errorf("conflict count = %d, want 0", len(result.conflicts))
	}
}

func TestDeduplicateImportedTransactionsByFingerprint(t *testing.T) {
	first := importedTransaction{
		source:          sourceRevolut,
		accountText:     "Current",
		occurredAtText:  "2026-08-04 10:00:00",
		completedAtText: "2026-08-04 10:01:00",
		amountText:      "-12.34",
		feeText:         "0",
		currencyText:    "EUR",
		rawDescription:  "Card purchase",
		typeText:        "Card Payment",
		stateText:       "COMPLETED",
	}

	duplicate := first
	duplicate.accountText = " Current "
	duplicate.amountText = " -12.34 "
	duplicate.rawDescription = " Card purchase "

	result := deduplicateImportedTransactions(
		[]importedTransaction{first, duplicate},
	)

	wantDuplicates := []duplicateCandidate{
		{
			transaction:       duplicate,
			firstPosition:     1,
			duplicatePosition: 2,
			identityKind:      identityFingerprint,
		},
	}

	if !slices.Equal(result.unique, []importedTransaction{first}) {
		t.Errorf("unique transactions = %+v, want first transaction", result.unique)
	}

	if !slices.Equal(result.duplicates, wantDuplicates) {
		t.Errorf(
			"duplicates = %+v, want %+v",
			result.duplicates,
			wantDuplicates,
		)
	}

	if len(result.conflicts) != 0 {
		t.Errorf("conflict count = %d, want 0", len(result.conflicts))
	}
}

func TestDeduplicateImportedTransactionsDetectsConflict(t *testing.T) {
	first := importedTransaction{
		source:           sourceSwedbank,
		accountText:      "account-1",
		occurredAtText:   "2026-08-04",
		amountText:       "10.00",
		currencyText:     "EUR",
		directionText:    "D",
		rawDescription:   "Card purchase",
		counterpartyText: "MAXIMA",
		externalID:       "record-123",
	}

	conflicting := first
	conflicting.amountText = "999.99"

	result := deduplicateImportedTransactions(
		[]importedTransaction{first, conflicting},
	)

	if !slices.Equal(result.unique, []importedTransaction{first}) {
		t.Errorf("unique transactions = %+v, want first transaction", result.unique)
	}

	if len(result.duplicates) != 0 {
		t.Errorf(
			"duplicates = %+v, want no duplicates",
			result.duplicates,
		)
	}

	wantConflicts := []duplicateConflict{
		{
			firstTransaction:       first,
			conflictingTransaction: conflicting,
			firstPosition:          1,
			conflictingPosition:    2,
			identityKind:           identityExternalID,
		},
	}

	if !slices.Equal(result.conflicts, wantConflicts) {
		t.Errorf(
			"conflicts = %+v, want %+v",
			result.conflicts,
			wantConflicts,
		)
	}
}

func TestDeduplicateImportedTransactionsMultipleCopiesReferToFirst(
	t *testing.T,
) {
	transaction := importedTransaction{
		source:          sourceRevolut,
		accountText:     "Current",
		completedAtText: "2026-08-04 10:01:00",
		amountText:      "-12.34",
		feeText:         "0",
		currencyText:    "EUR",
		rawDescription:  "Card purchase",
		typeText:        "Card Payment",
		stateText:       "COMPLETED",
	}

	result := deduplicateImportedTransactions(
		[]importedTransaction{
			transaction,
			transaction,
			transaction,
		},
	)

	wantDuplicates := []duplicateCandidate{
		{
			transaction:       transaction,
			firstPosition:     1,
			duplicatePosition: 2,
			identityKind:      identityFingerprint,
		},
		{
			transaction:       transaction,
			firstPosition:     1,
			duplicatePosition: 3,
			identityKind:      identityFingerprint,
		},
	}

	if !slices.Equal(result.unique, []importedTransaction{transaction}) {
		t.Errorf("unique transactions = %+v, want one transaction", result.unique)
	}

	if !slices.Equal(result.duplicates, wantDuplicates) {
		t.Errorf(
			"duplicates = %+v, want %+v",
			result.duplicates,
			wantDuplicates,
		)
	}
}

func TestDeduplicateImportedTransactionsPreservesOrder(t *testing.T) {
	first := importedTransaction{
		source:      sourceSwedbank,
		accountText: "account-1",
		externalID:  "record-1",
	}

	second := importedTransaction{
		source:      sourceSwedbank,
		accountText: "account-1",
		externalID:  "record-2",
	}

	third := importedTransaction{
		source:      sourceSwedbank,
		accountText: "account-1",
		externalID:  "record-3",
	}

	input := []importedTransaction{
		first,
		second,
		first,
		third,
		second,
	}

	result := deduplicateImportedTransactions(input)

	wantUnique := []importedTransaction{
		first,
		second,
		third,
	}

	if !slices.Equal(result.unique, wantUnique) {
		t.Errorf(
			"unique transactions = %+v, want %+v",
			result.unique,
			wantUnique,
		)
	}

	wantDuplicates := []duplicateCandidate{
		{
			transaction:       first,
			firstPosition:     1,
			duplicatePosition: 3,
			identityKind:      identityExternalID,
		},
		{
			transaction:       second,
			firstPosition:     2,
			duplicatePosition: 5,
			identityKind:      identityExternalID,
		},
	}

	if !slices.Equal(result.duplicates, wantDuplicates) {
		t.Errorf(
			"duplicates = %+v, want %+v",
			result.duplicates,
			wantDuplicates,
		)
	}
}

func TestDeduplicateImportedTransactionsScopesExternalIDByAccount(
	t *testing.T,
) {
	first := importedTransaction{
		source:      sourceSwedbank,
		accountText: "account-1",
		externalID:  "record-123",
	}

	second := first
	second.accountText = "account-2"

	input := []importedTransaction{first, second}
	result := deduplicateImportedTransactions(input)

	if !slices.Equal(result.unique, input) {
		t.Errorf(
			"unique transactions = %+v, want %+v",
			result.unique,
			input,
		)
	}

	if len(result.duplicates) != 0 {
		t.Errorf("duplicate count = %d, want 0", len(result.duplicates))
	}

	if len(result.conflicts) != 0 {
		t.Errorf("conflict count = %d, want 0", len(result.conflicts))
	}
}

func TestDeduplicateImportedTransactionsEmptyInput(t *testing.T) {
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := deduplicateImportedTransactions(tt.input)

			if len(result.unique) != 0 {
				t.Errorf(
					"unique transaction count = %d, want 0",
					len(result.unique),
				)
			}

			if len(result.duplicates) != 0 {
				t.Errorf(
					"duplicate count = %d, want 0",
					len(result.duplicates),
				)
			}

			if len(result.conflicts) != 0 {
				t.Errorf(
					"conflict count = %d, want 0",
					len(result.conflicts),
				)
			}
		})
	}
}
