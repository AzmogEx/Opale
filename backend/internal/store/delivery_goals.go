package store

import (
	"context"
	"fmt"
	"github.com/opale-app/opale/internal/money"
	"time"
)

// SaveGoal serializes allocations per profile. The same observed monthly
// saving capacity cannot be allocated to multiple goals simultaneously.
func (s *Store) SaveGoal(ctx context.Context, profile, id, name, icon string, target money.Cents, date *time.Time, asset *string, saving money.Cents) (Goal, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Goal{}, err
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, `SELECT id FROM profiles WHERE id=$1 FOR UPDATE`, profile); err != nil {
		return Goal{}, err
	}
	if asset != nil {
		var own bool
		err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM assets WHERE id=$1 AND profile_id=$2 AND NOT archived)`, *asset, profile).Scan(&own)
		if err != nil {
			return Goal{}, err
		}
		if !own {
			return Goal{}, ErrNotFound
		}
	}
	if saving > 0 {
		var allocated int64
		if err = tx.QueryRow(ctx, `SELECT COALESCE(SUM(monthly_savings_cents),0) FROM goals WHERE profile_id=$1 AND id::text<>$2`, profile, id).Scan(&allocated); err != nil {
			return Goal{}, err
		}
		var capacity int64
		if err = tx.QueryRow(ctx, `SELECT COALESCE(SUM(eur_cents),0)/3 FROM financial_transactions WHERE profile_id=$1 AND occurred_on>=CURRENT_DATE-INTERVAL '3 months' AND occurred_on<=CURRENT_DATE`, profile).Scan(&capacity); err != nil {
			return Goal{}, err
		}
		if capacity < allocated || int64(saving) > capacity-allocated {
			return Goal{}, fmt.Errorf("%w: épargne affectée supérieure à la capacité mensuelle observée (%s €)", ErrInvalid, money.Cents(max(capacity, 0)).String())
		}
	}
	if id == "" {
		err = tx.QueryRow(ctx, `INSERT INTO goals(profile_id,name,icon,target_cents,target_date,asset_id,monthly_savings_cents) VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING id`, profile, name, icon, target, date, asset, saving).Scan(&id)
	} else {
		var tag interface{ RowsAffected() int64 }
		tag, err = tx.Exec(ctx, `UPDATE goals SET name=$3,icon=$4,target_cents=$5,target_date=$6,asset_id=$7,monthly_savings_cents=$8 WHERE id=$1 AND profile_id=$2`, id, profile, name, icon, target, date, asset, saving)
		if err == nil && tag.RowsAffected() == 0 {
			return Goal{}, ErrNotFound
		}
	}
	if err != nil {
		return Goal{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Goal{}, err
	}
	return scanGoal(s.pool.QueryRow(ctx, goalSelect+` WHERE g.id=$1 AND g.profile_id=$2`, id, profile))
}
