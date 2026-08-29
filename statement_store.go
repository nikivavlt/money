package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"money/internal/statement"
	"slices"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

type Fingerprint [sha256.Size]byte

type NewStatement struct {
	UserID           int64
	Source           statement.Source
	OriginalFilename string
	Fingerprint      Fingerprint
	RawHeader        []string
}

type Statement struct {
	ID               int64
	UserID           int64
	Source           statement.Source
	OriginalFilename string
	Fingerprint      Fingerprint
	RawHeader        []string
	ImportedAt       time.Time
}

var ErrStatementAlreadyImported = errors.New("statement already imported")

const postgresUniqueViolation = "23505"

func isUniqueViolation(err error) bool {
	var postgresErr *pgconn.PgError

	if !errors.As(err, &postgresErr) {
		return false
	}

	return postgresErr.Code == postgresUniqueViolation
}

func (s *postgresStore) createStatement(ctx context.Context, input NewStatement) (Statement, error) {
	return insertStatement(ctx, s.db, input)
}

func insertStatement(ctx context.Context, db rowQuerier, input NewStatement) (Statement, error) {
	if input.UserID < 1 {
		return Statement{}, errors.New("statement user ID must be positive")
	}

	if input.Source != statement.Revolut && input.Source != statement.Swedbank {
		return Statement{}, fmt.Errorf("unsupported statement source %q", input.Source)
	}

	normalized := strings.TrimSpace(input.OriginalFilename)
	if normalized == "" {
		return Statement{}, errors.New("statement filename is empty")
	}

	if input.Fingerprint == (Fingerprint{}) {
		return Statement{}, errors.New("statement fingerprint is empty")
	}

	if len(input.RawHeader) == 0 {
		return Statement{}, errors.New("statement header is empty")
	}

	headerJSON, err := json.Marshal(input.RawHeader)
	if err != nil {
		return Statement{}, fmt.Errorf("encode statement header: %w", err)
	}

	const query = `
		INSERT INTO statements (user_id, source, original_filename, fingerprint, raw_header)
		VALUES ($1, $2, $3, $4, $5::jsonb)
		RETURNING id, imported_at`

	row := db.QueryRowContext(
		ctx,
		query,
		input.UserID,
		string(input.Source),
		normalized,
		input.Fingerprint[:],
		string(headerJSON),
	)

	var created Statement

	if err = row.Scan(&created.ID, &created.ImportedAt); err != nil {
		if isUniqueViolation(err) {
			return Statement{}, fmt.Errorf("%w for user %d", ErrStatementAlreadyImported, input.UserID)
		}

		return Statement{}, fmt.Errorf("create statement: %w", err)
	}

	created.UserID = input.UserID
	created.Source = input.Source
	created.OriginalFilename = normalized
	created.Fingerprint = input.Fingerprint
	created.RawHeader = slices.Clone(input.RawHeader)

	return created, nil
}
