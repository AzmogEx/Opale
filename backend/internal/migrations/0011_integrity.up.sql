-- The old clients recorded hundredths for every currency. Reject ambiguous
-- pre-existing non-two-decimal holdings for an explicit, backed-up conversion.
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM assets WHERE currency IN ('JPY','KRW','CLP','VND','XAF','XOF','XPF','BIF','DJF','GNF','ISK','KMF','PYG','RWF','UGX','VUV','BHD','IQD','JOD','KWD','LYD','OMR','TND','CLF','UYW') UNION ALL SELECT 1 FROM liabilities WHERE currency IN ('JPY','KRW','CLP','VND','XAF','XOF','XPF','BIF','DJF','GNF','ISK','KMF','PYG','RWF','UGX','VUV','BHD','IQD','JOD','KWD','LYD','OMR','TND','CLF','UYW')) THEN
 RAISE EXCEPTION 'Legacy currency units require review before migration 0011; see docs/EXPLOITATION.md';
 END IF;
END $$;
-- No silent repair: legacy cross-profile references stop this transaction.
ALTER TABLE assets ADD CONSTRAINT assets_owner_key UNIQUE(profile_id,id);
ALTER TABLE liabilities ADD CONSTRAINT liabilities_owner_key UNIQUE(profile_id,id);
ALTER TABLE transactions ADD CONSTRAINT transactions_owner_key UNIQUE(profile_id,id);
ALTER TABLE documents ADD CONSTRAINT documents_owner_key UNIQUE(profile_id,id);
ALTER TABLE assets ADD COLUMN archived_at DATE;
ALTER TABLE liabilities ADD COLUMN archived_at DATE;
UPDATE assets SET archived_at = updated_at::date WHERE archived;
UPDATE liabilities SET archived_at = updated_at::date WHERE archived;
CREATE FUNCTION record_archive_date() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  IF NEW.archived AND NOT OLD.archived THEN NEW.archived_at = CURRENT_DATE;
  ELSIF NOT NEW.archived THEN NEW.archived_at = NULL; END IF;
  RETURN NEW;
END $$;
CREATE TRIGGER asset_archive_date BEFORE UPDATE ON assets FOR EACH ROW EXECUTE FUNCTION record_archive_date();
CREATE TRIGGER liability_archive_date BEFORE UPDATE ON liabilities FOR EACH ROW EXECUTE FUNCTION record_archive_date();
ALTER TABLE transactions ADD CONSTRAINT transaction_asset_owner FOREIGN KEY(profile_id,asset_id) REFERENCES assets(profile_id,id) ON DELETE CASCADE;
ALTER TABLE valuations ADD CONSTRAINT valuation_asset_owner FOREIGN KEY(profile_id,asset_id) REFERENCES assets(profile_id,id) ON DELETE CASCADE;
ALTER TABLE valuations ADD CONSTRAINT valuation_liability_owner FOREIGN KEY(profile_id,liability_id) REFERENCES liabilities(profile_id,id) ON DELETE CASCADE;
ALTER TABLE goals ADD CONSTRAINT goal_asset_owner FOREIGN KEY(profile_id,asset_id) REFERENCES assets(profile_id,id) ON DELETE SET NULL(asset_id);
ALTER TABLE documents ADD CONSTRAINT document_asset_owner FOREIGN KEY(profile_id,asset_id) REFERENCES assets(profile_id,id) ON DELETE SET NULL(asset_id);
ALTER TABLE bank_links ADD CONSTRAINT bank_asset_owner FOREIGN KEY(profile_id,asset_id) REFERENCES assets(profile_id,id) ON DELETE CASCADE;
ALTER TABLE property_details ADD CONSTRAINT property_asset_owner FOREIGN KEY(profile_id,asset_id) REFERENCES assets(profile_id,id) ON DELETE CASCADE;
ALTER TABLE property_details ADD CONSTRAINT property_liability_owner FOREIGN KEY(profile_id,liability_id) REFERENCES liabilities(profile_id,id) ON DELETE SET NULL(liability_id);
ALTER TABLE object_details ADD CONSTRAINT object_asset_owner FOREIGN KEY(profile_id,asset_id) REFERENCES assets(profile_id,id) ON DELETE CASCADE;
ALTER TABLE company_details ADD CONSTRAINT company_asset_owner FOREIGN KEY(profile_id,asset_id) REFERENCES assets(profile_id,id) ON DELETE CASCADE;
-- Global categories are readable by all; private categories only by their owner.
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM transactions t JOIN categories c ON c.id=t.category_id WHERE c.profile_id IS NOT NULL AND c.profile_id<>t.profile_id)
 OR EXISTS(SELECT 1 FROM envelopes t JOIN categories c ON c.id=t.category_id WHERE c.profile_id IS NOT NULL AND c.profile_id<>t.profile_id)
 OR EXISTS(SELECT 1 FROM merchant_rules t JOIN categories c ON c.id=t.category_id WHERE c.profile_id IS NOT NULL AND c.profile_id<>t.profile_id)
 OR EXISTS(SELECT 1 FROM categories t JOIN categories c ON c.id=t.parent_id WHERE c.profile_id IS NOT NULL AND c.profile_id IS DISTINCT FROM t.profile_id)
 THEN RAISE EXCEPTION 'Migration 0011: category ownership inconsistency; inspect affected relations with an administrator, no data changed'; END IF;
