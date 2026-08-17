package main

import "slices"

func filterOutflows(transactions []Transaction) []Transaction {
	filtered := make([]Transaction, 0)

	for _, t := range transactions {
		if t.IsOutflow() {
			filtered = append(filtered, t)
		}
	}

	return filtered
}

func groupTransactionsByDescription(transactions []Transaction) map[string][]Transaction {
	grouped := make(map[string][]Transaction)

	for _, t := range transactions {
		grouped[t.Description] = append(grouped[t.Description], t)
	}

	return grouped
}

func sortedDescriptions(grouped map[string][]Transaction) []string {
	descriptions := make([]string, 0, len(grouped))

	for description := range grouped {
		descriptions = append(descriptions, description)
	}

	slices.Sort(descriptions)

	return descriptions
}
