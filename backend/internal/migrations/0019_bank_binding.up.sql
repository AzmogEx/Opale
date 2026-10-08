-- Retain the account-to-ledger identity after disconnect/reconnect. Imported
-- history cannot safely be silently moved into another asset.
CREATE TABLE bank_account_bindings (
 profile_id UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
 provider_account_id TEXT NOT NULL,
 asset_id UUID NOT NULL,
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 PRIMARY KEY(profile_id,provider_account_id),
 FOREIGN KEY(profile_id,asset_id) REFERENCES assets(profile_id,id) ON DELETE RESTRICT
);
INSERT INTO bank_account_bindings(profile_id,provider_account_id,asset_id)
 SELECT profile_id,provider_account_id,asset_id FROM bank_accounts WHERE asset_id IS NOT NULL AND last_synced_at IS NOT NULL;
