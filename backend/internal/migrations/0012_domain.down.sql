-- Une restauration de sauvegarde est nécessaire pour récupérer les données
-- propres aux modules retirés. Les actifs CCA créés restent conservés.
DROP TABLE emergency_grants, beneficiaries, investment_coverage, investment_flows, recurring_exclusions, calendar_occurrences, calendar_rules;
ALTER TABLE company_details DROP COLUMN cca_asset_id;
ALTER TABLE goals DROP COLUMN monthly_savings_cents;
ALTER TABLE contacts DROP CONSTRAINT contacts_domain_owner;
