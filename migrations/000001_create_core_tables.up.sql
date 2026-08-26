CREATE TABLE users (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,

    name TEXT NOT NULL
        CHECK (btrim(name) <> ''),

    created_at TIMESTAMPTZ NOT NULL
        DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE statements (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,

    user_id BIGINT NOT NULL
        REFERENCES users(id),

    source TEXT NOT NULL
        CHECK (source IN ('revolut', 'swedbank')),

    original_filename TEXT NOT NULL
        CHECK (btrim(original_filename) <> ''),

    fingerprint BYTEA NOT NULL
        CHECK (octet_length(fingerprint) = 32),

    raw_header JSONB NOT NULL
        CHECK (jsonb_typeof(raw_header) = 'array'),

    imported_at TIMESTAMPTZ NOT NULL
        DEFAULT CURRENT_TIMESTAMP,

    UNIQUE (user_id, fingerprint)
);

CREATE TABLE transactions (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,

    statement_id BIGINT NOT NULL
        REFERENCES statements(id),

    fingerprint BYTEA NOT NULL UNIQUE
        CHECK (octet_length(fingerprint) = 32),

    transaction_date DATE NOT NULL,

    amount_minor BIGINT NOT NULL,

    currency TEXT NOT NULL
        CHECK (currency ~ '^[A-Z]{3}$'),

    description TEXT NOT NULL
        CHECK (btrim(description) <> ''),

    counterparty TEXT
        CHECK (
            counterparty IS NULL
            OR btrim(counterparty) <> ''
        ),

    raw_record JSONB NOT NULL
        CHECK (jsonb_typeof(raw_record) = 'array'),

    created_at TIMESTAMPTZ NOT NULL
        DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX transactions_statement_id_idx
    ON transactions (statement_id);

CREATE INDEX transactions_transaction_date_idx
    ON transactions (transaction_date);