DO $$ BEGIN RAISE EXCEPTION 'Restore a verified backup: removing private FX history would change financial totals'; END $$;
