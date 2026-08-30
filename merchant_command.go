package main

import (
	"context"
	"fmt"
	"io"
)

func runMerchantsCommand(
	ctx context.Context,
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

	merchants, err := store.listMerchants(ctx, userID)
	if err != nil {
		return fmt.Errorf("load merchants: %w", err)
	}

	return writeMerchants(output, merchants)
}

func writeMerchants(output io.Writer, merchants []Merchant) error {
	if len(merchants) == 0 {
		if _, err := fmt.Fprintln(output, "No merchants."); err != nil {
			return fmt.Errorf("write merchants: %w", err)
		}

		return nil
	}

	if _, err := fmt.Fprintln(output, "ID\tNAME\tNORMALIZED_NAME"); err != nil {
		return fmt.Errorf("write merchants: %w", err)
	}

	for _, merchant := range merchants {
		if _, err := fmt.Fprintf(
			output,
			"%d\t%s\t%s\n",
			merchant.ID,
			merchant.Name,
			merchant.NormalizedName,
		); err != nil {
			return fmt.Errorf("write merchants: %w", err)
		}
	}

	return nil
}
