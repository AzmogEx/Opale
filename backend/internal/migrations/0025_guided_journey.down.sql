DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM profile_journey) THEN
  RAISE EXCEPTION 'Export or reset guided journey data before downgrading';
 END IF;
END $$;
DROP TABLE profile_journey;
