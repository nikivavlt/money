DROP TABLE transaction_classifications;

DROP INDEX transactions_review_status_idx;
DROP INDEX transactions_category_id_idx;
DROP INDEX transactions_merchant_id_idx;

ALTER TABLE transactions
DROP COLUMN review_status,
DROP COLUMN applied_rule_id,
DROP COLUMN categorization_source,
DROP COLUMN category_id,
DROP COLUMN merchant_id,
DROP COLUMN normalized_description;

DROP TABLE merchant_rules;
DROP TABLE merchants;
DROP TABLE categories;
