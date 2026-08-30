package categorization

import (
	"errors"
	"testing"
)

func TestNormalizeText(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "trim and uppercase", input: "  Maxima lt ", want: "MAXIMA LT"},
		{name: "punctuation separates", input: "WOLT*ORDER-29382", want: "WOLT ORDER 29382"},
		{name: "collapse separators", input: "STRIPE***XYZ_SERVICES", want: "STRIPE XYZ SERVICES"},
		{name: "unicode", input: "Oü Näide", want: "OÜ NÄIDE"},
		{name: "empty", input: " \t ", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeText(tt.input); got != tt.want {
				t.Errorf("NormalizeText(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSuggestMerchantName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{input: "CARD PAYMENT MAXIMA LT VILNIUS LT", want: "Maxima"},
		{input: "SPOTIFY AB STOCKHOLM", want: "Spotify"},
		{input: "WOLT*ORDER 29382", want: "Wolt"},
		{input: "STRIPE*XYZSERVICES", want: "XYZ Services"},
		{input: "CIRCLE K LT 1234", want: "Circle K"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := SuggestMerchantName(tt.input); got != tt.want {
				t.Errorf("SuggestMerchantName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestMatchAnyKeepsCounterpartyAndDescriptionSeparate(t *testing.T) {
	rules := []Rule{
		{
			ID: 1, MatchType: MatchExact, NormalizedPattern: "MAXIMA",
			MerchantID: 1, CategoryID: 1, Enabled: true,
		},
	}

	got, err := MatchAny("swedbank", []string{"MAXIMA", "CARD PURCHASE"}, rules)
	if err != nil {
		t.Fatalf("MatchAny() returned an unexpected error: %v", err)
	}

	if got.Rule.ID != 1 {
		t.Errorf("winning rule = %d, want 1", got.Rule.ID)
	}
}

func TestMatchUsesPrioritySpecificityAndSource(t *testing.T) {
	rules := []Rule{
		{
			ID: 1, MatchType: MatchContains, NormalizedPattern: "MAXIMA",
			MerchantID: 1, CategoryID: 1, Priority: 10, Enabled: true,
		},
		{
			ID: 2, MatchType: MatchExact, NormalizedPattern: "MAXIMA LT",
			MerchantID: 2, CategoryID: 2, Priority: 10, Enabled: true,
		},
		{
			ID: 3, Source: "revolut", MatchType: MatchExact, NormalizedPattern: "MAXIMA LT",
			MerchantID: 3, CategoryID: 3, Priority: 20, Enabled: true,
		},
	}

	got, err := Match("swedbank", "maxima-lt", rules)
	if err != nil {
		t.Fatalf("Match() returned an unexpected error: %v", err)
	}

	if got.Rule.ID != 2 {
		t.Errorf("Swedbank winning rule = %d, want 2", got.Rule.ID)
	}

	got, err = Match("revolut", "maxima-lt", rules)
	if err != nil {
		t.Fatalf("Match() returned an unexpected error: %v", err)
	}

	if got.Rule.ID != 3 {
		t.Errorf("Revolut winning rule = %d, want 3", got.Rule.ID)
	}
}

func TestMatchDetectsEquallyPreferredDisagreement(t *testing.T) {
	rules := []Rule{
		{
			ID: 10, MatchType: MatchContains, NormalizedPattern: "MAXIMA",
			MerchantID: 1, CategoryID: 1, Priority: 100, Enabled: true,
		},
		{
			ID: 11, MatchType: MatchContains, NormalizedPattern: "VILNIU",
			MerchantID: 2, CategoryID: 2, Priority: 100, Enabled: true,
		},
	}

	got, err := Match("swedbank", "MAXIMA VILNIUS", rules)
	if !errors.Is(err, ErrRuleConflict) {
		t.Fatalf("Match() error = %v, want ErrRuleConflict", err)
	}

	if got.Rule != (Rule{}) {
		t.Errorf("winning rule = %+v, want zero Rule", got.Rule)
	}

	if len(got.Conflicts) != 2 {
		t.Errorf("conflict count = %d, want 2", len(got.Conflicts))
	}
}

func TestNormalizePatternRejectsInvalidRegex(t *testing.T) {
	if _, err := NormalizePattern(MatchRegex, "["); err == nil {
		t.Fatal("NormalizePattern() error = nil, want non-nil")
	}
}
