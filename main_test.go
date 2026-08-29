package main

import (
	"bytes"
	"context"
	"strings"
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

func TestRunHelpUsesProvidedStdout(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := run(context.Background(), []string{"help"}, &stdout, &stderr, func(string) string {
		return ""
	})

	if exitCode != 0 {
		t.Errorf("run() exit code = %d, want 0", exitCode)
	}

	if !strings.Contains(stdout.String(), "Usage:") {
		t.Errorf("run() stdout = %q, want usage information", stdout.String())
	}

	if stderr.Len() != 0 {
		t.Errorf("run() stderr = %q, want empty", stderr.String())
	}
}

func TestRunVersionUsesProvidedStdout(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := run(context.Background(), []string{"version"}, &stdout, &stderr, func(string) string {
		return ""
	})

	if exitCode != 0 {
		t.Errorf("run() exit code = %d, want 0", exitCode)
	}

	want := "money dev\n"

	if stdout.String() != want {
		t.Errorf("run() stdout = %q, want %q", stdout.String(), want)
	}

	if stderr.Len() != 0 {
		t.Errorf("run() stderr = %q, want empty", stderr.String())
	}
}

func TestRunUnknownCommandUsesProvidedStderr(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := run(context.Background(), []string{"unknown"}, &stdout, &stderr, func(string) string {
		return ""
	})

	if exitCode != 2 {
		t.Errorf("run() exit code = %d, want 2", exitCode)
	}

	want := "money: unknown command \"unknown\"\n"

	if stderr.String() != want {
		t.Errorf("run() stderr = %q, want %q", stderr.String(), want)
	}

	if stdout.Len() != 0 {
		t.Errorf("run() stdout = %q, want empty", stdout.String())
	}
}
