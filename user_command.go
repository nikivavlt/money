package main

import (
	"context"
	"fmt"
	"io"
	"time"
)

func runCreateUserCommand(ctx context.Context, name string, output io.Writer, getenv func(string) string) error {
	store, err := openCommandStore(ctx, getenv)
	if err != nil {
		return err
	}
	defer store.db.Close()

	user, err := store.createUser(ctx, name)
	if err != nil {
		return fmt.Errorf("create user: %w", err)
	}

	if err := writeCreatedUser(output, user); err != nil {
		return err
	}

	return nil
}

func runListUsersCommand(ctx context.Context, output io.Writer, getenv func(string) string) error {
	store, err := openCommandStore(ctx, getenv)
	if err != nil {
		return err
	}
	defer store.db.Close()

	users, err := store.listUsers(ctx)
	if err != nil {
		return fmt.Errorf("list users: %w", err)
	}

	if err := writeUsers(output, users); err != nil {
		return err
	}

	return nil
}

func writeCreatedUser(output io.Writer, user User) error {
	_, err := fmt.Fprintf(
		output,
		"User ID:    %d\n"+
			"Name:       %s\n"+
			"Created at: %s\n",
		user.ID,
		user.Name,
		user.CreatedAt.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("write created user: %w", err)
	}

	return nil
}

func writeUsers(output io.Writer, users []User) error {
	if len(users) == 0 {
		if _, err := fmt.Fprintln(output, "No users."); err != nil {
			return fmt.Errorf("write users: %w", err)
		}

		return nil
	}

	if _, err := fmt.Fprintln(output, "ID\tNAME\tCREATED_AT"); err != nil {
		return fmt.Errorf("write users: %w", err)
	}

	for _, user := range users {
		_, err := fmt.Fprintf(
			output,
			"%d\t%s\t%s\n",
			user.ID,
			user.Name,
			user.CreatedAt.UTC().Format(time.RFC3339),
		)
		if err != nil {
			return fmt.Errorf("write users: %w", err)
		}
	}

	return nil
}
