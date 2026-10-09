package store

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
)

// Persistent, per-occurrence delivery. Ownership is checked again at claim and completion.
func (s *Store) ClaimContractPush(ctx context.Context, profile, contract, key, token string) (bool, error) {
	var ok bool
	e := s.pool.QueryRow(ctx, `INSERT INTO contract_push_deliveries(profile_id,contract_id,alert_key,token,lease_until)
 SELECT c.profile_id,c.id,$3,p.token,now()+interval '5 minutes' FROM financial_contracts c
 JOIN push_tokens p ON p.profile_id=c.profile_id WHERE c.profile_id=$1 AND c.id=$2 AND p.token=$4 AND c.active
 ON CONFLICT(contract_id,alert_key,token) DO UPDATE SET lease_until=EXCLUDED.lease_until
 WHERE contract_push_deliveries.delivered_at IS NULL AND contract_push_deliveries.lease_until<now() RETURNING true`, profile, contract, key, token).Scan(&ok)
	if errors.Is(e, pgx.ErrNoRows) {
		return false, nil
	}
	return ok, e
}
func (s *Store) CompleteContractPush(ctx context.Context, profile, contract, key, token string) error {
	_, e := s.pool.Exec(ctx, `UPDATE contract_push_deliveries d SET delivered_at=now() FROM push_tokens p
 WHERE d.profile_id=$1 AND d.contract_id=$2 AND d.alert_key=$3 AND d.token=$4 AND p.token=d.token AND p.profile_id=d.profile_id`, profile, contract, key, token)
	return e
}
