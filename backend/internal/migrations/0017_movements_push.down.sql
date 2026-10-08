DO $$ BEGIN RAISE EXCEPTION 'Restore a verified backup before reverting accounting movements'; END $$;
