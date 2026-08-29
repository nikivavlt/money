ALTER TABLE transactions
DROP CONSTRAINT transactions_statement_id_fingerprint_key;

ALTER TABLE transactions
ADD CONSTRAINT transactions_fingerprint_key
UNIQUE (fingerprint);