-- Domaines approfondis : toutes les données restent rattachées à leur profil.
ALTER TABLE goals ADD COLUMN monthly_savings_cents BIGINT NOT NULL DEFAULT 0 CHECK (monthly_savings_cents >= 0);

CREATE TABLE calendar_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    asset_id UUID NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
    label TEXT NOT NULL CHECK (length(trim(label)) BETWEEN 1 AND 200),
    amount_cents BIGINT NOT NULL CHECK (amount_cents <> 0),
    starts_on DATE NOT NULL,
    frequency TEXT NOT NULL CHECK (frequency IN ('once','weekly','monthly','yearly')),
    ends_on DATE,
    active BOOLEAN NOT NULL DEFAULT true,
    merchant_key TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (ends_on IS NULL OR ends_on >= starts_on),
    FOREIGN KEY (profile_id, asset_id) REFERENCES assets(profile_id,id) ON DELETE CASCADE,
    UNIQUE (profile_id,id)
);
CREATE TABLE calendar_occurrences (
    profile_id UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    rule_id UUID NOT NULL,
    occurs_on DATE NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('excluded','realized','planned')),
    amount_cents BIGINT,
    transaction_id UUID REFERENCES transactions(id) ON DELETE SET NULL,
    PRIMARY KEY (rule_id,occurs_on),
    FOREIGN KEY (profile_id,rule_id) REFERENCES calendar_rules(profile_id,id) ON DELETE CASCADE
);
CREATE TABLE recurring_exclusions (
    profile_id UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    merchant_key TEXT NOT NULL,
    PRIMARY KEY (profile_id,merchant_key)
);
CREATE UNIQUE INDEX calendar_realized_transaction ON calendar_occurrences(transaction_id) WHERE transaction_id IS NOT NULL;
ALTER TABLE calendar_occurrences ADD CONSTRAINT calendar_transaction_owner FOREIGN KEY(profile_id,transaction_id) REFERENCES transactions(profile_id,id) ON DELETE SET NULL (transaction_id);

CREATE TABLE investment_flows (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    asset_id UUID NOT NULL,
    kind TEXT NOT NULL CHECK (kind IN ('contribution','withdrawal','distribution','fee')),
    amount_cents BIGINT NOT NULL CHECK (amount_cents > 0),
    occurred_on DATE NOT NULL,
    note TEXT NOT NULL DEFAULT '',
    FOREIGN KEY (profile_id,asset_id) REFERENCES assets(profile_id,id) ON DELETE CASCADE
);
CREATE INDEX investment_flows_asset_date ON investment_flows(profile_id,asset_id,occurred_on);
CREATE TABLE investment_coverage (
    profile_id UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    asset_id UUID NOT NULL,
    complete BOOLEAN NOT NULL DEFAULT false,
    PRIMARY KEY (profile_id,asset_id),
    FOREIGN KEY (profile_id,asset_id) REFERENCES assets(profile_id,id) ON DELETE CASCADE
);

ALTER TABLE company_details ADD COLUMN cca_asset_id UUID REFERENCES assets(id) ON DELETE RESTRICT;
CREATE UNIQUE INDEX company_cca_unique ON company_details(cca_asset_id) WHERE cca_asset_id IS NOT NULL;
ALTER TABLE company_details ADD CONSTRAINT company_cca_distinct CHECK (cca_asset_id IS NULL OR cca_asset_id <> asset_id);
ALTER TABLE company_details ADD CONSTRAINT company_cca_profile FOREIGN KEY (profile_id,cca_asset_id) REFERENCES assets(profile_id,id) ON DELETE RESTRICT;
-- Les CCA historiques constituent des créances distinctes. La date est celle
-- de leur dernière saisie connue, sans inventer un historique antérieur.
DO $$ DECLARE c RECORD; linked UUID; BEGIN
 FOR c IN SELECT cd.*, a.name,a.currency FROM company_details cd JOIN assets a ON a.id=cd.asset_id WHERE cd.cca_cents>0 LOOP
  INSERT INTO assets(profile_id,name,kind,currency,note) VALUES(c.profile_id,'CCA — '||c.name,'other',c.currency,'Créance de compte courant associé') RETURNING id INTO linked;
  INSERT INTO valuations(profile_id,asset_id,value_cents,as_of,note) VALUES(c.profile_id,linked,c.cca_cents,c.updated_at::date,'Reprise de la dernière valeur de CCA connue');
  UPDATE company_details SET cca_asset_id=linked WHERE asset_id=c.asset_id;
 END LOOP;
END $$;

CREATE TABLE beneficiaries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    contact_id UUID NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    document_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    share_bps INT NOT NULL CHECK (share_bps BETWEEN 1 AND 10000),
    note TEXT NOT NULL DEFAULT '',
    UNIQUE (profile_id,contact_id,document_id)
);
ALTER TABLE contacts ADD CONSTRAINT contacts_domain_owner UNIQUE(profile_id,id);
ALTER TABLE beneficiaries ADD CONSTRAINT beneficiary_contact_owner FOREIGN KEY(profile_id,contact_id) REFERENCES contacts(profile_id,id) ON DELETE CASCADE;
ALTER TABLE beneficiaries ADD CONSTRAINT beneficiary_document_owner FOREIGN KEY(profile_id,document_id) REFERENCES documents(profile_id,id) ON DELETE CASCADE;
CREATE TABLE emergency_grants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_profile_id UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    recipient_profile_id UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    asset_ids UUID[] NOT NULL DEFAULT '{}',
    document_ids UUID[] NOT NULL DEFAULT '{}',
    active BOOLEAN NOT NULL DEFAULT false,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK(owner_profile_id<>recipient_profile_id)
);
CREATE INDEX emergency_recipient ON emergency_grants(recipient_profile_id);
