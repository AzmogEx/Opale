ALTER TABLE bank_links ADD COLUMN sync_status TEXT NOT NULL DEFAULT 'pending_consent';
ALTER TABLE bank_links ADD COLUMN next_sync_at TIMESTAMPTZ NOT NULL DEFAULT now();
ALTER TABLE bank_links ADD COLUMN lease_until TIMESTAMPTZ;
ALTER TABLE bank_links ADD COLUMN last_error TEXT NOT NULL DEFAULT '';
ALTER TABLE bank_links ADD COLUMN attempts INT NOT NULL DEFAULT 0;
ALTER TABLE bank_links ADD CONSTRAINT bank_links_owner_identity UNIQUE(profile_id,id);

CREATE TABLE bank_accounts (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
 profile_id UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
 link_id UUID NOT NULL,
 provider_account_id TEXT NOT NULL,
 asset_id UUID,
 currency TEXT NOT NULL DEFAULT '',
 name TEXT NOT NULL DEFAULT '',
 status TEXT NOT NULL DEFAULT 'needs_mapping',
 last_synced_at TIMESTAMPTZ,
 last_error TEXT NOT NULL DEFAULT '',
 balance_cents BIGINT,
 balance_date DATE,
 balance_type TEXT NOT NULL DEFAULT '',
 balance_reconciled BOOLEAN NOT NULL DEFAULT false,
 FOREIGN KEY(profile_id,link_id) REFERENCES bank_links(profile_id,id) ON DELETE CASCADE,
 FOREIGN KEY(profile_id,asset_id) REFERENCES assets(profile_id,id) ON DELETE RESTRICT,
 UNIQUE(profile_id,provider_account_id),
 UNIQUE(profile_id,id)
);
CREATE UNIQUE INDEX bank_account_asset_unique ON bank_accounts(profile_id,asset_id) WHERE asset_id IS NOT NULL;
CREATE TABLE bank_pending (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
 profile_id UUID NOT NULL,
 bank_account_id UUID NOT NULL,
 amount_cents BIGINT NOT NULL,
 occurred_on DATE,
 label TEXT NOT NULL,
 FOREIGN KEY(profile_id,bank_account_id) REFERENCES bank_accounts(profile_id,id) ON DELETE CASCADE
);
