ALTER TABLE assets ADD COLUMN quote_refreshed_at TIMESTAMPTZ;
ALTER TABLE assets ADD COLUMN quote_error TEXT NOT NULL DEFAULT '';
-- Dated manual rates may be backfilled explicitly by the owner; never infer a historical rate.
CREATE FUNCTION native_from_eur(amount BIGINT, code TEXT, day DATE DEFAULT CURRENT_DATE) RETURNS BIGINT LANGUAGE plpgsql STABLE AS $$
DECLARE rate BIGINT; BEGIN
 IF code='EUR' THEN RETURN amount; END IF;
 SELECT rate_micro INTO rate FROM fx_history WHERE currency=code AND as_of<=day ORDER BY as_of DESC LIMIT 1;
 IF rate IS NULL THEN RAISE EXCEPTION 'missing dated FX rate for %',code USING ERRCODE='22023'; END IF;
 RETURN round(amount::numeric*power(10::numeric,currency_exponent(code))*1000000/(rate::numeric*100))::bigint;
END $$;
-- Check newly introduced domain links too, including direct SQL callers.
ALTER TABLE calendar_occurrences ADD CONSTRAINT occurrence_transaction_owner FOREIGN KEY(profile_id,transaction_id) REFERENCES transactions(profile_id,id) ON DELETE SET NULL(transaction_id);
