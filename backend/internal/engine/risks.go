package engine

import (
	"fmt"
	"sort"

	"github.com/opale-app/opale/internal/money"
)

// Radar de risques (EF-061) : détecte ce qui peut casser le plan.
// Règles 100 % déterministes sur des mesures agrégées — l'IA ne fait
// qu'expliquer ces verdicts (EIA-041).

// Sévérités du radar, de la simple info au signal critique.
const (
	SeverityInfo     = "info"
	SeverityWarning  = "warning"
	SeverityCritical = "critical"
)

// Risk — un risque détecté par le radar.
type Risk struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Severity string `json:"severity"`
	Detail   string `json:"detail"`
}

// RiskInputs — mesures nécessaires au radar. Rassemblées par le store,
// le calcul reste pur et testable (CA-2).
type RiskInputs struct {
	Cash        money.Cents // cash disponible (comptes + livrets)
	Assets      money.Cents // total actifs
	Liabilities money.Cents // total dettes
	Income3M    money.Cents // revenus des 3 derniers mois
	Expenses3M  money.Cents // dépenses des 3 derniers mois (positives)
	FixedMonthly money.Cents // charges fixes mensuelles (récurrentes, positives)
	// IncomeSources : nombre de flux récurrents entrants distincts.
	IncomeSources int
	// ProjectedCash30d : cash projeté à 30 jours (si HasProjection).
	ProjectedCash30d money.Cents
	HasProjection    bool
	// Valeur par type d'actif (concentration / illiquidité).
	AssetKindValues map[string]money.Cents
}

// severityRank ordonne critical > warning > info.
func severityRank(s string) int {
	switch s {
	case SeverityCritical:
		return 0
	case SeverityWarning:
		return 1
	default:
		return 2
	}
}

// DetectRisks passe les règles du radar et renvoie les risques détectés,
// triés par sévérité décroissante (ordre stable et déterministe).
func DetectRisks(in RiskInputs) []Risk {
	var risks []Risk
	add := func(id, title, severity, detail string) {
		risks = append(risks, Risk{ID: id, Title: title, Severity: severity, Detail: detail})
	}

	monthlyExpenses := int64(in.Expenses3M) / 3
	monthlyIncome := int64(in.Income3M) / 3

	// ── Taux d'épargne négatif ────────────────────────────────────────────
	if in.Income3M > 0 && in.Expenses3M > in.Income3M {
		add("negative_savings", "Épargne négative", SeverityCritical,
			"Sur 3 mois, tes dépenses dépassent tes revenus : le patrimoine se vide.")
	}

	// ── Fonds d'urgence insuffisant ───────────────────────────────────────
	if monthlyExpenses > 0 {
		monthsX100 := int64(in.Cash) * 100 / monthlyExpenses
		switch {
		case monthsX100 < 100:
			add("emergency_fund", "Fonds d'urgence quasi inexistant", SeverityCritical,
				"Moins d'un mois de dépenses en réserve : un imprévu casserait le plan.")
		case monthsX100 < 300:
			add("emergency_fund", "Fonds d'urgence insuffisant", SeverityWarning,
				fmt.Sprintf("Environ %d mois de dépenses en réserve — vise au moins 3, idéalement 6.", monthsX100/100))
		}
	}

	// ── Surendettement ────────────────────────────────────────────────────
	if in.Liabilities > 0 {
		if in.Assets <= 0 {
			add("debt", "Dettes sans actifs en face", SeverityCritical,
				"Des dettes existent sans actifs pour les couvrir.")
		} else {
			ratioBps := int64(in.Liabilities) * 10_000 / int64(in.Assets)
			switch {
			case ratioBps >= 6_000:
				add("debt", "Endettement lourd", SeverityCritical,
					"Les dettes représentent plus de 60 % des actifs.")
			case ratioBps >= 4_000:
				add("debt", "Endettement à surveiller", SeverityWarning,
					"Les dettes dépassent 40 % des actifs.")
			}
		}
	}

	// ── Cash dormant ──────────────────────────────────────────────────────
	if monthlyExpenses > 0 && in.Assets > 0 {
		monthsOfCash := int64(in.Cash) / monthlyExpenses
		cashShareBps := int64(in.Cash) * 10_000 / int64(in.Assets)
		if monthsOfCash >= 12 && cashShareBps >= 3_000 {
			add("idle_cash", "Cash dormant", SeverityInfo,
				fmt.Sprintf("%d mois de dépenses dorment en liquidités (%d %% du patrimoine) : au-delà du fonds d'urgence, ce cash perd de la valeur avec l'inflation.",
					monthsOfCash, cashShareBps/100))
		}
	}

	// ── Dépendance à un revenu unique ─────────────────────────────────────
	if in.IncomeSources == 1 {
		add("single_income", "Dépendance à un revenu unique", SeverityWarning,
			"Tout le plan repose sur une seule source de revenus récurrente.")
	}

	// ── Charges fixes écrasantes ──────────────────────────────────────────
	if monthlyIncome > 0 && in.FixedMonthly > 0 {
		ratioBps := int64(in.FixedMonthly) * 10_000 / monthlyIncome
		switch {
		case ratioBps >= 8_000:
			add("fixed_costs", "Charges fixes écrasantes", SeverityCritical,
				"Les charges fixes absorbent plus de 80 % des revenus : aucune marge de manœuvre.")
		case ratioBps >= 6_000:
			add("fixed_costs", "Charges fixes élevées", SeverityWarning,
				"Les charges fixes dépassent 60 % des revenus.")
		}
	}

	// ── Concentration / illiquidité ───────────────────────────────────────
	if len(in.AssetKindValues) > 0 {
		var total int64
		for _, v := range in.AssetKindValues {
			total += int64(v)
		}
		if total > 0 {
			// Concentration : un seul type d'actif domine.
			kinds := make([]string, 0, len(in.AssetKindValues))
			for k := range in.AssetKindValues {
				kinds = append(kinds, k)
			}
			sort.Strings(kinds) // ordre déterministe
			for _, k := range kinds {
				share := int64(in.AssetKindValues[k]) * 10_000 / total
				if share >= 8_000 && len(in.AssetKindValues) >= 2 {
					add("concentration", "Patrimoine trop concentré", SeverityWarning,
						fmt.Sprintf("Un seul type d'actif (%s) pèse %d %% du patrimoine.", k, share/100))
					break
				}
			}
			// Illiquidité : immobilier + objets ≥ 70 %.
			illiquid := int64(in.AssetKindValues["real_estate"]) + int64(in.AssetKindValues["object"])
			if illiquid*10_000/total >= 7_000 {
				add("illiquidity", "Patrimoine peu liquide", SeverityWarning,
					"Plus de 70 % du patrimoine est difficile à mobiliser rapidement (immobilier, objets).")
			}
		}
	}

	// ── Solde prévisionnel bas ────────────────────────────────────────────
	if in.HasProjection {
		switch {
		case in.ProjectedCash30d < 0:
			add("low_cash_ahead", "Découvert prévu sous 30 jours", SeverityCritical,
				"Au rythme actuel, le cash passe en négatif dans le mois.")
		case monthlyExpenses > 0 && int64(in.ProjectedCash30d) < monthlyExpenses:
			add("low_cash_ahead", "Cash prévu très bas", SeverityWarning,
				"Dans 30 jours, il restera moins d'un mois de dépenses en cash.")
		}
	}

	// Tri stable : critique d'abord, puis l'ordre d'évaluation.
	sort.SliceStable(risks, func(i, j int) bool {
		return severityRank(risks[i].Severity) < severityRank(risks[j].Severity)
	})
	return risks
}
