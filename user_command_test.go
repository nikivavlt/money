package main

import (
	"bytes"
	"testing"
	"time"
)

func TestWriteCreatedUser(t *testing.T) {
	var output bytes.Buffer

	user := User{
		ID:   17,
		Name: "Test User",
		CreatedAt: time.Date(
			2026,
			time.August,
			29,
			12,
			30,
			0,
			0,
			time.FixedZone("test", 2*60*60),
		),
	}

	err := writeCreatedUser(&output, user)
	if err != nil {
		t.Fatalf("writeCreatedUser() returned an unexpected error: %v", err)
	}

	want := "" +
		"User ID:    17\n" +
		"Name:       Test User\n" +
		"Created at: 2026-08-29T10:30:00Z\n"

	if output.String() != want {
		t.Errorf("writeCreatedUser() output = %q, want %q", output.String(), want)
	}
}

func TestWriteUsers(t *testing.T) {
	var output bytes.Buffer

	users := []User{
		{
			ID:        1,
			Name:      "Alice",
			CreatedAt: time.Date(2026, time.August, 29, 10, 0, 0, 0, time.UTC),
		},
		{
			ID:        2,
			Name:      "Bob",
			CreatedAt: time.Date(2026, time.August, 30, 11, 30, 0, 0, time.UTC),
		},
	}

	err := writeUsers(&output, users)
	if err != nil {
		t.Fatalf("writeUsers() returned an unexpected error: %v", err)
	}

	want := "" +
		"ID\tNAME\tCREATED_AT\n" +
		"1\tAlice\t2026-08-29T10:00:00Z\n" +
		"2\tBob\t2026-08-30T11:30:00Z\n"

	if output.String() != want {
		t.Errorf("writeUsers() output = %q, want %q", output.String(), want)
	}
}

func TestWriteUsersEmpty(t *testing.T) {
	var output bytes.Buffer

	err := writeUsers(&output, nil)
	if err != nil {
		t.Fatalf("writeUsers() returned an unexpected error: %v", err)
	}

	if output.String() != "No users.\n" {
		t.Errorf("writeUsers() output = %q, want %q", output.String(), "No users.\n")
	}
}
