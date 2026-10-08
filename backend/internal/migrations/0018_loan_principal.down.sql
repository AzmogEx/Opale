DO $$ BEGIN RAISE EXCEPTION 'Restore a verified backup before reverting principal accounting'; END $$;
