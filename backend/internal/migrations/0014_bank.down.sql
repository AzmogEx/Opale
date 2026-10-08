DROP TABLE bank_pending,bank_accounts;
ALTER TABLE bank_links DROP COLUMN sync_status,DROP COLUMN next_sync_at,DROP COLUMN lease_until,DROP COLUMN last_error,DROP COLUMN attempts;
ALTER TABLE bank_links DROP CONSTRAINT bank_links_owner_identity;
