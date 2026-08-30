CREATE TABLE categories (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,

    user_id BIGINT NOT NULL
        REFERENCES users(id)
        ON DELETE CASCADE,

    name TEXT NOT NULL
        CHECK (btrim(name) <> ''),

    is_default BOOLEAN NOT NULL
        DEFAULT FALSE,

    created_at TIMESTAMPTZ NOT NULL
        DEFAULT CURRENT_TIMESTAMP,

    UNIQUE (user_id, id)
);

CREATE UNIQUE INDEX categories_user_name_key
    ON categories (user_id, lower(name));

INSERT INTO categories (user_id, name, is_default)
SELECT users.id, defaults.name, TRUE
FROM users
CROSS JOIN (
    VALUES
        ('Income'),
        ('Housing'),
        ('Groceries'),
        ('Restaurants'),
        ('Food Delivery'),
        ('Transport'),
        ('Fuel'),
        ('Shopping'),
        ('Entertainment'),
        ('Subscriptions'),
        ('Health'),
        ('Insurance'),
        ('Travel'),
        ('Education'),
        ('Utilities'),
        ('Cash'),
        ('Transfers'),
        ('Fees'),
        ('Gifts'),
        ('Other')
) AS defaults(name)
ON CONFLICT (user_id, (lower(name))) DO NOTHING;

CREATE TABLE merchants (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,

    user_id BIGINT NOT NULL
        REFERENCES users(id)
        ON DELETE CASCADE,

    name TEXT NOT NULL
        CHECK (btrim(name) <> ''),

    normalized_name TEXT NOT NULL
        CHECK (btrim(normalized_name) <> ''),

    created_at TIMESTAMPTZ NOT NULL
        DEFAULT CURRENT_TIMESTAMP,

    updated_at TIMESTAMPTZ NOT NULL
        DEFAULT CURRENT_TIMESTAMP,

    UNIQUE (user_id, id),
    UNIQUE (user_id, normalized_name)
);

CREATE TABLE merchant_rules (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,

    user_id BIGINT NOT NULL
        REFERENCES users(id)
        ON DELETE CASCADE,

    source TEXT
        CHECK (
            source IS NULL
            OR source IN ('revolut', 'swedbank')
        ),

    match_type TEXT NOT NULL
        CHECK (match_type IN ('exact', 'prefix', 'contains', 'regex')),

    pattern TEXT NOT NULL
        CHECK (btrim(pattern) <> ''),

    normalized_pattern TEXT NOT NULL
        CHECK (btrim(normalized_pattern) <> ''),

    merchant_id BIGINT NOT NULL,

    category_id BIGINT NOT NULL,

    priority INTEGER NOT NULL
        DEFAULT 0,

    enabled BOOLEAN NOT NULL
        DEFAULT TRUE,

    created_at TIMESTAMPTZ NOT NULL
        DEFAULT CURRENT_TIMESTAMP,

    updated_at TIMESTAMPTZ NOT NULL
        DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT merchant_rules_merchant_owner_fk
        FOREIGN KEY (user_id, merchant_id)
        REFERENCES merchants(user_id, id),

    CONSTRAINT merchant_rules_category_owner_fk
        FOREIGN KEY (user_id, category_id)
        REFERENCES categories(user_id, id)
);

CREATE UNIQUE INDEX merchant_rules_unique_match_key
    ON merchant_rules (
        user_id,
        COALESCE(source, ''),
        match_type,
        normalized_pattern
    );

CREATE INDEX merchant_rules_matching_idx
    ON merchant_rules (user_id, enabled, priority DESC);

ALTER TABLE transactions
ADD COLUMN normalized_description TEXT NOT NULL
    DEFAULT '';

UPDATE transactions
SET normalized_description = upper(
    regexp_replace(
        btrim(COALESCE(counterparty, description)),
        '[^[:alnum:]]+',
        ' ',
        'g'
    )
);

ALTER TABLE transactions
ADD COLUMN merchant_id BIGINT
    REFERENCES merchants(id),
ADD COLUMN category_id BIGINT
    REFERENCES categories(id),
ADD COLUMN categorization_source TEXT
    CHECK (
        categorization_source IS NULL
        OR categorization_source IN ('rule', 'manual')
    ),
ADD COLUMN applied_rule_id BIGINT
    REFERENCES merchant_rules(id)
    ON DELETE SET NULL,
ADD COLUMN review_status TEXT NOT NULL
    DEFAULT 'pending'
    CHECK (review_status IN ('pending', 'resolved', 'skipped'));

CREATE INDEX transactions_merchant_id_idx
    ON transactions (merchant_id);

CREATE INDEX transactions_category_id_idx
    ON transactions (category_id);

CREATE INDEX transactions_review_status_idx
    ON transactions (review_status, transaction_date, id);

CREATE TABLE transaction_classifications (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,

    transaction_id BIGINT NOT NULL
        REFERENCES transactions(id)
        ON DELETE CASCADE,

    merchant_id BIGINT NOT NULL
        REFERENCES merchants(id),

    category_id BIGINT NOT NULL
        REFERENCES categories(id),

    source TEXT NOT NULL
        CHECK (source IN ('rule', 'manual')),

    rule_id BIGINT
        REFERENCES merchant_rules(id)
        ON DELETE SET NULL,

    created_at TIMESTAMPTZ NOT NULL
        DEFAULT CURRENT_TIMESTAMP,

    CHECK (
        (source = 'rule' AND rule_id IS NOT NULL)
        OR (source = 'manual' AND rule_id IS NULL)
    )
);

CREATE INDEX transaction_classifications_transaction_id_idx
    ON transaction_classifications (transaction_id, created_at DESC, id DESC);
