package store

import (
	"context"
	"fmt"
	"time"

	"github.com/opale-app/opale/internal/money"
)

// NetWorthPoint — un point daté de la courbe du patrimoine net (EF-013).
type NetWorthPoint struct {
	AsOf             time.Time   `json:"as_of"`
	AssetsTotal      money.Cents `json:"assets_total_cents"`
	LiabilitiesTotal money.Cents `json:"liabilities_total_cents"`
	Net              money.Cents `json:"net_cents"`
}

// NetWorthHistory — série temporelle du patrimoine net d'un profil.
type NetWorthHistory struct {
	Points   []NetWorthPoint `json:"points"`
	Currency string          `json:"currency"`
}

// ComputeNetWorthHistory calcule l'évolution du patrimoine net sur les `months`
// derniers mois (EF-012/EF-013), de façon déterministe (EIA-040) : un point par
// fin de mois (le dernier point est plafonné à aujourd'hui), chaque point sommant
// la dernière valorisation connue de chaque actif/passif NON archivé à cette date.
//
// Comme partout : centimes entiers, soustraction via le package money.
func (s *Store) ComputeNetWorthHistory(ctx context.Context, profileID string, months int) (NetWorthHistory, error) {
	rows, err := s.pool.Query(ctx, `
		WITH month_starts AS (
			SELECT generate_series(
				date_trunc('month', CURRENT_DATE) - make_interval(months => $2 - 1),
				date_trunc('month', CURRENT_DATE),
				interval '1 month'
			)::date AS month_start
		),
		points AS (
			SELECT LEAST(
				(month_start + interval '1 month' - interval '1 day')::date,
				CURRENT_DATE
			) AS as_of
			FROM month_starts
		)
		SELECT
			p.as_of,
			COALESCE((
				SELECT SUM(lv.value_cents) FROM (
					SELECT DISTINCT ON (v.asset_id)
					       v.value_cents * COALESCE(fx.rate_micro, 1000000) / 1000000 AS value_cents
					FROM valuations v
					JOIN assets a ON a.id = v.asset_id
					LEFT JOIN fx_rates fx ON fx.currency = a.currency
					WHERE v.profile_id = $1 AND a.archived = false AND v.as_of <= p.as_of
					ORDER BY v.asset_id, v.as_of DESC, v.created_at DESC
				) lv
			), 0) AS assets_cents,
			COALESCE((
				SELECT SUM(lv.value_cents) FROM (
					SELECT DISTINCT ON (v.liability_id)
					       v.value_cents * COALESCE(fx.rate_micro, 1000000) / 1000000 AS value_cents
					FROM valuations v
					JOIN liabilities l ON l.id = v.liability_id
					LEFT JOIN fx_rates fx ON fx.currency = l.currency
					WHERE v.profile_id = $1 AND l.archived = false AND v.as_of <= p.as_of
					ORDER BY v.liability_id, v.as_of DESC, v.created_at DESC
				) lv
			), 0) AS liabilities_cents
		FROM points p
		ORDER BY p.as_of`,
		profileID, months,
	)
	if err != nil {
		return NetWorthHistory{}, fmt.Errorf("ComputeNetWorthHistory: %w", err)
	}
	defer rows.Close()

	h := NetWorthHistory{Points: []NetWorthPoint{}, Currency: "EUR"}
	for rows.Next() {
		var (
			pt                     NetWorthPoint
			assetsCents, liabCents int64
		)
		if err := rows.Scan(&pt.AsOf, &assetsCents, &liabCents); err != nil {
			return NetWorthHistory{}, fmt.Errorf("ComputeNetWorthHistory: scan: %w", err)
		}
		net, err := money.Sub(money.Cents(assetsCents), money.Cents(liabCents))
		if err != nil {
			return NetWorthHistory{}, fmt.Errorf("ComputeNetWorthHistory: %w", err)
		}
		pt.AssetsTotal = money.Cents(assetsCents)
		pt.LiabilitiesTotal = money.Cents(liabCents)
		pt.Net = net
		h.Points = append(h.Points, pt)
	}
	if err := rows.Err(); err != nil {
		return NetWorthHistory{}, fmt.Errorf("ComputeNetWorthHistory: rows: %w", err)
	}
	return h, nil
}
