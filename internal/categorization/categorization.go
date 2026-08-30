package categorization

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

type MatchType string

const (
	MatchExact    MatchType = "exact"
	MatchPrefix   MatchType = "prefix"
	MatchContains MatchType = "contains"
	MatchRegex    MatchType = "regex"
)

var ErrRuleConflict = errors.New("merchant rule conflict")

type Rule struct {
	ID                int64
	Source            string
	MatchType         MatchType
	Pattern           string
	NormalizedPattern string
	MerchantID        int64
	MerchantName      string
	CategoryID        int64
	CategoryName      string
	Priority          int
	Enabled           bool
}

type MatchResult struct {
	Rule      Rule
	Conflicts []Rule
}

func NormalizeText(input string) string {
	var output strings.Builder
	output.Grow(len(input))

	previousWasSpace := true

	for _, character := range strings.TrimSpace(input) {
		if unicode.IsLetter(character) || unicode.IsDigit(character) {
			output.WriteRune(unicode.ToUpper(character))
			previousWasSpace = false
			continue
		}

		if !previousWasSpace {
			output.WriteByte(' ')
			previousWasSpace = true
		}
	}

	return strings.TrimSpace(output.String())
}

func MatchText(counterparty string, description string) string {
	if normalized := NormalizeText(counterparty); normalized != "" {
		return normalized
	}

	return NormalizeText(description)
}

func SuggestMerchantName(input string) string {
	words := strings.Fields(NormalizeText(input))
	if len(words) == 0 {
		return ""
	}

	words = trimTransactionPrefix(words)
	words = trimProcessorPrefix(words)
	words = trimMerchantSuffix(words)
	words = splitDisplayWords(words)

	if len(words) == 0 {
		return ""
	}

	for index, word := range words {
		words[index] = titleWord(word)
	}

	return strings.Join(words, " ")
}

func splitDisplayWords(words []string) []string {
	result := make([]string, 0, len(words)+1)

	for _, word := range words {
		const suffix = "SERVICES"
		if strings.HasSuffix(word, suffix) && len(word) > len(suffix) {
			result = append(result, strings.TrimSuffix(word, suffix), suffix)
			continue
		}

		result = append(result, word)
	}

	return result
}

func trimTransactionPrefix(words []string) []string {
	prefixes := [][]string{
		{"CARD", "PAYMENT"},
		{"CARD", "PURCHASE"},
		{"CASH", "WITHDRAWAL"},
		{"PAYMENT"},
		{"PURCHASE"},
	}

	for _, prefix := range prefixes {
		if len(words) >= len(prefix) && slices.Equal(words[:len(prefix)], prefix) {
			return words[len(prefix):]
		}
	}

	return words
}

func trimProcessorPrefix(words []string) []string {
	if len(words) < 2 {
		return words
	}

	switch words[0] {
	case "STRIPE", "PAYPAL":
		return words[1:]
	default:
		return words
	}
}

func trimMerchantSuffix(words []string) []string {
	for index, word := range words {
		switch word {
		case "ORDER":
			if index > 0 {
				return words[:index]
			}
		case "AB", "AS", "GMBH", "INC", "LLC", "LTD", "OU", "OÜ", "SIA", "UAB":
			if index > 0 {
				return words[:index]
			}
		case "EE", "GB", "LT", "LTU", "LV", "LVA":
			if index > 0 {
				return words[:index]
			}
		}

		if index > 0 && isDigits(word) {
			return words[:index]
		}
	}

	return words
}

func isDigits(value string) bool {
	if value == "" {
		return false
	}

	for _, character := range value {
		if !unicode.IsDigit(character) {
			return false
		}
	}

	return true
}

func titleWord(word string) string {
	if utf8.RuneCountInString(word) <= 3 {
		return word
	}

	characters := []rune(strings.ToLower(word))
	characters[0] = unicode.ToUpper(characters[0])

	return string(characters)
}

func NormalizePattern(matchType MatchType, pattern string) (string, error) {
	if !matchType.Valid() {
		return "", fmt.Errorf("unsupported merchant rule match type %q", matchType)
	}

	if matchType == MatchRegex {
		normalized := strings.TrimSpace(pattern)
		if normalized == "" {
			return "", errors.New("merchant rule pattern is empty")
		}

		if _, err := regexp.Compile("(?i:" + normalized + ")"); err != nil {
			return "", fmt.Errorf("compile merchant rule regular expression: %w", err)
		}

		return normalized, nil
	}

	normalized := NormalizeText(pattern)
	if normalized == "" {
		return "", errors.New("merchant rule pattern is empty")
	}

	return normalized, nil
}

