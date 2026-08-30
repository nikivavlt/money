package main

import (
	"bytes"
	"testing"
)

func TestWriteMerchants(t *testing.T) {
	var output bytes.Buffer

	err := writeMerchants(&output, []Merchant{
		{ID: 9, Name: "Maxima", NormalizedName: "MAXIMA"},
	})
	if err != nil {
		t.Fatalf("writeMerchants() returned an unexpected error: %v", err)
	}

	want := "ID\tNAME\tNORMALIZED_NAME\n9\tMaxima\tMAXIMA\n"
	if output.String() != want {
		t.Errorf("output = %q, want %q", output.String(), want)
	}
}
