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
// micro-euros ; un taux manquant bloque explicitement le calcul).
//
// Aucun calcul n'utilise de float : les centimes sont des entiers, et la
// soustraction finale passe par le package money (détection d'overflow).
func (s *Store) ComputeNetWorth(ctx context.Context, profileID string) (NetWorth, error) {
	var assetsCents, liabCents int64
	var missing int
	err := s.pool.QueryRow(ctx, `
 SELECT
 COALESCE((SELECT SUM(amount_eur(current_asset_value(a.profile_id,a.id),a.currency,CURRENT_DATE,a.profile_id)) FROM assets a WHERE a.profile_id=$1 AND NOT a.archived),0),
 COALESCE((SELECT SUM(amount_eur(current_liability_value(l.profile_id,l.id),l.currency,CURRENT_DATE,l.profile_id)) FROM liabilities l WHERE l.profile_id=$1 AND NOT l.archived),0),
 (SELECT count(*) FROM assets a WHERE a.profile_id=$1 AND NOT a.archived AND NOT EXISTS(SELECT 1 FROM valuations v WHERE v.profile_id=a.profile_id AND v.asset_id=a.id AND v.as_of<=CURRENT_DATE))+
 (SELECT count(*) FROM liabilities l WHERE l.profile_id=$1 AND NOT l.archived AND NOT EXISTS(SELECT 1 FROM valuations v WHERE v.profile_id=l.profile_id AND v.liability_id=l.id AND v.as_of<=CURRENT_DATE))`, profileID).Scan(&assetsCents, &liabCents, &missing)
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
		Complete:         missing == 0, MissingValuations: missing,
	}, nil
}
