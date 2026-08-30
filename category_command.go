package main

import (
	"context"
	"fmt"
	"io"
)

func runCategoriesCommand(
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

	switch {
	case len(args) == 0:
		categories, err := store.listCategories(ctx, userID)
		if err != nil {
			return fmt.Errorf("load categories: %w", err)
		}

		return writeCategories(output, categories)
	case len(args) == 2 && args[0] == "add":
		category, err := store.createCategory(ctx, userID, args[1])
		if err != nil {
			return fmt.Errorf("add category: %w", err)
		}

		if _, err := fmt.Fprintf(output, "Category %q has ID %d.\n", category.Name, category.ID); err != nil {
			return fmt.Errorf("write category: %w", err)
		}

		return nil
	default:
		return fmt.Errorf("usage: money categories [add <name>]")
	}
}

func writeCategories(output io.Writer, categories []Category) error {
	if len(categories) == 0 {
		if _, err := fmt.Fprintln(output, "No categories."); err != nil {
			return fmt.Errorf("write categories: %w", err)
		}

		return nil
	}

	if _, err := fmt.Fprintln(output, "ID\tNAME\tKIND"); err != nil {
		return fmt.Errorf("write categories: %w", err)
	}

	for _, category := range categories {
		kind := "custom"
		if category.IsDefault {
			kind = "default"
		}

		if _, err := fmt.Fprintf(output, "%d\t%s\t%s\n", category.ID, category.Name, kind); err != nil {
			return fmt.Errorf("write categories: %w", err)
		}
	}

	return nil
}
