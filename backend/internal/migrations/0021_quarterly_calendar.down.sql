-- Un retour arrière ne supprime ni ne transforme les séries existantes.
LOCK TABLE calendar_rules IN ACCESS EXCLUSIVE MODE;
DO $$ BEGIN
    IF EXISTS (SELECT 1 FROM calendar_rules WHERE frequency='quarterly') THEN
        RAISE EXCEPTION 'Retour arrière impossible : des règles trimestrielles existent. Les conserver ou les modifier explicitement avant de réessayer.';
    END IF;
END $$;
ALTER TABLE calendar_rules DROP CONSTRAINT calendar_rules_frequency_check;
ALTER TABLE calendar_rules ADD CONSTRAINT calendar_rules_frequency_check
    CHECK (frequency IN ('once','weekly','monthly','yearly'));
