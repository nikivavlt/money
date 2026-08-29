package main

import (
	"context"
	"fmt"
	"io"

	"money/internal/finance"
)

func runTransactionsCommand(
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

	transactions, err := store.listTransactionsByUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("load transactions: %w", err)
	}

	if err := writeTransactions(output, transactions); err != nil {
		return err
	}

	return nil
}

func writeTransactions(
	output io.Writer,
	transactions []listedTransaction,
) error {
	if len(transactions) == 0 {
		if _, err := fmt.Fprintln(output, "No transactions."); err != nil {
			return fmt.Errorf("write transactions: %w", err)
		}

		return nil
	}

	if _, err := fmt.Fprintln(output, "ID\tDATE\tAMOUNT\tDESCRIPTION\tCOUNTERPARTY\tSOURCE\tSTATEMENT\tFILE"); err != nil {
		return fmt.Errorf("write transactions: %w", err)
	}

	for _, transaction := range transactions {
		amount, err := finance.FormatMoney(transaction.Transaction.Amount)
		if err != nil {
			return fmt.Errorf("write transaction %d: %w", transaction.ID, err)
		}

		_, err = fmt.Fprintf(
			output,
			"%d\t%s\t%s\t%q\t%q\t%s\t%d\t%q\n",
			transaction.ID,
			transaction.Transaction.Date.Format("2006-01-02"),
			amount,
			transaction.Transaction.Description,
			transaction.Transaction.Counterparty,
			transaction.Source,
			transaction.StatementID,
			transaction.OriginalFilename,
		)
		if err != nil {
			return fmt.Errorf("write transactions: %w", err)
		}
	}

	return nil
}
