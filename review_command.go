package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"money/internal/categorization"
	"money/internal/finance"
)

var errReviewQuit = errors.New("review quit")

func runReviewCommand(
	ctx context.Context,
	args []string,
	input io.Reader,
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

	var transactions []ReviewTransaction

	switch len(args) {
	case 0:
		transactions, err = store.listPendingReviews(ctx, userID)
		if err != nil {
			return fmt.Errorf("load review queue: %w", err)
		}
	case 1:
		transactionID, parseErr := strconv.ParseInt(args[0], 10, 64)
		if parseErr != nil || transactionID < 1 {
			return fmt.Errorf("transaction ID must be a positive integer")
		}

		transaction, findErr := store.findReviewTransaction(ctx, userID, transactionID)
		if findErr != nil {
			return findErr
		}
		transactions = []ReviewTransaction{transaction}
	default:
		return errors.New("usage: money review [transaction-id]")
	}
	if len(transactions) == 0 {
		_, err := fmt.Fprintln(output, "No transactions need review.")
		return err
	}

	scanner := bufio.NewScanner(input)

	for _, transaction := range transactions {
		categories, err := store.listCategories(ctx, userID)
		if err != nil {
			return fmt.Errorf("load review categories: %w", err)
		}

		err = reviewOneTransaction(
			ctx,
			store,
			userID,
			transaction,
			categories,
			scanner,
			output,
		)
		if errors.Is(err, errReviewQuit) {
			return nil
		}
		if err != nil {
			return err
		}
	}

	return nil
}

func reviewOneTransaction(
	ctx context.Context,
	store *postgresStore,
	userID int64,
	transaction ReviewTransaction,
	categories []Category,
	scanner *bufio.Scanner,
	output io.Writer,
) error {
	amount, err := financeFormatForReview(transaction)
	if err != nil {
		return err
	}

	merchantInput := transaction.Counterparty
	if strings.TrimSpace(merchantInput) == "" {
		merchantInput = transaction.Description
	}
	suggestedMerchant := categorization.SuggestMerchantName(merchantInput)

	if _, err := fmt.Fprintf(
		output,
		"\nTransaction %d\nDate:        %s\nAmount:      %s\nDescription: %s\nCounterparty: %s\nSuggested merchant: %s\n\n",
		transaction.ID,
		transaction.Date.Format("2006-01-02"),
		amount,
		transaction.Description,
		transaction.Counterparty,
		suggestedMerchant,
	); err != nil {
		return fmt.Errorf("write review transaction: %w", err)
	}

	for index, category := range categories {
		if _, err := fmt.Fprintf(output, "%d. %s\n", index+1, category.Name); err != nil {
			return fmt.Errorf("write review categories: %w", err)
		}
	}
	if _, err := fmt.Fprintln(output, "n. Create category\ns. Skip\nq. Quit"); err != nil {
		return fmt.Errorf("write review choices: %w", err)
	}

	category, action, err := readCategoryChoice(ctx, store, userID, categories, scanner, output)
	if err != nil {
		return err
	}

	switch action {
	case "quit":
		return errReviewQuit
	case "skip":
		if err := store.skipReview(ctx, userID, transaction.ID); err != nil {
			return err
		}
		_, err := fmt.Fprintln(output, "Skipped.")
		return err
	}

	merchantName, err := readDefaultedValue(scanner, output, "Merchant", suggestedMerchant)
	if err != nil {
		return err
	}
	if strings.TrimSpace(merchantName) == "" {
		return errors.New("merchant name cannot be empty")
	}

	merchant, category, err := store.applyManualCorrection(
		ctx,
		userID,
		ManualCorrection{
			TransactionID: transaction.ID,
			CategoryID:    category.ID,
			MerchantName:  merchantName,
		},
	)
	if err != nil {
		return fmt.Errorf("save manual correction: %w", err)
	}

	if _, err := fmt.Fprintf(output, "Saved: %s -> %s.\n", merchant.Name, category.Name); err != nil {
		return fmt.Errorf("write manual correction: %w", err)
	}

	createRule, err := readYesNo(scanner, output, "Create a source-specific exact rule for future matches? [y/N] ")
	if err != nil {
		return err
	}
	if !createRule {
		return nil
	}

	rule, err := store.learnMerchantRule(
		ctx,
		NewMerchantRule{
			UserID:       userID,
			Source:       transaction.Source,
			MatchType:    categorization.MatchExact,
			Pattern:      merchantInput,
			MerchantName: merchant.Name,
			CategoryName: category.Name,
			Priority:     100,
		},
	)
	if err != nil {
		return fmt.Errorf("learn merchant rule: %w", err)
	}

	if _, err := fmt.Fprintf(output, "Learned rule %d for %q.\n", rule.ID, rule.NormalizedPattern); err != nil {
		return fmt.Errorf("write learned merchant rule: %w", err)
	}

	return nil
}

func financeFormatForReview(transaction ReviewTransaction) (string, error) {
	formatted, err := finance.FormatMoney(transaction.Amount)
	if err != nil {
		return "", fmt.Errorf("format transaction %d amount: %w", transaction.ID, err)
	}

	return formatted, nil
}

func readCategoryChoice(
	ctx context.Context,
	store *postgresStore,
	userID int64,
	categories []Category,
	scanner *bufio.Scanner,
	output io.Writer,
) (Category, string, error) {
	for {
		value, err := readValue(scanner, output, "Choose category: ")
		if err != nil {
			return Category{}, "", err
		}

		switch strings.ToLower(value) {
		case "q":
			return Category{}, "quit", nil
		case "s":
			return Category{}, "skip", nil
		case "n":
			name, err := readValue(scanner, output, "New category name: ")
			if err != nil {
				return Category{}, "", err
			}

			category, err := store.createCategory(ctx, userID, name)
			if err != nil {
				return Category{}, "", fmt.Errorf("create review category: %w", err)
			}

			return category, "categorize", nil
		}

		selection, err := strconv.Atoi(value)
		if err == nil && selection >= 1 && selection <= len(categories) {
			return categories[selection-1], "categorize", nil
		}

		if _, err := fmt.Fprintln(output, "Enter a category number, n, s, or q."); err != nil {
			return Category{}, "", fmt.Errorf("write review validation: %w", err)
		}
	}
}

func readDefaultedValue(
	scanner *bufio.Scanner,
	output io.Writer,
	label string,
	defaultValue string,
) (string, error) {
	prompt := label + ": "
	if defaultValue != "" {
		prompt = fmt.Sprintf("%s [%s]: ", label, defaultValue)
	}

	value, err := readValue(scanner, output, prompt)
	if err != nil {
		return "", err
	}
	if value == "" {
		return defaultValue, nil
	}

	return value, nil
}

func readYesNo(scanner *bufio.Scanner, output io.Writer, prompt string) (bool, error) {
	value, err := readValue(scanner, output, prompt)
	if err != nil {
		return false, err
	}

	switch strings.ToLower(value) {
	case "", "n", "no":
		return false, nil
	case "y", "yes":
		return true, nil
	default:
		return false, fmt.Errorf("expected yes or no, got %q", value)
	}
}

func readValue(scanner *bufio.Scanner, output io.Writer, prompt string) (string, error) {
	if _, err := fmt.Fprint(output, prompt); err != nil {
		return "", fmt.Errorf("write review prompt: %w", err)
	}

	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return "", fmt.Errorf("read review input: %w", err)
		}

		return "", io.EOF
	}

	return strings.TrimSpace(scanner.Text()), nil
}
