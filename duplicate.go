package main

import (
	"crypto/sha256"
	"encoding/binary"
	"io"
	"strings"
)

type transactionIdentityKind string

const (
	identityExternalID  transactionIdentityKind = "external_id"
	identityFingerprint transactionIdentityKind = "fingerprint"
)

type transactionIdentity struct {
	kind   transactionIdentityKind
	digest [sha256.Size]byte
}

type duplicateCandidate struct {
	transaction       importedTransaction
	firstPosition     int
	duplicatePosition int
	identityKind      transactionIdentityKind
}

type duplicateConflict struct {
	firstTransaction       importedTransaction
	conflictingTransaction importedTransaction
	firstPosition          int
	conflictingPosition    int
	identityKind           transactionIdentityKind
}

type deduplicationResult struct {
	unique     []importedTransaction
	duplicates []duplicateCandidate
	conflicts  []duplicateConflict
}

type firstOccurrence struct {
	transaction   importedTransaction
	position      int
	payloadDigest [sha256.Size]byte
}

func importedTransactionIdentity(
	transaction importedTransaction,
) transactionIdentity {
	source := string(transaction.source)
	account := strings.TrimSpace(transaction.accountText)
	externalID := strings.TrimSpace(transaction.externalID)

	if externalID != "" {
		return transactionIdentity{
			kind: identityExternalID,
			digest: hashIdentityParts(
				source,
				account,
				externalID,
			),
		}
	}

	return transactionIdentity{
		kind:   identityFingerprint,
		digest: importedTransactionPayloadDigest(transaction),
	}
}

func importedTransactionPayloadDigest(
	transaction importedTransaction,
) [sha256.Size]byte {
	return hashIdentityParts(
		string(transaction.source),
		strings.TrimSpace(transaction.accountText),
		strings.TrimSpace(transaction.occurredAtText),
		strings.TrimSpace(transaction.completedAtText),
		strings.TrimSpace(transaction.amountText),
		strings.TrimSpace(transaction.feeText),
		strings.TrimSpace(transaction.currencyText),
		strings.TrimSpace(transaction.directionText),
		strings.TrimSpace(transaction.rawDescription),
		strings.TrimSpace(transaction.counterpartyText),
		strings.TrimSpace(transaction.externalID),
		strings.TrimSpace(transaction.typeText),
		strings.TrimSpace(transaction.stateText),
	)
}

func hashIdentityParts(parts ...string) [sha256.Size]byte {
	hasher := sha256.New()
	var lengthBuffer [8]byte

	for _, part := range parts {
		binary.BigEndian.PutUint64(
			lengthBuffer[:],
			uint64(len(part)),
		)

		_, _ = hasher.Write(lengthBuffer[:])
		_, _ = io.WriteString(hasher, part)
	}

	var digest [sha256.Size]byte
	copy(digest[:], hasher.Sum(nil))

	return digest
}

func deduplicateImportedTransactions(
	transactions []importedTransaction,
) deduplicationResult {
	result := deduplicationResult{
		unique: make([]importedTransaction, 0, len(transactions)),
	}

	firstOccurrences := make(
		map[transactionIdentity]firstOccurrence,
		len(transactions),
	)

	for index, transaction := range transactions {
		position := index + 1
		identity := importedTransactionIdentity(transaction)
		payloadDigest := importedTransactionPayloadDigest(transaction)

		first, exists := firstOccurrences[identity]
		if exists {
			if payloadDigest != first.payloadDigest {
				result.conflicts = append(
					result.conflicts,
					duplicateConflict{
						firstTransaction:       first.transaction,
						conflictingTransaction: transaction,
						firstPosition:          first.position,
						conflictingPosition:    position,
						identityKind:           identity.kind,
					},
				)

				continue
			}

			result.duplicates = append(
				result.duplicates,
				duplicateCandidate{
					transaction:       transaction,
					firstPosition:     first.position,
					duplicatePosition: position,
					identityKind:      identity.kind,
				},
			)

			continue
		}

		firstOccurrences[identity] = firstOccurrence{
			transaction:   transaction,
			position:      position,
			payloadDigest: payloadDigest,
		}

		result.unique = append(result.unique, transaction)
	}

	return result
}
