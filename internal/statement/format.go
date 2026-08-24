package statement

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
) (Source, error) {
	_, revolutErr := revolutColumnIndexes(header)
	_, swedbankErr := swedbankColumnIndexes(header)

	revolutMatches := revolutErr == nil
	swedbankMatches := swedbankErr == nil

	switch {
	case revolutMatches && !swedbankMatches:
		return Revolut, nil

	case !revolutMatches && swedbankMatches:
		return Swedbank, nil

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
