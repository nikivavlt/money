package main

import (
	"context"
	"fmt"
	"io"
	"os"
)

const version = "dev"

func main() {
	exitCode := run(
		context.Background(),
		os.Args[1:],
		os.Stdout,
		os.Stderr,
		os.Getenv,
	)

	if exitCode != 0 {
		os.Exit(exitCode)
	}
}

func run(
	ctx context.Context,
	args []string,
	stdout io.Writer,
	stderr io.Writer,
	getenv func(string) string,
) int {
	if len(args) == 0 {
		printHelp(stdout)
		return 0
	}

	command := args[0]
	rest := args[1:]

	switch command {
	case "help", "-h", "--help":
		if len(rest) != 0 {
			fmt.Fprintf(stderr, "money: help: unexpected argument %q\n", rest[0])
			return 2
		}

		printHelp(stdout)
		return 0
	case "version":
		if len(rest) != 0 {
			fmt.Fprintf(stderr, "money: version: unexpected argument %q\n", rest[0])
			return 2
		}

		printVersion(stdout)
		return 0
	case "import":
		if len(rest) != 1 {
			fmt.Fprintln(stderr, "money: import: usage: money import <path>")
			return 2
		}

		if err := runImportCommand(ctx, rest[0], stdout, getenv); err != nil {
			fmt.Fprintf(stderr, "money: import: %v\n", err)
			return 1
		}

		return 0
	case "user":
		if len(rest) != 2 || rest[0] != "create" {
			fmt.Fprintln(stderr, "money: user: usage: money user create <name>")
			return 2
		}

		if err := runCreateUserCommand(ctx, rest[1], stdout, getenv); err != nil {
			fmt.Fprintf(stderr, "money: user create: %v\n", err)
			return 1
		}

		return 0
	case "users":
		if len(rest) != 0 {
			fmt.Fprintf(stderr, "money: users: unexpected argument %q\n", rest[0])
			return 2
		}

		if err := runListUsersCommand(ctx, stdout, getenv); err != nil {
			fmt.Fprintf(stderr, "money: users: %v\n", err)
			return 1
		}

		return 0
	case "transactions":
		if len(rest) != 0 {
			fmt.Fprintf(stderr, "money: transactions: unexpected argument %q\n", rest[0])
			return 2
		}

		if err := runTransactionsCommand(ctx, stdout, getenv); err != nil {
			fmt.Fprintf(stderr, "money: transactions: %v\n", err)
			return 1
		}

		return 0
	case "month":
		if len(rest) != 1 {
			fmt.Fprintln(stderr, "money: month: usage: money month <YYYY-MM>")
			return 2
		}

		month, err := parseMonth(rest[0])
		if err != nil {
			fmt.Fprintf(stderr, "money: month: %v\n", err)
			return 2
		}

		if err := runMonthCommand(ctx, month, stdout, getenv); err != nil {
			fmt.Fprintf(stderr, "money: month: %v\n", err)
			return 1
		}

		return 0
	default:
		fmt.Fprintf(stderr, "money: unknown command %q\n", command)
		return 2
	}

	return 0
}

func printHelp(w io.Writer) {
	fmt.Fprintln(w, `money - automatic expense analyzer

Usage:
  money <command> [arguments]

Commands:
  import <path>           Import a bank statement
  transactions            List imported transactions
  month <YYYY-MM>         Show monthly cash flow
  user create <name>      Create a user
  users                   List users
  help                    Show help
  version                 Show version`)
}

func printVersion(w io.Writer) {
	fmt.Fprintln(w, "money", version)
}
