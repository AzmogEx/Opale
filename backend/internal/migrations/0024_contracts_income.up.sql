CREATE TABLE financial_contracts (
 id UUID PRIMARY KEY, profile_id UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
 revision BIGINT NOT NULL DEFAULT 1 CHECK(revision>0),
 name TEXT NOT NULL CHECK(length(trim(name)) BETWEEN 1 AND 200),
 category TEXT NOT NULL CHECK(category IN ('subscription','insurance','housing','utilities','other')),
 amount_cents BIGINT NOT NULL CHECK(amount_cents BETWEEN 1 AND 999999999999),
 currency CHAR(3) NOT NULL CHECK(currency IN ('EUR','USD','GBP','CHF','JPY','KWD','CAD','AUD')),
 frequency TEXT NOT NULL CHECK(frequency IN ('monthly','quarterly','yearly')),
 next_due_date DATE NOT NULL, asset_id UUID, calendar_rule_id UUID UNIQUE,
 merchant_key TEXT NOT NULL DEFAULT '' CHECK(length(merchant_key)<=200),
 trial_end DATE, commitment_end DATE, renewal_date DATE,
 auto_renew BOOLEAN NOT NULL DEFAULT false,
 notice_days INTEGER NOT NULL DEFAULT 30 CHECK(notice_days BETWEEN 0 AND 365),
 reminder_days INTEGER NOT NULL DEFAULT 7 CHECK(reminder_days BETWEEN 0 AND 90),
 active BOOLEAN NOT NULL DEFAULT true, note TEXT NOT NULL DEFAULT '' CHECK(length(note)<=2000),
 price_since DATE NOT NULL DEFAULT CURRENT_DATE,
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 UNIQUE(profile_id,id),
 FOREIGN KEY(profile_id,asset_id) REFERENCES assets(profile_id,id) ON DELETE SET NULL(asset_id),
 FOREIGN KEY(profile_id,calendar_rule_id) REFERENCES calendar_rules(profile_id,id) ON DELETE SET NULL(calendar_rule_id)
);
CREATE INDEX financial_contracts_owner ON financial_contracts(profile_id);
CREATE TABLE contract_prices (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(), profile_id UUID NOT NULL,
 contract_id UUID NOT NULL, amount_cents BIGINT NOT NULL CHECK(amount_cents BETWEEN 1 AND 999999999999),
 currency CHAR(3) NOT NULL, effective_on DATE NOT NULL, source TEXT NOT NULL CHECK(source IN ('declared','observed')),
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 FOREIGN KEY(profile_id,contract_id) REFERENCES financial_contracts(profile_id,id) ON DELETE CASCADE
);
CREATE TABLE contract_price_dismissals (
 profile_id UUID NOT NULL, contract_id UUID NOT NULL, transaction_id UUID NOT NULL,
 PRIMARY KEY(contract_id,transaction_id),
 FOREIGN KEY(profile_id,contract_id) REFERENCES financial_contracts(profile_id,id) ON DELETE CASCADE,
 FOREIGN KEY(profile_id,transaction_id) REFERENCES transactions(profile_id,id) ON DELETE CASCADE
);
CREATE TABLE contract_push_deliveries (
 profile_id UUID NOT NULL, contract_id UUID NOT NULL, alert_key TEXT NOT NULL, token TEXT NOT NULL,
 lease_until TIMESTAMPTZ NOT NULL, delivered_at TIMESTAMPTZ,
 PRIMARY KEY(contract_id,alert_key,token),
 FOREIGN KEY(profile_id,contract_id) REFERENCES financial_contracts(profile_id,id) ON DELETE CASCADE,
 FOREIGN KEY(token) REFERENCES push_tokens(token) ON DELETE CASCADE
);
CREATE TABLE variable_incomes (
 id UUID PRIMARY KEY, profile_id UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
 revision BIGINT NOT NULL DEFAULT 1 CHECK(revision>0),
 name TEXT NOT NULL CHECK(length(trim(name)) BETWEEN 1 AND 200),
 kind TEXT NOT NULL CHECK(kind IN ('freelance','bonus','rental','dividend','other')),
 currency CHAR(3) NOT NULL CHECK(currency IN ('EUR','USD','GBP','CHF','JPY','KWD','CAD','AUD')),
 low_cents BIGINT NOT NULL CHECK(low_cents>=0), usual_cents BIGINT NOT NULL CHECK(usual_cents>0),
 high_cents BIGINT NOT NULL CHECK(high_cents<=999999999999),
 CHECK(low_cents<=usual_cents AND usual_cents<=high_cents),
 frequency TEXT NOT NULL CHECK(frequency IN ('once','monthly','quarterly','yearly')),
 next_date DATE NOT NULL, forecast TEXT NOT NULL CHECK(forecast IN ('off','prudent','usual')),
 asset_id UUID, calendar_rule_id UUID UNIQUE, merchant_key TEXT NOT NULL DEFAULT '' CHECK(length(merchant_key)<=200),
 active BOOLEAN NOT NULL DEFAULT true, note TEXT NOT NULL DEFAULT '' CHECK(length(note)<=2000),
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 FOREIGN KEY(profile_id,asset_id) REFERENCES assets(profile_id,id) ON DELETE SET NULL(asset_id),
 FOREIGN KEY(profile_id,calendar_rule_id) REFERENCES calendar_rules(profile_id,id) ON DELETE SET NULL(calendar_rule_id)
);
CREATE INDEX variable_incomes_owner ON variable_incomes(profile_id);
-- Preserve subscriptions already declared through onboarding; no new cash flow.
INSERT INTO financial_contracts(id,profile_id,name,category,amount_cents,currency,frequency,next_due_date,asset_id,calendar_rule_id,merchant_key,active,price_since)
 SELECT gen_random_uuid(),r.profile_id,r.label,'subscription',-r.amount_cents,a.currency,r.frequency,r.starts_on,r.asset_id,r.id,r.merchant_key,r.active,CURRENT_DATE
 FROM profile_onboarding p CROSS JOIN LATERAL jsonb_array_elements_text(p.result->'subscription_rule_ids') j(id)
 JOIN calendar_rules r ON r.id=j.id::uuid AND r.profile_id=p.profile_id
 JOIN assets a ON a.id=r.asset_id AND a.profile_id=r.profile_id
 WHERE p.status='completed' AND r.amount_cents<0 AND r.amount_cents>=-999999999999 AND r.frequency IN ('monthly','quarterly','yearly');
INSERT INTO contract_prices(profile_id,contract_id,amount_cents,currency,effective_on,source)
 SELECT profile_id,id,amount_cents,currency,price_since,'declared' FROM financial_contracts;
