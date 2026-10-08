ALTER TABLE fx_rates ADD COLUMN as_of DATE, ADD COLUMN source TEXT NOT NULL DEFAULT 'legacy reference';
UPDATE fx_rates SET as_of=updated_at::date;
ALTER TABLE fx_rates ALTER COLUMN as_of SET NOT NULL, ALTER COLUMN as_of SET DEFAULT CURRENT_DATE;
CREATE OR REPLACE FUNCTION save_fx_history() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN
 INSERT INTO fx_history(currency,as_of,rate_micro,source) VALUES(NEW.currency,NEW.as_of,NEW.rate_micro,NEW.source)
 ON CONFLICT(currency,as_of) DO UPDATE SET rate_micro=EXCLUDED.rate_micro,source=EXCLUDED.source,recorded_at=now(); RETURN NEW;
END $$;
-- Preserve the original timestamp; existing dates are never rewritten as fresh.
DROP VIEW financial_transactions;
CREATE VIEW financial_transactions AS
 SELECT t.*, a.currency, amount_eur(t.amount_cents,a.currency,t.occurred_on,t.profile_id) AS eur_cents
 FROM transactions t JOIN assets a ON a.profile_id=t.profile_id AND a.id=t.asset_id
 LEFT JOIN categories c ON c.id=t.category_id
 WHERE t.bank_status='booked' AND t.flow_kind IN ('expense_income','interest','fee')
 AND NOT (t.flow_kind='expense_income' AND COALESCE(c.name,'')='Virements' AND c.profile_id IS NULL);
