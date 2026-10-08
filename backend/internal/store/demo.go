package store

import (
	"context"
	"fmt"
	"github.com/opale-app/opale/internal/auth"
	"time"
)

// CreateDemo creates an isolated, hidden, expiring profile and all its data atomically.
func (s *Store) CreateDemo(ctx context.Context, seed func(*Store, string) error) (Profile, string, string, time.Time, error) {
	var p Profile
	pin, err := auth.NewToken()
	if err != nil {
		return p, "", "", time.Time{}, err
	}
	pin = pin[:16]
	token, err := auth.NewToken()
	if err != nil {
		return p, "", "", time.Time{}, err
	}
	hash, err := auth.HashPIN(pin)
	if err != nil {
		return p, "", "", time.Time{}, err
	}
	expires := time.Now().Add(24 * time.Hour)
	err = s.Atomic(ctx, func(st *Store) error {
		if _, err := st.pool.Exec(ctx, "SELECT pg_advisory_xact_lock(74201902)"); err != nil {
			return err
		}
		var recent int
		if err := st.pool.QueryRow(ctx, "SELECT count(*) FROM profiles WHERE is_demo AND created_at>now()-interval '1 minute'").Scan(&recent); err != nil {
			return err
		}
		if recent >= 5 {
			return fmt.Errorf("demo creation rate limited")
		}
		var err error
		p, err = st.CreateProfile(ctx, "Démo", hash, "N1")
		if err != nil {
			return err
		}
		if _, err = st.pool.Exec(ctx, `UPDATE profiles SET is_demo=true,demo_expires_at=$2,demo_ready=false WHERE id=$1`, p.ID, expires); err != nil {
			return err
		}
		if err = seed(st, p.ID); err != nil {
			return err
		}
		if _, err = st.pool.Exec(ctx, "UPDATE profiles SET demo_ready=true WHERE id=$1", p.ID); err != nil {
			return err
		}
		return st.CreateSession(ctx, p.ID, auth.HashToken(token), expires)
	})
	return p, pin, token, expires, err
}
func (s *Store) DeleteExpiredDemos(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, "DELETE FROM profiles WHERE is_demo AND demo_expires_at<now()")
	return err
}
