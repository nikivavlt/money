package main

import (
	"money/internal/finance"
	"slices"
)

func filterOutflows(transactions []finance.Transaction) []finance.Transaction {
	filtered := make([]finance.Transaction, 0)

	for _, t := range transactions {
		if t.IsOutflow() {
			filtered = append(filtered, t)
		}
	}

	return filtered
}

func groupTransactionsByDescription(transactions []finance.Transaction) map[string][]finance.Transaction {
	grouped := make(map[string][]finance.Transaction)

	for _, t := range transactions {
		grouped[t.Description] = append(grouped[t.Description], t)
	}

	return grouped
}

func sortedDescriptions(grouped map[string][]finance.Transaction) []string {
	descriptions := make([]string, 0, len(grouped))

	for description := range grouped {
		descriptions = append(descriptions, description)
	}

	slices.Sort(descriptions)

	return descriptions
}
