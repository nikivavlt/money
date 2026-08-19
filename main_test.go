package main

import (
	"bytes"
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
