-- Refuse downgrade while negative balances exist; never discard them.
ALTER TABLE valuations ADD CONSTRAINT valuations_value_cents_check CHECK (value_cents >= 0);
DROP TRIGGER trg_valuation_overdraft ON valuations;
DROP FUNCTION validate_valuation_overdraft();
DROP TRIGGER trg_asset_overdraft ON assets;
DROP FUNCTION preserve_checking_overdraft();
ALTER TABLE valuations DROP CONSTRAINT valuations_nonnegative_liability;
