package main

import (
	"errors"
	"fmt"
)

var (
	ErrUnknownStatementFormat = errors.New("unknown statement format")

	ErrAmbiguousStatementFormat = errors.New(
		"ambiguous statement format",
	)
)

func detectStatementSource(
	header []string,
) (statementSource, error) {
	_, revolutErr := revolutColumnIndexes(header)
	_, swedbankErr := swedbankColumnIndexes(header)

	revolutMatches := revolutErr == nil
	swedbankMatches := swedbankErr == nil

	switch {
	case revolutMatches && !swedbankMatches:
		return sourceRevolut, nil

	case !revolutMatches && swedbankMatches:
		return sourceSwedbank, nil

	case revolutMatches && swedbankMatches:
		return "", fmt.Errorf(
			"%w: headers satisfy both schemas",
			ErrAmbiguousStatementFormat,
		)

	default:
		return "", fmt.Errorf(
			"%w: revolut schema: %v; swedbank schema: %v",
			ErrUnknownStatementFormat,
			revolutErr,
			swedbankErr,
		)
	}
}