END $$;
CREATE FUNCTION check_category_owner() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE category UUID;
BEGIN
 IF TG_TABLE_NAME='categories' THEN category=NEW.parent_id; ELSE category=NEW.category_id; END IF;
 IF category IS NOT NULL AND NOT EXISTS(SELECT 1 FROM categories c WHERE c.id=category AND (c.profile_id IS NULL OR c.profile_id=NEW.profile_id))
 THEN RAISE EXCEPTION 'referenced resource unavailable' USING ERRCODE='23503'; END IF;
 RETURN NEW;
END $$;
CREATE TRIGGER transactions_category_owner BEFORE INSERT OR UPDATE ON transactions FOR EACH ROW EXECUTE FUNCTION check_category_owner();
CREATE TRIGGER envelopes_category_owner BEFORE INSERT OR UPDATE ON envelopes FOR EACH ROW EXECUTE FUNCTION check_category_owner();
CREATE TRIGGER merchant_rules_category_owner BEFORE INSERT OR UPDATE ON merchant_rules FOR EACH ROW EXECUTE FUNCTION check_category_owner();
CREATE TRIGGER categories_parent_owner BEFORE INSERT OR UPDATE ON categories FOR EACH ROW EXECUTE FUNCTION check_category_owner();
-- A membership FK closes the check/use race with member removal.
ALTER TABLE transactions ADD CONSTRAINT transaction_space_member FOREIGN KEY(space_id,profile_id) REFERENCES space_members(space_id,profile_id) ON DELETE SET NULL(space_id);
ALTER TABLE profiles ADD COLUMN is_demo BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE profiles ADD COLUMN demo_expires_at TIMESTAMPTZ;
ALTER TABLE profiles ADD COLUMN demo_ready BOOLEAN NOT NULL DEFAULT true;
ALTER TABLE transactions ADD COLUMN flow_kind TEXT NOT NULL DEFAULT 'expense_income'
 CHECK(flow_kind IN ('expense_income','internal_transfer','investment_contribution','investment_withdrawal','loan_principal','interest','fee'));
ALTER TABLE transactions ADD COLUMN transfer_id UUID;
ALTER TABLE transactions ADD COLUMN source_id TEXT NOT NULL DEFAULT '';
ALTER TABLE transactions ADD COLUMN bank_status TEXT NOT NULL DEFAULT 'booked' CHECK(bank_status IN ('pending','booked'));
-- Persistent identity survives splitting/deleting the visible transaction.
CREATE TABLE imported_operations (
 profile_id UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
 asset_id UUID NOT NULL,
 source_key TEXT NOT NULL,
 transaction_id UUID REFERENCES transactions(id) ON DELETE SET NULL,
 original_amount BIGINT NOT NULL,
 occurred_on DATE NOT NULL,
 raw_label TEXT NOT NULL,
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 PRIMARY KEY(profile_id,asset_id,source_key),
 FOREIGN KEY(profile_id,asset_id) REFERENCES assets(profile_id,id) ON DELETE CASCADE
);
CREATE TABLE fx_history (
 currency TEXT NOT NULL, as_of DATE NOT NULL, rate_micro BIGINT NOT NULL CHECK(rate_micro>0),
 source TEXT NOT NULL DEFAULT 'manual', recorded_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 PRIMARY KEY(currency,as_of)
);
INSERT INTO fx_history(currency,as_of,rate_micro) SELECT currency,updated_at::date,rate_micro FROM fx_rates;
CREATE FUNCTION save_fx_history() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN
 INSERT INTO fx_history(currency,as_of,rate_micro) VALUES(NEW.currency,NEW.updated_at::date,NEW.rate_micro)
 ON CONFLICT(currency,as_of) DO UPDATE SET rate_micro=EXCLUDED.rate_micro,recorded_at=now(); RETURN NEW;
