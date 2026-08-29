package main

import (
	"context"
	"fmt"
	"io"
	"time"

	"money/internal/finance"
)

func parseMonth(input string) (time.Time, error) {
	month, err := time.Parse("2006-01", input)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse month %q: expected YYYY-MM: %w", input, err)
	}

	return month, nil
}

func runMonthCommand(
	ctx context.Context,
	month time.Time,
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

	summaries, err := store.summarizeMonthByUser(ctx, userID, month)
	if err != nil {
		return fmt.Errorf("load monthly summary: %w", err)
	}

	if err := writeMonthlyCashFlow(output, month, summaries); err != nil {
		return err
	}

	return nil
}

func writeMonthlyCashFlow(
	output io.Writer,
	month time.Time,
	summaries []monthlyCashFlow,
) error {
	monthText := month.Format("2006-01")

	if len(summaries) == 0 {
		if _, err := fmt.Fprintf(output, "No transactions for %s.\n", monthText); err != nil {
			return fmt.Errorf("write monthly summary: %w", err)
		}

		return nil
	}

	if _, err := fmt.Fprintln(output, "MONTH\tCURRENCY\tTRANSACTIONS\tINCOME\tEXPENSES\tSAVINGS"); err != nil {
		return fmt.Errorf("write monthly summary: %w", err)
	}

	for _, summary := range summaries {
		income, err := finance.FormatMoney(summary.Income)
		if err != nil {
			return fmt.Errorf("write monthly income: %w", err)
		}

		expenses, err := finance.FormatMoney(summary.Expenses)
		if err != nil {
			return fmt.Errorf("write monthly expenses: %w", err)
		}

		savings, err := finance.FormatMoney(summary.Savings)
		if err != nil {
			return fmt.Errorf("write monthly savings: %w", err)
		}

		_, err = fmt.Fprintf(
			output,
			"%s\t%s\t%d\t%s\t%s\t%s\n",
			monthText,
			summary.Currency,
			summary.TransactionCount,
			income,
			expenses,
			savings,
		)
		if err != nil {
			return fmt.Errorf("write monthly summary: %w", err)
		}
	}

	return nil
}
