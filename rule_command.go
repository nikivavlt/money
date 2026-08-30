package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"strconv"
	"strings"

	"money/internal/categorization"
	"money/internal/statement"
)

func runRulesCommand(
	ctx context.Context,
	args []string,
	output io.Writer,
	getenv func(string) string,
) error {
	userID, err := parseConfiguredUserID(getenv("MONEY_USER_ID"))
	if err != nil {
		return err
	}

	store, err := openCommandStore(ctx, getenv)
	if err != nil {
		return err
	}
	defer store.db.Close()

	if len(args) == 0 {
		rules, err := store.listMerchantRules(ctx, userID)
		if err != nil {
			return fmt.Errorf("load merchant rules: %w", err)
		}

		return writeMerchantRules(output, rules)
	}

	switch args[0] {
	case "add":
		input, err := parseRuleAddArguments(userID, args[1:])
		if err != nil {
			return err
		}

		rule, err := store.createMerchantRule(ctx, input)
		if err != nil {
			return fmt.Errorf("add merchant rule: %w", err)
		}

		return writeCreatedMerchantRule(output, rule)
	case "apply":
		if len(args) != 1 {
			return rulesUsageError()
		}

		summary, err := store.applyMerchantRules(ctx, userID)
		if err != nil {
			return fmt.Errorf("apply merchant rules: %w", err)
		}

		return writeRuleApplicationSummary(output, summary)
	case "enable", "disable":
		if len(args) != 2 {
			return rulesUsageError()
		}

		ruleID, err := parsePositiveRuleID(args[1])
		if err != nil {
			return err
		}

		enabled := args[0] == "enable"
		if err := store.setMerchantRuleEnabled(ctx, userID, ruleID, enabled); err != nil {
			return err
		}

		state := "disabled"
		if enabled {
			state = "enabled"
		}
		_, err = fmt.Fprintf(output, "Rule %d %s.\n", ruleID, state)
		return err
	case "priority":
		if len(args) != 3 {
			return rulesUsageError()
		}

		ruleID, err := parsePositiveRuleID(args[1])
		if err != nil {
			return err
		}
		priority, err := strconv.Atoi(args[2])
		if err != nil {
			return fmt.Errorf("parse merchant rule priority %q: %w", args[2], err)
		}

		if err := store.setMerchantRulePriority(ctx, userID, ruleID, priority); err != nil {
			return err
		}

		_, err = fmt.Fprintf(output, "Rule %d priority set to %d.\n", ruleID, priority)
		return err
	default:
		return rulesUsageError()
	}
}

func parsePositiveRuleID(input string) (int64, error) {
	ruleID, err := strconv.ParseInt(input, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("parse merchant rule ID %q: %w", input, err)
	}
	if ruleID < 1 {
		return 0, errors.New("merchant rule ID must be positive")
	}

	return ruleID, nil
}

func parseRuleAddArguments(userID int64, args []string) (NewMerchantRule, error) {
	flags := flag.NewFlagSet("money rules add", flag.ContinueOnError)
	flags.SetOutput(io.Discard)

	matchType := flags.String("match", "", "exact, prefix, contains, or regex")
	pattern := flags.String("pattern", "", "bank text to match")
	merchant := flags.String("merchant", "", "canonical merchant name")
	category := flags.String("category", "", "configured category name")
	priority := flags.Int("priority", 0, "higher rules run first")
	source := flags.String("source", "", "revolut, swedbank, or empty for all")

	if err := flags.Parse(args); err != nil {
		return NewMerchantRule{}, fmt.Errorf("parse merchant rule: %w", err)
	}
	if flags.NArg() != 0 {
		return NewMerchantRule{}, rulesUsageError()
	}

	input := NewMerchantRule{
		UserID:       userID,
		Source:       statement.Source(strings.ToLower(strings.TrimSpace(*source))),
		MatchType:    categorization.MatchType(strings.ToLower(strings.TrimSpace(*matchType))),
		Pattern:      *pattern,
		MerchantName: *merchant,
		CategoryName: *category,
		Priority:     *priority,
	}

	if _, err := validateNewMerchantRule(input); err != nil {
		return NewMerchantRule{}, fmt.Errorf("invalid merchant rule: %w", err)
	}

	return input, nil
}

func rulesUsageError() error {
	return errors.New("usage: money rules [add --match <type> --pattern <text> --merchant <name> --category <name> [--priority <n>] [--source <bank>] | apply | enable <id> | disable <id> | priority <id> <n>]")
}

func writeCreatedMerchantRule(output io.Writer, rule MerchantRule) error {
	source := "all"
	if rule.Source != "" {
		source = string(rule.Source)
	}

	_, err := fmt.Fprintf(
		output,
		"Rule ID:  %d\n"+
			"Match:    %s %q\n"+
			"Source:   %s\n"+
			"Merchant: %s\n"+
			"Category: %s\n"+
			"Priority: %d\n",
		rule.ID,
		rule.MatchType,
		rule.Pattern,
		source,
		rule.Merchant.Name,
		rule.Category.Name,
		rule.Priority,
	)
	if err != nil {
		return fmt.Errorf("write merchant rule: %w", err)
	}

	return nil
}

func writeMerchantRules(output io.Writer, rules []MerchantRule) error {
	if len(rules) == 0 {
		if _, err := fmt.Fprintln(output, "No merchant rules."); err != nil {
			return fmt.Errorf("write merchant rules: %w", err)
		}

		return nil
	}

	if _, err := fmt.Fprintln(output, "ID\tENABLED\tPRIORITY\tSOURCE\tMATCH\tPATTERN\tMERCHANT\tCATEGORY"); err != nil {
		return fmt.Errorf("write merchant rules: %w", err)
	}

	for _, rule := range rules {
		source := "all"
		if rule.Source != "" {
			source = string(rule.Source)
		}

		if _, err := fmt.Fprintf(
			output,
			"%d\t%t\t%d\t%s\t%s\t%q\t%q\t%q\n",
			rule.ID,
			rule.Enabled,
			rule.Priority,
			source,
			rule.MatchType,
			rule.Pattern,
			rule.Merchant.Name,
			rule.Category.Name,
		); err != nil {
			return fmt.Errorf("write merchant rules: %w", err)
		}
	}

	return nil
}

func writeRuleApplicationSummary(output io.Writer, summary RuleApplicationSummary) error {
	if _, err := fmt.Fprintf(
		output,
		"Transactions considered: %d\n"+
			"Transactions classified: %d\n"+
			"Rule conflicts:          %d\n",
		summary.Considered,
		summary.Classified,
		len(summary.Conflicts),
	); err != nil {
		return fmt.Errorf("write merchant rule application summary: %w", err)
	}

	for _, conflict := range summary.Conflicts {
		if _, err := fmt.Fprintf(
			output,
			"Conflict: transaction %d matched rules %v\n",
			conflict.TransactionID,
			conflict.RuleIDs,
		); err != nil {
			return fmt.Errorf("write merchant rule conflict: %w", err)
		}
	}

	return nil
}
