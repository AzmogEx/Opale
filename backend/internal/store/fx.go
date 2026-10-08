package store

// Multi-devises (EF-008) : taux de change manuels, en micro-euros par unité
// (1 EUR = 1 000 000). Entiers uniquement (ENF-007) ; aucune API externe par
// défaut — la confidentialité d'abord (ENF-005).

import (
	"context"
	"fmt"
	"time"

	"github.com/opale-app/opale/internal/money"
)

// FXRate — un taux de change manuel.
type FXRate struct {
	Currency  string    `json:"currency"`
	RateMicro int64     `json:"rate_micro"` // 1 unité = RateMicro micro-euros
	UpdatedAt time.Time `json:"updated_at"`
	AsOf      string    `json:"as_of,omitempty"`
	Source    string    `json:"source,omitempty"`
}

// ListFXRates — les taux connus + les devises utilisées SANS taux (comptées
// totals blocked until an applicable rate is supplied).
func (s *Store) ListFXRates(ctx context.Context, profileIDs ...string) ([]FXRate, []string, error) {
	profileID := ""
	if len(profileIDs) > 0 {
		profileID = profileIDs[0]
	}
	rows, err := s.pool.Query(ctx, `SELECT DISTINCT ON(currency) currency,rate_micro,updated_at,as_of::text,source FROM (SELECT currency,rate_micro,updated_at,as_of,source,0 priority FROM fx_rates UNION ALL SELECT currency,rate_micro,updated_at,as_of,source,1 priority FROM profile_fx_history WHERE profile_id::text=$1 AND as_of<=CURRENT_DATE) r ORDER BY currency,priority DESC,as_of DESC`, profileID)
	if err != nil {
		return nil, nil, fmt.Errorf("ListFXRates: %w", err)
	}
	defer rows.Close()

	var rates []FXRate
	rated := map[string]bool{"EUR": true}
	for rows.Next() {
		var r FXRate
		if err := rows.Scan(&r.Currency, &r.RateMicro, &r.UpdatedAt, &r.AsOf, &r.Source); err != nil {
			return nil, nil, fmt.Errorf("ListFXRates: %w", err)
		}
		rated[r.Currency] = true
		rates = append(rates, r)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	// Usage scoped to the caller; workers may omit the filter.
	used, err := s.pool.Query(ctx, `
		SELECT DISTINCT currency FROM assets WHERE NOT archived AND ($1='' OR profile_id::text=$1)
		UNION SELECT DISTINCT currency FROM liabilities WHERE NOT archived AND ($1='' OR profile_id::text=$1)
		ORDER BY currency`, profileID)
	if err != nil {
		return nil, nil, fmt.Errorf("ListFXRates: usage : %w", err)
	}
	defer used.Close()

	var unrated []string
	for used.Next() {
		var c string
		if err := used.Scan(&c); err != nil {
			return nil, nil, err
		}
		if !rated[c] {
			unrated = append(unrated, c)
		}
	}
	return rates, unrated, used.Err()
}

// UpsertFXRate crée ou met à jour un taux.
func (s *Store) UpsertFXRate(ctx context.Context, currency string, rateMicro int64) (FXRate, error) {
	var r FXRate
	err := s.pool.QueryRow(ctx, `
		INSERT INTO fx_rates (currency, rate_micro) VALUES ($1, $2)
		ON CONFLICT (currency) DO UPDATE SET rate_micro = $2, updated_at = now()
		RETURNING currency, rate_micro, updated_at`,
		currency, rateMicro,
	).Scan(&r.Currency, &r.RateMicro, &r.UpdatedAt)
	if err != nil {
		return FXRate{}, fmt.Errorf("UpsertFXRate: %w", err)
	}
	return r, nil
}

// DeleteFXRate supprime un taux (la devise repasse à 1:1).
func (s *Store) DeleteFXRate(ctx context.Context, currency string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM fx_rates WHERE currency = $1`, currency)
	if err != nil {
		return fmt.Errorf("DeleteFXRate: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ConvertToEUR — conversion ponctuelle (affichage) : centimes × taux.
func ConvertToEUR(value money.Cents, rateMicro int64) money.Cents {
	return money.Cents(int64(value) * rateMicro / 1_000_000)
}

func (s *Store) UpsertProfileFX(ctx context.Context, owner, currency, day string, rate int64) (FXRate, error) {
	var r FXRate
	err := s.pool.QueryRow(ctx, `INSERT INTO profile_fx_history(profile_id,currency,as_of,rate_micro) VALUES($1,$2,$3,$4) ON CONFLICT(profile_id,currency,as_of) DO UPDATE SET rate_micro=$4,updated_at=now() RETURNING currency,rate_micro,updated_at,as_of::text,source`, owner, currency, day, rate).Scan(&r.Currency, &r.RateMicro, &r.UpdatedAt, &r.AsOf, &r.Source)
	return r, err
}
func (s *Store) DeleteProfileFX(ctx context.Context, owner, currency string) error {
	tag, e := s.pool.Exec(ctx, `DELETE FROM profile_fx_history WHERE profile_id=$1 AND currency=$2`, owner, currency)
	if e == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return e
}

func (s *Store) UpsertReferenceFX(ctx context.Context, currency string, rate int64, day time.Time) error {
	_, e := s.pool.Exec(ctx, `INSERT INTO fx_rates(currency,rate_micro,as_of,source) VALUES($1,$2,$3,'ECB') ON CONFLICT(currency) DO UPDATE SET rate_micro=EXCLUDED.rate_micro,as_of=EXCLUDED.as_of,source='ECB',updated_at=now() WHERE fx_rates.as_of<=EXCLUDED.as_of`, currency, rate, day)
	return e
}
