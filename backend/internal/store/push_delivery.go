package store

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
)

// ClaimPush serialises deliveries across processes and restarts. Failed attempts
// retry after a five-minute lease; a delivered notification is not repeated.
func (s *Store) ClaimPush(ctx context.Context, profile, alert, token string) (bool, error) {
	var ok bool
	e := s.pool.QueryRow(ctx, `INSERT INTO push_deliveries(alert_id,token,day,lease_until) SELECT a.id,p.token,CURRENT_DATE,now()+interval '5 minutes' FROM custom_alerts a JOIN push_tokens p ON p.profile_id=a.profile_id WHERE a.id=$1 AND a.profile_id=$2 AND p.token=$3 AND a.enabled ON CONFLICT(alert_id,token,day) DO UPDATE SET lease_until=EXCLUDED.lease_until WHERE push_deliveries.delivered_at IS NULL AND push_deliveries.lease_until<now() RETURNING true`, alert, profile, token).Scan(&ok)
	if errors.Is(e, pgx.ErrNoRows) {
		return false, nil
	}
	return ok, e
}
func (s *Store) CompletePush(ctx context.Context, profile, alert, token string) error {
	_, e := s.pool.Exec(ctx, `UPDATE push_deliveries d SET delivered_at=now() FROM push_tokens p WHERE d.alert_id=$1 AND d.token=$2 AND d.day=CURRENT_DATE AND p.token=d.token AND p.profile_id=$3`, alert, token, profile)
	return e
}
