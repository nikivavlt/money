package main

import (
	"fmt"
	"io"
	"os"
)

const version = "dev"

func main() {
	if len(os.Args) < 2 {
		printHelp(os.Stdout)
		return
	}

	command := os.Args[1]
	rest := os.Args[2:]

	switch command {
	case "help", "-h", "--help":
		if len(rest) != 0 {
			fmt.Fprintf(os.Stderr, "money: help: unexpected argument %q\n", rest[0])
			os.Exit(2)
		}

		printHelp(os.Stdout)
	case "version":
		if len(rest) != 0 {
			fmt.Fprintf(os.Stderr, "money: version: unexpected argument %q\n", rest[0])
			os.Exit(2)
		}

		printVersion(os.Stdout)
	default:
		fmt.Fprintf(os.Stderr, "money: unknown command %q\n", command)
		os.Exit(2)
	}
}

func printHelp(w io.Writer) {
	fmt.Fprintln(w, `money - automatic expense analyzer

Usage:
  money <command>

Commands:
  help      Show help
  version   Show version`)
}

func printVersion(w io.Writer) {
	fmt.Fprintln(w, "money", version)
}
