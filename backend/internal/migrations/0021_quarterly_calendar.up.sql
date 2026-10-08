-- Préserver la fréquence trimestrielle issue du détecteur de récurrences.
ALTER TABLE calendar_rules DROP CONSTRAINT calendar_rules_frequency_check;
ALTER TABLE calendar_rules ADD CONSTRAINT calendar_rules_frequency_check
    CHECK (frequency IN ('once','weekly','monthly','quarterly','yearly'));