END $$;
CREATE TRIGGER fx_history_record AFTER INSERT OR UPDATE ON fx_rates FOR EACH ROW EXECUTE FUNCTION save_fx_history();
-- Exact minor-unit conversion: unsupported currencies fail explicitly.
CREATE FUNCTION currency_exponent(code TEXT) RETURNS integer LANGUAGE plpgsql IMMUTABLE AS $$ BEGIN
 IF code IN ('JPY','KRW','CLP','VND','XAF','XOF','XPF','BIF','DJF','GNF','ISK','KMF','PYG','RWF','UGX','VUV') THEN RETURN 0; END IF;
 IF code IN ('BHD','IQD','JOD','KWD','LYD','OMR','TND') THEN RETURN 3; END IF;
 IF code IN ('CLF','UYW') THEN RETURN 4; END IF;
 IF code IN ('EUR','USD','GBP','CHF','CAD','AUD','NZD','CNY','HKD','SGD','SEK','NOK','DKK','PLN','CZK','HUF','RON','BGN','TRY','INR','BRL','MXN','ZAR','AED','SAR','ILS','THB','MYR','IDR','PHP','TWD','MAD','DZD','EGP','UAH','RSD','RUB','GEL','ARS','COP','PEN','UYU','BOB','CRC','DOP','GTQ','HNL','NIO','PAB','BND','PKR','BDT','LKR','NPR','MUR','MGA','KES','TZS','NGN','GHS','BWP','ZMW','ALL','BAM','MDL','MKD','AZN','KZT','QAR','SCR') THEN RETURN 2; END IF;
 RAISE EXCEPTION 'unsupported currency' USING ERRCODE='22023';
END $$;
CREATE FUNCTION amount_eur(amount BIGINT, code TEXT, day DATE DEFAULT CURRENT_DATE) RETURNS BIGINT LANGUAGE plpgsql STABLE AS $$
DECLARE rate BIGINT; result NUMERIC;
BEGIN
 IF code='EUR' THEN RETURN amount; END IF;
 SELECT rate_micro INTO rate FROM fx_history WHERE currency=code AND as_of<=day ORDER BY as_of DESC LIMIT 1;
 IF rate IS NULL THEN RAISE EXCEPTION 'missing dated FX rate for %',code USING ERRCODE='22023'; END IF;
 result=round(amount::numeric * rate * 100 / (power(10::numeric,currency_exponent(code))*1000000));
 RETURN result::bigint;
END $$;
CREATE FUNCTION current_asset_value(owner UUID, asset UUID, day DATE DEFAULT CURRENT_DATE) RETURNS BIGINT LANGUAGE sql STABLE AS $$
 SELECT COALESCE(v.value_cents,0)+CASE WHEN a.kind IN ('checking','savings') THEN COALESCE((
 SELECT SUM(t.amount_cents) FROM transactions t WHERE t.profile_id=owner AND t.asset_id=asset
 AND t.bank_status='booked' AND t.occurred_on>COALESCE(v.as_of,'0001-01-01'::date) AND t.occurred_on<=day),0) ELSE 0 END
 FROM assets a LEFT JOIN LATERAL (SELECT value_cents,as_of FROM valuations WHERE profile_id=owner AND asset_id=asset AND as_of<=day ORDER BY as_of DESC,created_at DESC,id DESC LIMIT 1) v ON true
 WHERE a.id=asset AND a.profile_id=owner;
$$;
CREATE VIEW financial_transactions AS
 SELECT t.*, a.currency, amount_eur(t.amount_cents,a.currency,t.occurred_on) AS eur_cents
 FROM transactions t JOIN assets a ON a.id=t.asset_id AND a.profile_id=t.profile_id
 LEFT JOIN categories c ON c.id=t.category_id
 WHERE t.bank_status='booked' AND t.flow_kind IN ('expense_income','interest','fee')
 AND NOT (t.flow_kind='expense_income' AND COALESCE(c.name,'')='Virements' AND c.profile_id IS NULL);

CREATE FUNCTION validate_currency() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN
 PERFORM currency_exponent(NEW.currency); RETURN NEW; END $$;
CREATE TRIGGER asset_currency BEFORE INSERT OR UPDATE ON assets FOR EACH ROW EXECUTE FUNCTION validate_currency();
CREATE TRIGGER liability_currency BEFORE INSERT OR UPDATE ON liabilities FOR EACH ROW EXECUTE FUNCTION validate_currency();

ALTER TABLE assets ADD COLUMN client_request_id UUID;
ALTER TABLE liabilities ADD COLUMN client_request_id UUID;
CREATE UNIQUE INDEX asset_request_once ON assets(profile_id,client_request_id);
CREATE UNIQUE INDEX liability_request_once ON liabilities(profile_id,client_request_id);
