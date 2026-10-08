ALTER TABLE transactions ADD COLUMN linked_liability_id UUID;
ALTER TABLE transactions ADD CONSTRAINT transactions_liability_owner_fk FOREIGN KEY(profile_id,linked_liability_id) REFERENCES liabilities(profile_id,id);
CREATE TABLE push_deliveries (
 alert_id UUID NOT NULL REFERENCES custom_alerts(id) ON DELETE CASCADE,
 token TEXT NOT NULL REFERENCES push_tokens(token) ON DELETE CASCADE,
 day DATE NOT NULL,
 lease_until TIMESTAMPTZ NOT NULL,
 delivered_at TIMESTAMPTZ,
 PRIMARY KEY(alert_id,token,day)
);
