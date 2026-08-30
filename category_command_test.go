	package main

import (
	"bytes"
	"testing"
)

func TestWriteCategories(t *testing.T) {
	var output bytes.Buffer

	err := writeCategories(&output, []Category{
		{ID: 1, Name: "Groceries", IsDefault: true},
		{ID: 2, Name: "Pets", IsDefault: false},
	})
	if err != nil {
		t.Fatalf("writeCategories() returned an unexpected error: %v", err)
	}

	want := "ID\tNAME\tKIND\n1\tGroceries\tdefault\n2\tPets\tcustom\n"
	if output.String() != want {
		t.Errorf("output = %q, want %q", output.String(), want)
	}
}
