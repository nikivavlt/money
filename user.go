package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

type User struct {
	ID        int64
	Name      string
	CreatedAt time.Time
}

func (s *postgresStore) createUser(ctx context.Context, name string) (User, error) {
	normalized := strings.TrimSpace(name)
	if normalized == "" {
		return User{}, errors.New("user name is empty")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return User{}, fmt.Errorf("begin create user: %w", err)
	}
	defer tx.Rollback()

	const query = `
		INSERT INTO users (name)
		VALUES ($1)
		RETURNING id, name, created_at`

	row := tx.QueryRowContext(ctx, query, normalized)

	var user User
	err = row.Scan(&user.ID, &user.Name, &user.CreatedAt)
	if err != nil {
		return User{}, fmt.Errorf("create user: %w", err)
	}

	if err := seedDefaultCategories(ctx, tx, user.ID); err != nil {
		return User{}, fmt.Errorf("create user categories: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return User{}, fmt.Errorf("commit create user: %w", err)
	}

	return user, nil
}

func (s *postgresStore) findUserByID(ctx context.Context, id int64) (User, error) {
	const query = `
		SELECT id, name, created_at
		FROM users
		WHERE id = $1
	`

	row := s.db.QueryRowContext(ctx, query, id)

	var user User
	err := row.Scan(&user.ID, &user.Name, &user.CreatedAt)
	if err != nil {
		return User{}, fmt.Errorf("find user %d: %w", id, err)
	}

	return user, nil
}

func (s *postgresStore) listUsers(ctx context.Context) ([]User, error) {
	const query = `
		SELECT id, name, created_at
		FROM users
		ORDER BY id
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	var users []User

	for rows.Next() {
		var user User

		if err := rows.Scan(&user.ID, &user.Name, &user.CreatedAt); err != nil {
			return nil, fmt.Errorf("list users: scan row: %w", err)
		}

		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list users: iterate rows: %w", err)
	}

	return users, nil
}
