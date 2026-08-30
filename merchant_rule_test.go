package main

import (
	"errors"
	"strings"
	"testing"

	"money/internal/categorization"
	"money/internal/statement"
)

func validNewMerchantRuleForTest() NewMerchantRule {
	return NewMerchantRule{
		UserID:       1,
		Source:       statement.Swedbank,
		MatchType:    categorization.MatchContains,
		Pattern:      "  maxima*lt  ",
		MerchantName: "Maxima",
		CategoryName: "Groceries",
		Priority:     100,
	}
}

func TestValidateNewMerchantRule(t *testing.T) {
	tests := []struct {
		name      string
		change    func(*NewMerchantRule)
		wantError string
	}{
		{
			name: "invalid user",
			change: func(input *NewMerchantRule) {
				input.UserID = 0
			},
			wantError: "merchant rule user ID must be positive",
		},
		{
			name: "invalid source",
			change: func(input *NewMerchantRule) {
				input.Source = statement.Source("unknown")
			},
			wantError: `unsupported merchant rule source "unknown"`,
		},
		{
			name: "empty merchant",
			change: func(input *NewMerchantRule) {
				input.MerchantName = "  "
			},
			wantError: "merchant rule merchant name is empty",
		},
		{
			name: "empty category",
			change: func(input *NewMerchantRule) {
				input.CategoryName = ""
			},
			wantError: "merchant rule category name is empty",
		},
		{
			name: "invalid match type",
			change: func(input *NewMerchantRule) {
				input.MatchType = categorization.MatchType("suffix")
			},
			wantError: `unsupported merchant rule match type "suffix"`,
		},
		{
			name: "invalid regex",
			change: func(input *NewMerchantRule) {
				input.MatchType = categorization.MatchRegex
				input.Pattern = "["
			},
			wantError: "compile merchant rule regular expression",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := validNewMerchantRuleForTest()
			tt.change(&input)

			_, err := validateNewMerchantRule(input)
			if err == nil {
				t.Fatal("validateNewMerchantRule() error = nil, want non-nil")
			}
			if tt.name == "invalid regex" {
				if !strings.Contains(err.Error(), tt.wantError) {
					t.Errorf("error = %q, want it to contain %q", err, tt.wantError)
				}
				return
			}
			if err.Error() != tt.wantError {
				t.Errorf("error = %q, want %q", err, tt.wantError)
			}
		})
	}
}

func TestValidateNewMerchantRuleNormalizesPattern(t *testing.T) {
	got, err := validateNewMerchantRule(validNewMerchantRuleForTest())
	if err != nil {
		t.Fatalf("validateNewMerchantRule() returned an unexpected error: %v", err)
	}

	if got != "MAXIMA LT" {
		t.Errorf("normalized pattern = %q, want %q", got, "MAXIMA LT")
	}
}

func TestParsePositiveRuleID(t *testing.T) {
	if got, err := parsePositiveRuleID("42"); err != nil || got != 42 {
		t.Errorf("parsePositiveRuleID(42) = (%d, %v), want (42, nil)", got, err)
	}

	for _, input := range []string{"", "abc", "0", "-1"} {
		if _, err := parsePositiveRuleID(input); err == nil {
			t.Errorf("parsePositiveRuleID(%q) error = nil, want non-nil", input)
		}
	}
}

func TestParseRuleAddArguments(t *testing.T) {
	got, err := parseRuleAddArguments(7, []string{
		"--match", "prefix",
		"--pattern", "WOLT*",
		"--merchant", "Wolt",
		"--category", "Food Delivery",
		"--priority", "50",
		"--source", "REVOLUT",
	})
	if err != nil {
		t.Fatalf("parseRuleAddArguments() returned an unexpected error: %v", err)
	}

	if got.UserID != 7 ||
		got.Source != statement.Revolut ||
		got.MatchType != categorization.MatchPrefix ||
		got.Pattern != "WOLT*" ||
		got.MerchantName != "Wolt" ||
		got.CategoryName != "Food Delivery" ||
		got.Priority != 50 {
		t.Errorf("parsed rule = %+v", got)
	}
}

func TestCreateMerchantRuleRejectsInvalidInputBeforeDatabaseAccess(t *testing.T) {
	store := newPostgresStore(nil)
	input := validNewMerchantRuleForTest()
	input.UserID = 0

	got, err := store.createMerchantRule(t.Context(), input)
	if err == nil {
		t.Fatal("createMerchantRule() error = nil, want non-nil")
	}
	if got != (MerchantRule{}) {
		t.Errorf("createMerchantRule() = %+v, want zero MerchantRule", got)
	}
	if errors.Is(err, ErrMerchantRuleAlreadyExists) {
		t.Errorf("error = %v, should not match ErrMerchantRuleAlreadyExists", err)
	}
}
