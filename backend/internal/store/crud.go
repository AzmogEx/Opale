package store

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/opale-app/opale/internal/money"
	"strings"
	"time"
)

type MerchantRule struct {
	ID          string `json:"id"`
	MerchantKey string `json:"merchant_key"`
	CategoryID  string `json:"category_id"`
}

func (s *Store) SaveCategory(ctx context.Context, owner, id, name, icon string) (Category, error) {
	var c Category
	var err error
	if id == "" {
		err = s.pool.QueryRow(ctx, `INSERT INTO categories(profile_id,name,icon) VALUES($1,$2,$3) RETURNING id,profile_id,parent_id,name,icon`, owner, name, icon).Scan(&c.ID, &c.ProfileID, &c.ParentID, &c.Name, &c.Icon)
	} else {
		err = s.pool.QueryRow(ctx, `UPDATE categories SET name=$3,icon=$4 WHERE id=$1 AND profile_id=$2 RETURNING id,profile_id,parent_id,name,icon`, id, owner, name, icon).Scan(&c.ID, &c.ProfileID, &c.ParentID, &c.Name, &c.Icon)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		err = ErrNotFound
	}
	return c, err
}
func (s *Store) DeleteCategory(ctx context.Context, owner, id string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM categories WHERE profile_id=$1 AND id=$2`, owner, id)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}
func (s *Store) ListRules(ctx context.Context, owner string) ([]MerchantRule, error) {
	rows, err := s.pool.Query(ctx, `SELECT id,merchant_key,category_id FROM merchant_rules WHERE profile_id=$1 ORDER BY merchant_key`, owner)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []MerchantRule{}
	for rows.Next() {
		var r MerchantRule
		if err = rows.Scan(&r.ID, &r.MerchantKey, &r.CategoryID); err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, rows.Err()
}
func (s *Store) SaveRule(ctx context.Context, owner, id, key, category string) (MerchantRule, error) {
	var r MerchantRule
	var err error
	if id == "" {
		err = s.pool.QueryRow(ctx, `INSERT INTO merchant_rules(profile_id,merchant_key,category_id) VALUES($1,$2,$3) ON CONFLICT(profile_id,merchant_key) DO UPDATE SET category_id=EXCLUDED.category_id RETURNING id,merchant_key,category_id`, owner, key, category).Scan(&r.ID, &r.MerchantKey, &r.CategoryID)
	} else {
		err = s.pool.QueryRow(ctx, `UPDATE merchant_rules SET merchant_key=$3,category_id=$4 WHERE profile_id=$1 AND id=$2 RETURNING id,merchant_key,category_id`, owner, id, key, category).Scan(&r.ID, &r.MerchantKey, &r.CategoryID)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		err = ErrNotFound
	}
	return r, err
}
func (s *Store) DeleteRule(ctx context.Context, owner, id string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM merchant_rules WHERE profile_id=$1 AND id=$2`, owner, id)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}
func (s *Store) UpdateProfile(ctx context.Context, owner string, name, pinHash, privacy *string) (Profile, error) {
	err := s.Atomic(ctx, func(st *Store) error {
		_, err := st.pool.Exec(ctx, `UPDATE profiles SET name=COALESCE($2,name),pin_hash=COALESCE($3,pin_hash),privacy_default=COALESCE($4,privacy_default) WHERE id=$1`, owner, name, pinHash, privacy)
		if err != nil {
			return err
		}
		if pinHash != nil {
			_, err = st.pool.Exec(ctx, `DELETE FROM sessions WHERE profile_id=$1`, owner)
		}
		return err
	})
	if err != nil {
		return Profile{}, err
	}
	return s.GetProfile(ctx, owner)
}
func (s *Store) UpdateValuation(ctx context.Context, owner, id string, value *int64, day *time.Time, note *string) (Valuation, error) {
	var v Valuation
	err := s.Atomic(ctx, func(st *Store) error {
		liability, e := st.valuationLiability(ctx, owner, id)
		if e != nil {
			return e
		}
		if liability != nil {
			if e = st.lockLiability(ctx, owner, *liability); e != nil {
				return e
			}
		}
		e = st.pool.QueryRow(ctx, `UPDATE valuations SET value_cents=COALESCE($3,value_cents),as_of=COALESCE($4,as_of),note=COALESCE($5,note) WHERE profile_id=$1 AND id=$2 RETURNING id,profile_id,asset_id,liability_id,value_cents,as_of,note,created_at`, owner, id, value, day, note).Scan(&v.ID, &v.ProfileID, &v.AssetID, &v.LiabilityID, &v.Value, &v.AsOf, &v.Note, &v.CreatedAt)
		if e != nil {
			return domainNotFound(e)
		}
		if liability != nil {
			return st.validateLiabilityHistory(ctx, owner, *liability)
		}
		return nil
	})
	return v, err
}
func (s *Store) DeleteValuation(ctx context.Context, owner, id string) error {
	return s.Atomic(ctx, func(st *Store) error {
		liability, e := st.valuationLiability(ctx, owner, id)
		if e != nil {
			return e
		}
		if liability != nil {
			if e = st.lockLiability(ctx, owner, *liability); e != nil {
				return e
			}
		}
		tag, e := st.pool.Exec(ctx, `DELETE FROM valuations WHERE profile_id=$1 AND id=$2`, owner, id)
		if e != nil {
			return e
		}
		if tag.RowsAffected() == 0 {
			return ErrNotFound
		}
		if liability != nil {
			return st.validateLiabilityHistory(ctx, owner, *liability)
		}
		return nil
	})
}
func (s *Store) DeleteProfilePushToken(ctx context.Context, owner, token string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM push_tokens WHERE profile_id=$1 AND token=$2`, owner, token)
	return err
}
func (s *Store) IsSpaceOwner(ctx context.Context, space, owner string) (bool, error) {
	var ok bool
	err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM spaces WHERE id=$1 AND created_by=$2)`, space, owner).Scan(&ok)
	return ok, err
}
func (s *Store) CurrentAssetValue(ctx context.Context, owner, id string) (money.Cents, error) {
	var value int64
	err := s.pool.QueryRow(ctx, `SELECT amount_eur(current_asset_value(profile_id,id),currency,CURRENT_DATE,profile_id) FROM assets WHERE profile_id=$1 AND id=$2`, owner, id).Scan(&value)
	if errors.Is(err, pgx.ErrNoRows) {
		err = ErrNotFound
	}
	return money.Cents(value), err
}

