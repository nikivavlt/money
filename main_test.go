package main

import (
	"bytes"
	"context"
	"testing"
)

func TestPrintVersion(t *testing.T) {
	var output bytes.Buffer

	printVersion(&output)

	want := "money dev\n"
	got := output.String()

	if got != want {
		t.Errorf("printVersion() output = %q, want %q", got, want)
	}
}

func TestRunUserCreateRequiresName(t *testing.T) {
	var (
		stdout bytes.Buffer
		stderr bytes.Buffer
	)

	exitCode := run(
		context.Background(),
		[]string{"user", "create"},
		&stdout,
		&stderr,
		func(string) string {
			return ""
		},
	)

	if exitCode != 2 {
		t.Errorf("run() exit code = %d, want 2", exitCode)
	}

	want := "money: user: usage: money user create <name>\n"

	if stderr.String() != want {
		t.Errorf("run() stderr = %q, want %q", stderr.String(), want)
	}
}

func TestRunUsersRejectsArguments(t *testing.T) {
	var (
		stdout bytes.Buffer
		stderr bytes.Buffer
	)

	exitCode := run(
		context.Background(),
		[]string{"users", "unexpected"},
		&stdout,
		&stderr,
		func(string) string {
			return ""
		},
	)

	if exitCode != 2 {
		t.Errorf("run() exit code = %d, want 2", exitCode)
	}

	want := `money: users: unexpected argument "unexpected"` + "\n"

	if stderr.String() != want {
		t.Errorf("run() stderr = %q, want %q", stderr.String(), want)
	}
}
