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
}

// ListFXRates — les taux connus + les devises utilisées SANS taux (comptées
// 1:1 dans le patrimoine, à signaler à l'utilisateur).
func (s *Store) ListFXRates(ctx context.Context) ([]FXRate, []string, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT currency, rate_micro, updated_at FROM fx_rates ORDER BY currency`)
	if err != nil {
		return nil, nil, fmt.Errorf("ListFXRates: %w", err)
	}
	defer rows.Close()

	var rates []FXRate
	rated := map[string]bool{"EUR": true}
	for rows.Next() {
		var r FXRate
		if err := rows.Scan(&r.Currency, &r.RateMicro, &r.UpdatedAt); err != nil {
			return nil, nil, fmt.Errorf("ListFXRates: %w", err)
		}
		rated[r.Currency] = true
		rates = append(rates, r)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	// Devises en usage (actifs + passifs, tous profils du foyer confondus).
	used, err := s.pool.Query(ctx, `
		SELECT DISTINCT currency FROM assets WHERE NOT archived
		UNION SELECT DISTINCT currency FROM liabilities WHERE NOT archived
		ORDER BY currency`)
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