// CreateFundedAsset is safe to retry with the same client-generated identifier.
func (s *Store) CreateFundedAsset(ctx context.Context, owner, name, kind, currency, note, requestID string, initial *int64, day time.Time, debt bool) (any, error) {
	var result any
	err := s.Atomic(ctx, func(st *Store) error {
		table := "assets"
		if debt {
			table = "liabilities"
		}
		if requestID != "" {
			if _, err := st.pool.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, owner+table+requestID); err != nil {
				return err
			}
			var id string
			err := st.pool.QueryRow(ctx, `SELECT id FROM `+table+` WHERE profile_id=$1 AND client_request_id=$2`, owner, requestID).Scan(&id)
			if err == nil {
				if debt {
					v, e := st.GetLiability(ctx, owner, id)
					result = v
					return e
				}
				v, e := st.GetAsset(ctx, owner, id)
				result = v
				return e
			}
			if !errors.Is(err, pgx.ErrNoRows) {
				return err
			}
		}
		var id string
		if debt {
			v, err := st.CreateLiability(ctx, owner, name, kind, currency, note)
			if err != nil {
				return err
			}
			result = v
			id = v.ID
		} else {
			v, err := st.CreateAsset(ctx, owner, name, kind, currency, note)
			if err != nil {
				return err
			}
			result = v
			id = v.ID
		}
		if requestID != "" {
			if _, err := st.pool.Exec(ctx, `UPDATE `+table+` SET client_request_id=$3 WHERE id=$1 AND profile_id=$2`, id, owner, requestID); err != nil {
				return err
			}
		}
		if initial != nil {
			var err error
			if debt {
				_, err = st.AddLiabilityValuation(ctx, owner, id, money.Cents(*initial), day, "")
			} else {
				_, err = st.AddAssetValuation(ctx, owner, id, money.Cents(*initial), day, "")
			}
			return err
		}
		return nil
	})
	return result, err
}
func validText(v string) bool { return strings.TrimSpace(v) != "" }