func (matchType MatchType) Valid() bool {
	switch matchType {
	case MatchExact, MatchPrefix, MatchContains, MatchRegex:
		return true
	default:
		return false
	}
}

func Match(source string, text string, rules []Rule) (MatchResult, error) {
	return MatchAny(source, []string{text}, rules)
}

func MatchAny(source string, texts []string, rules []Rule) (MatchResult, error) {
	normalizedTexts := make([]string, 0, len(texts))
	seenTexts := make(map[string]struct{}, len(texts))

	for _, text := range texts {
		normalized := NormalizeText(text)
		if normalized == "" {
			continue
		}
		if _, exists := seenTexts[normalized]; exists {
			continue
		}

		seenTexts[normalized] = struct{}{}
		normalizedTexts = append(normalizedTexts, normalized)
	}

	if len(normalizedTexts) == 0 {
		return MatchResult{}, nil
	}

	matching := make([]Rule, 0, len(rules))
	seenRules := make(map[int64]struct{}, len(rules))

	for _, rule := range rules {
		if !rule.Enabled || (rule.Source != "" && rule.Source != source) {
			continue
		}

		for _, normalizedText := range normalizedTexts {
			matched, err := ruleMatches(normalizedText, rule)
			if err != nil {
				return MatchResult{}, fmt.Errorf("match merchant rule %d: %w", rule.ID, err)
			}

			if matched {
				if _, exists := seenRules[rule.ID]; !exists {
					matching = append(matching, rule)
					seenRules[rule.ID] = struct{}{}
				}
				break
			}
		}
	}

	if len(matching) == 0 {
		return MatchResult{}, nil
	}

	slices.SortStableFunc(matching, compareRulePrecedence)
	winner := matching[0]
	conflicts := []Rule{winner}

	for _, candidate := range matching[1:] {
		if !samePrecedence(winner, candidate) {
			break
		}

		if candidate.MerchantID != winner.MerchantID || candidate.CategoryID != winner.CategoryID {
			conflicts = append(conflicts, candidate)
		}
	}

	if len(conflicts) > 1 {
		return MatchResult{Conflicts: conflicts}, fmt.Errorf(
			"%w: %d equally preferred rules matched transaction text",
			ErrRuleConflict,
			len(conflicts),
		)
	}

	return MatchResult{Rule: winner}, nil
}

func ruleMatches(text string, rule Rule) (bool, error) {
	pattern := rule.NormalizedPattern
	if pattern == "" {
		var err error
		pattern, err = NormalizePattern(rule.MatchType, rule.Pattern)
		if err != nil {
			return false, err
		}
	}

	switch rule.MatchType {
	case MatchExact:
		return text == pattern, nil
	case MatchPrefix:
		return strings.HasPrefix(text, pattern), nil
	case MatchContains:
		return strings.Contains(text, pattern), nil
	case MatchRegex:
		expression, err := regexp.Compile("(?i:" + pattern + ")")
		if err != nil {
			return false, fmt.Errorf("compile regular expression: %w", err)
		}

		return expression.MatchString(text), nil
	default:
		return false, fmt.Errorf("unsupported match type %q", rule.MatchType)
	}
}

func compareRulePrecedence(left Rule, right Rule) int {
	if left.Priority != right.Priority {
		return right.Priority - left.Priority
	}

	leftRank := matchTypeRank(left.MatchType)
	rightRank := matchTypeRank(right.MatchType)

	if leftRank != rightRank {
		return rightRank - leftRank
	}

	leftLength := utf8.RuneCountInString(left.NormalizedPattern)
	rightLength := utf8.RuneCountInString(right.NormalizedPattern)

	if leftLength != rightLength {
		return rightLength - leftLength
	}

	if left.ID < right.ID {
		return -1
	}

	if left.ID > right.ID {
		return 1
	}

	return 0
}

func samePrecedence(left Rule, right Rule) bool {
	return left.Priority == right.Priority &&
		matchTypeRank(left.MatchType) == matchTypeRank(right.MatchType) &&
		utf8.RuneCountInString(left.NormalizedPattern) == utf8.RuneCountInString(right.NormalizedPattern)
}

func matchTypeRank(matchType MatchType) int {
	switch matchType {
	case MatchExact:
		return 4
	case MatchPrefix:
		return 3
	case MatchContains:
		return 2
	case MatchRegex:
		return 1
	default:
		return 0
	}
}
