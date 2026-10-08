ALTER TABLE calendar_occurrences DROP CONSTRAINT occurrence_transaction_owner;
DROP FUNCTION native_from_eur;
ALTER TABLE assets DROP COLUMN quote_refreshed_at,DROP COLUMN quote_error;
