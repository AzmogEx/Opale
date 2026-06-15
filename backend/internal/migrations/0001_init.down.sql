-- Annulation de la migration 0001.
DROP TABLE IF EXISTS valuations;
DROP TABLE IF EXISTS liabilities;
DROP TABLE IF EXISTS assets;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS profiles;
DROP FUNCTION IF EXISTS set_updated_at();
