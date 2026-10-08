DROP TABLE push_tokens;
DROP TABLE custom_alerts;
DROP TABLE allocation_targets;
DROP TABLE monthly_snapshots;
ALTER TABLE assets DROP COLUMN quote_quantity_micro, DROP COLUMN quote_symbol;
