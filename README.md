# Money

Money is a personal expense analyzer written in Go. It imports Revolut and
Swedbank CSV statements, stores transactions in PostgreSQL, calculates monthly
cash flow, and categorizes merchants using deterministic rules and manual
corrections.

## Setup

Create a PostgreSQL database and set its connection URL:

```sh
export MONEY_DATABASE_URL='postgres://user:password@localhost:5432/money?sslmode=disable'
```

Apply the migrations in order:

```sh
psql "$MONEY_DATABASE_URL" -f migrations/000001_create_core_tables.up.sql
psql "$MONEY_DATABASE_URL" -f migrations/000002_scope_transaction_fingerprint.up.sql
psql "$MONEY_DATABASE_URL" -f migrations/000003_add_categories_and_merchants.up.sql
```

## Run

Create a user:

```sh
go run . user create "Nikita"
```

Set the returned user ID:

```sh
export MONEY_USER_ID=1
```

Import a statement and inspect the results:

```sh
go run . import statement.csv
go run . transactions
go run . month 2026-08
```

## Categorization

List categories and add a merchant rule:

```sh
go run . categories
go run . rules add --match contains --pattern MAXIMA --merchant Maxima --category Groceries --priority 100
go run . rules apply
```

Review transactions that were not categorized automatically:

```sh
go run . review
```

Run `go run . help` to see all available commands.

## Configuration

| Variable                  | Description                                      |
| ------------------------- | ------------------------------------------------ |
| `MONEY_DATABASE_URL`      | PostgreSQL connection URL                        |
| `MONEY_USER_ID`           | User used by imports, reports, rules, and review |
| `MONEY_TEST_DATABASE_URL` | PostgreSQL URL for integration tests             |

## Test

```sh
go test ./...
go test -race ./...
```

PostgreSQL integration tests run when `MONEY_TEST_DATABASE_URL` is set.
