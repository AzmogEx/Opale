-- Intentionally not automatic: imported identities and dated FX are historical
-- records. Restore the pre-upgrade backup to an isolated database instead.
DO $$ BEGIN RAISE EXCEPTION '0011 rollback requires restoring a verified backup; no destructive downgrade provided'; END $$;
