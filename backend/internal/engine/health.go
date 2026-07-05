package engine

import "github.com/opale-app/opale/internal/money"

// HealthInputs — mesures nécessaires au score de santé financière (EF-015).
// Toutes rassemblées par le store ; le calcul reste pur et testable.
type HealthInputs struct {
	Income3M   money.Cents // revenus des 3 derniers mois
	Expenses3M money.Cents // dépenses des 3 derniers mois (valeur positive)
	Cash       money.Cents // cash disponible (comptes + livrets)
	Assets     money.Cents // total actifs
	Liabilities money.Cents // total dettes
	FixedMonthly money.Cents // dépenses fixes mensuelles (récurrentes, positive)
	// Valeur par type d'actif (répartition / diversification).
	AssetKindValues map[string]money.Cents
}

// HealthComponent — une composante du score, avec son verdict.
type HealthComponent struct {
	Name    string `json:"name"`
	Score   int    `json:"score"`
	Max     int    `json:"max"`
	Comment string `json:"comment"`
}

// HealthScore — le score global /100 (EF-015).
type HealthScore struct {
	Score      int               `json:"score"`
	Components []HealthComponent `json:"components"`
}

// ComputeHealthScore calcule le score de santé financière /100, en cinq
// composantes pondérées. Interpolations linéaires en arithmétique entière.
//
//	Taux d'épargne        /25 — plein à ≥ 20 % des revenus
//	Fonds d'urgence       /25 — plein à ≥ 6 mois de dépenses en cash
//	Endettement           /20 — plein à ≤ 10 % des actifs, nul à ≥ 60 %
//	Diversification       /15 — plein à ≥ 4 types d'actifs significatifs (> 5 %)
//	Poids des charges fixes /15 — plein à ≤ 40 % des revenus, nul à ≥ 80 %
func ComputeHealthScore(in HealthInputs) HealthScore {
	var comps []HealthComponent

	// ── Taux d'épargne /25 ────────────────────────────────────────────────
	{
		score, comment := 0, "aucun revenu enregistré sur 3 mois"
		if in.Income3M > 0 {
			saved := int64(in.Income3M) - int64(in.Expenses3M)
			rateBps := saved * 10_000 / int64(in.Income3M) // peut être négatif
			score = scaleUp(rateBps, 0, 2_000, 25)
			switch {
			case rateBps <= 0:
				comment = "tu dépenses plus que tu ne gagnes"
			case rateBps >= 2_000:
				comment = "excellente capacité d'épargne (≥ 20 %)"
			default:
				comment = "épargne en construction"
			}
		}
		comps = append(comps, HealthComponent{"Taux d'épargne", score, 25, comment})
	}

	// ── Fonds d'urgence /25 ───────────────────────────────────────────────
	{
		monthly := int64(in.Expenses3M) / 3
		score, comment := 0, "dépenses mensuelles inconnues"
		if monthly > 0 {
			// mois de réserve × 100 pour garder de la précision entière
			monthsX100 := int64(in.Cash) * 100 / monthly
			score = scaleUp(monthsX100, 0, 600, 25)
			switch {
			case monthsX100 >= 600:
				comment = "≥ 6 mois de dépenses en réserve"
			case monthsX100 >= 300:
				comment = "réserve correcte, vise 6 mois"
			default:
				comment = "fonds d'urgence insuffisant"
			}
		} else if in.Cash > 0 {
			score, comment = 25, "du cash disponible et aucune dépense connue"
		}
		comps = append(comps, HealthComponent{"Fonds d'urgence", score, 25, comment})
	}

	// ── Endettement /20 ───────────────────────────────────────────────────
	{
		score, comment := 20, "aucune dette"
		if in.Liabilities > 0 {
			if in.Assets <= 0 {
				score, comment = 0, "dettes sans actifs en face"
			} else {
				ratioBps := int64(in.Liabilities) * 10_000 / int64(in.Assets)
				score = scaleDown(ratioBps, 1_000, 6_000, 20)
				switch {
				case ratioBps <= 1_000:
					comment = "endettement très maîtrisé (≤ 10 %)"
				case ratioBps >= 6_000:
					comment = "endettement lourd (≥ 60 % des actifs)"
				default:
					comment = "endettement raisonnable"
				}
			}
		}
		comps = append(comps, HealthComponent{"Endettement", score, 20, comment})
	}

	// ── Diversification /15 ───────────────────────────────────────────────
	{
		var total int64
		for _, v := range in.AssetKindValues {
			total += int64(v)
		}
		score, comment := 0, "aucun actif"
		if total > 0 {
			significant := 0
			domBps := int64(0)
			for _, v := range in.AssetKindValues {
				share := int64(v) * 10_000 / total
				if share > 500 { // > 5 %
					significant++
				}
				if share > domBps {
					domBps = share
				}
			}
			score = scaleUp(int64(significant)*100, 100, 400, 15)
			comment = "diversification à construire"
			if significant >= 4 {
				comment = "patrimoine bien diversifié"
			}
			if domBps >= 8_000 && significant >= 2 {
				// Un type d'actif ≥ 80 % : concentration excessive.
				score = min(score, 7)
				comment = "trop concentré sur un seul type d'actif"
			}
		}
		comps = append(comps, HealthComponent{"Diversification", score, 15, comment})
	}

	// ── Poids des charges fixes /15 ───────────────────────────────────────
	{
		monthlyIncome := int64(in.Income3M) / 3
		score, comment := 0, "revenus mensuels inconnus"
		if monthlyIncome > 0 {
			ratioBps := int64(in.FixedMonthly) * 10_000 / monthlyIncome
			score = scaleDown(ratioBps, 4_000, 8_000, 15)
			switch {
			case ratioBps <= 4_000:
				comment = "charges fixes légères (≤ 40 % des revenus)"
			case ratioBps >= 8_000:
				comment = "charges fixes écrasantes (≥ 80 %)"
			default:
				comment = "charges fixes contenues"
			}
		}
		comps = append(comps, HealthComponent{"Charges fixes", score, 15, comment})
	}

	total := 0
	for _, c := range comps {
		total += c.Score
	}
	return HealthScore{Score: total, Components: comps}
}

// scaleUp : interpolation linéaire croissante — v ≤ lo → 0, v ≥ hi → max.
func scaleUp(v, lo, hi int64, max int) int {
	if v <= lo {
		return 0
	}
	if v >= hi {
		return max
	}
	return int((v - lo) * int64(max) / (hi - lo))
}

// scaleDown : interpolation linéaire décroissante — v ≤ lo → max, v ≥ hi → 0.
func scaleDown(v, lo, hi int64, max int) int {
	if v <= lo {
		return max
	}
	if v >= hi {
		return 0
	}
	return int((hi - v) * int64(max) / (hi - lo))
}
