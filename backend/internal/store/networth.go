package store

import (
	"context"
	"fmt"

	"github.com/opale-app/opale/internal/money"
)

// ComputeNetWorth calcule le patrimoine net d'un profil de façon déterministe
// (CA-1, EIA-040). Il somme, pour chaque actif et chaque passif NON archivé, sa
// dernière valorisation connue — convertie en euros si l'actif est libellé
// dans une autre devise (EF-008 : taux manuels de fx_rates, entiers en
// micro-euros ; devise sans taux = comptée 1:1, signalée par /v1/fx).
//
// Aucun calcul n'utilise de float : les centimes sont des entiers, et la
// soustraction finale passe par le package money (détection d'overflow).
func (s *Store) ComputeNetWorth(ctx context.Context, profileID string) (NetWorth, error) {
	var assetsCents, liabCents int64
	err := s.pool.QueryRow(ctx, `
		WITH latest_assets AS (
			SELECT DISTINCT ON (v.asset_id)
			       v.value_cents * COALESCE(fx.rate_micro, 1000000) / 1000000 AS value_cents
			FROM valuations v
			JOIN assets a ON a.id = v.asset_id
			LEFT JOIN fx_rates fx ON fx.currency = a.currency
			WHERE v.profile_id = $1 AND a.archived = false
			ORDER BY v.asset_id, v.as_of DESC, v.created_at DESC
		),
		latest_liab AS (
			SELECT DISTINCT ON (v.liability_id)
			       v.value_cents * COALESCE(fx.rate_micro, 1000000) / 1000000 AS value_cents
			FROM valuations v
			JOIN liabilities l ON l.id = v.liability_id
			LEFT JOIN fx_rates fx ON fx.currency = l.currency
			WHERE v.profile_id = $1 AND l.archived = false
			ORDER BY v.liability_id, v.as_of DESC, v.created_at DESC
		)
		SELECT
			COALESCE((SELECT SUM(value_cents) FROM latest_assets), 0),
			COALESCE((SELECT SUM(value_cents) FROM latest_liab), 0)`,
		profileID,
	).Scan(&assetsCents, &liabCents)
	if err != nil {
		return NetWorth{}, fmt.Errorf("ComputeNetWorth: %w", err)
	}

	net, err := money.Sub(money.Cents(assetsCents), money.Cents(liabCents))
	if err != nil {
		return NetWorth{}, fmt.Errorf("ComputeNetWorth: %w", err)
	}

	return NetWorth{
		AssetsTotal:      money.Cents(assetsCents),
		LiabilitiesTotal: money.Cents(liabCents),
		Net:              net,
		Currency:         "EUR",
	}, nil
}
