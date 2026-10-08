-- Idempotent reinforcement for installations that applied early domain builds.
CREATE UNIQUE INDEX IF NOT EXISTS calendar_realized_transaction ON calendar_occurrences(transaction_id) WHERE transaction_id IS NOT NULL;
DO $$ BEGIN
 IF NOT EXISTS(SELECT 1 FROM pg_constraint WHERE conname='calendar_transaction_owner') THEN
  ALTER TABLE calendar_occurrences ADD CONSTRAINT calendar_transaction_owner FOREIGN KEY(profile_id,transaction_id) REFERENCES transactions(profile_id,id) ON DELETE SET NULL (transaction_id);
 END IF;
 IF NOT EXISTS(SELECT 1 FROM pg_constraint WHERE conname='contacts_domain_owner') THEN
  ALTER TABLE contacts ADD CONSTRAINT contacts_domain_owner UNIQUE(profile_id,id);
 END IF;
 IF NOT EXISTS(SELECT 1 FROM pg_constraint WHERE conname='beneficiary_contact_owner') THEN
  ALTER TABLE beneficiaries ADD CONSTRAINT beneficiary_contact_owner FOREIGN KEY(profile_id,contact_id) REFERENCES contacts(profile_id,id) ON DELETE CASCADE;
 END IF;
 IF NOT EXISTS(SELECT 1 FROM pg_constraint WHERE conname='beneficiary_document_owner') THEN
  ALTER TABLE beneficiaries ADD CONSTRAINT beneficiary_document_owner FOREIGN KEY(profile_id,document_id) REFERENCES documents(profile_id,id) ON DELETE CASCADE;
 END IF;
END $$;
-- Providers may omit dates on provisional entries; never invent one.
ALTER TABLE bank_pending ALTER COLUMN occurred_on DROP NOT NULL;
