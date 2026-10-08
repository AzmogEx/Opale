CREATE FUNCTION current_liability_value(owner UUID, liability UUID, day DATE DEFAULT CURRENT_DATE) RETURNS BIGINT LANGUAGE plpgsql STABLE AS $$
DECLARE balance BIGINT; boundary DATE; payments NUMERIC;
BEGIN
 SELECT value_cents,as_of INTO balance,boundary FROM valuations WHERE profile_id=owner AND liability_id=liability AND as_of<=day ORDER BY as_of DESC,created_at DESC,id DESC LIMIT 1;
 SELECT COALESCE(SUM(amount_cents),0) INTO payments FROM transactions WHERE profile_id=owner AND linked_liability_id=liability AND flow_kind='loan_principal' AND bank_status='booked' AND occurred_on>COALESCE(boundary,'-infinity'::date) AND occurred_on<=day;
 balance:=(COALESCE(balance,0)::numeric+payments)::bigint;
 IF balance<0 THEN RAISE EXCEPTION 'Principal repayments exceed recorded debt balance' USING ERRCODE='22023'; END IF;
 RETURN balance;
END $$;
CREATE FUNCTION check_loan_movement() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF NEW.linked_liability_id IS NOT NULL AND NOT EXISTS(SELECT 1 FROM liabilities l JOIN assets a ON a.profile_id=l.profile_id WHERE l.profile_id=NEW.profile_id AND l.id=NEW.linked_liability_id AND a.id=NEW.asset_id AND a.currency=l.currency AND NEW.flow_kind='loan_principal' AND NEW.amount_cents<0) THEN
 RAISE EXCEPTION 'Principal movement requires an owned debt in the account currency and a negative amount' USING ERRCODE='23514';
 END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER loan_movement_guard BEFORE INSERT OR UPDATE ON transactions FOR EACH ROW EXECUTE FUNCTION check_loan_movement();
