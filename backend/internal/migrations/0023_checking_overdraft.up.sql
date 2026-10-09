-- A checking account may start in overdraft. Represent the observed balance
-- directly rather than fabricating a booked expense to bypass positivity.
ALTER TABLE valuations DROP CONSTRAINT valuations_value_cents_check;
ALTER TABLE valuations ADD CONSTRAINT valuations_nonnegative_liability
    CHECK (value_cents >= 0 OR asset_id IS NOT NULL);

CREATE OR REPLACE FUNCTION validate_valuation_overdraft() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.value_cents < 0 THEN
        -- SHARE conflicts with a concurrent kind update. If the update wins,
        -- PostgreSQL rechecks the kind after the lock is acquired.
        PERFORM 1 FROM assets WHERE id=NEW.asset_id AND profile_id=NEW.profile_id AND kind='checking' FOR SHARE;
        IF NOT FOUND THEN
            RAISE EXCEPTION 'Only checking accounts accept a negative closing balance' USING ERRCODE='23514';
        END IF;
    END IF;
    RETURN NEW;
END $$;
CREATE TRIGGER trg_valuation_overdraft BEFORE INSERT OR UPDATE ON valuations
    FOR EACH ROW EXECUTE FUNCTION validate_valuation_overdraft();

CREATE FUNCTION preserve_checking_overdraft() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF NEW.kind <> 'checking' AND EXISTS(SELECT 1 FROM valuations WHERE asset_id=NEW.id AND value_cents<0) THEN
        RAISE EXCEPTION 'Negative balances require a checking account' USING ERRCODE='23514';
    END IF;
    RETURN NEW;
END $$;
CREATE TRIGGER trg_asset_overdraft BEFORE UPDATE OF kind ON assets
    FOR EACH ROW EXECUTE FUNCTION preserve_checking_overdraft();
