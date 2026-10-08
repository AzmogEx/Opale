// Package twin construit le FINANCIAL TWIN (EF-060) : le double financier
// complet d'un profil — revenus, charges, actifs, dettes, objectifs,
// habitudes, risques — assemblé depuis le moteur déterministe.
//
// Ce snapshot a deux usages :
//   - il est renvoyé tel quel à l'app (GET /v1/twin) ;
//   - il sert de contexte à l'IA, en version complète (N2 homelab, privé)
//     Le cloud reçoit exclusivement ai.CloudFacts, jamais ce texte.
package twin

import (
	"fmt"
	"strings"

	"github.com/opale-app/opale/internal/engine"
	"github.com/opale-app/opale/internal/money"
)

// Goal — un objectif résumé pour le twin.
type Goal struct {
	Name    string      `json:"name"`
	Target  money.Cents `json:"target_cents"`
	Percent int         `json:"percent"`
	OnTrack *bool       `json:"on_track,omitempty"`
}

// Snapshot — le double financier complet, 100 % issu du moteur.
type Snapshot struct {
	// Patrimoine.
	Complete          bool                   `json:"complete"`
	MissingValuations int                    `json:"missing_valuations"`
	NetWorth          money.Cents            `json:"net_worth_cents"`
	Assets            money.Cents            `json:"assets_cents"`
	Liabilities       money.Cents            `json:"liabilities_cents"`
	Cash              money.Cents            `json:"cash_cents"`
	AssetKinds        map[string]money.Cents `json:"asset_kinds"`

	// Habitudes (moyennes 3 mois).
	MonthlyIncome   money.Cents `json:"monthly_income_cents"`
	MonthlyExpenses money.Cents `json:"monthly_expenses_cents"`
	MonthlySavings  money.Cents `json:"monthly_savings_cents"`
	FixedMonthly    money.Cents `json:"fixed_monthly_cents"`
	// SavingsRateBps : taux d'épargne en points de base (peut être négatif).
	SavingsRateBps int `json:"savings_rate_bps"`

	// Verdicts du moteur.
	Health       engine.HealthScore  `json:"health"`
	Risks        []engine.Risk       `json:"risks"`
	Independence engine.Independence `json:"independence"`
	Goals        []Goal              `json:"goals"`
}

// kindLabels : libellés français neutres par type d'actif.
var kindLabels = map[string]string{
	"checking":    "comptes courants",
	"savings":     "livrets",
	"stocks":      "placements actions",
	"crypto":      "crypto",
	"real_estate": "immobilier",
	"object":      "objets de valeur",
	"other":       "autres actifs",
}

func kindLabel(kind string) string {
	if l, ok := kindLabels[kind]; ok {
		return l
	}
	return kind
}

// Describe produit le contexte complet pour le homelab (N2 — privé, pas
// d'anonymisation nécessaire) : mêmes données, montants exacts en euros.
func Describe(s Snapshot) string {
	var b strings.Builder
	b.WriteString("Situation financière actuelle (chiffres exacts du moteur) :\n")
	if !s.Complete {
		fmt.Fprintf(&b, "Données incomplètes : %d actifs ou dettes sans valorisation ; les totaux représentent seulement les valeurs renseignées.\n", s.MissingValuations)
	}
	fmt.Fprintf(&b, "- Patrimoine net : %s (actifs %s, dettes %s)\n",
		eurosFR(s.NetWorth), eurosFR(s.Assets), eurosFR(s.Liabilities))
	fmt.Fprintf(&b, "- Cash disponible : %s\n", eurosFR(s.Cash))
	for _, kind := range sortedKinds(s.AssetKinds) {
		fmt.Fprintf(&b, "- %s : %s\n", kindLabel(kind), eurosFR(s.AssetKinds[kind]))
	}
	fmt.Fprintf(&b, "- Revenu mensuel moyen : %s ; dépenses : %s ; épargne : %s (taux %d %%)\n",
		eurosFR(s.MonthlyIncome), eurosFR(s.MonthlyExpenses), eurosFR(s.MonthlySavings), s.SavingsRateBps/100)
	if s.FixedMonthly > 0 {
		fmt.Fprintf(&b, "- Charges fixes mensuelles : %s\n", eurosFR(s.FixedMonthly))
	}
	fmt.Fprintf(&b, "- Score de santé financière : %d/100\n", s.Health.Score)
	for _, c := range s.Health.Components {
		fmt.Fprintf(&b, "  - %s : %d/%d (%s)\n", c.Name, c.Score, c.Max, c.Comment)
	}
	for _, g := range s.Goals {
		fmt.Fprintf(&b, "- Objectif « %s » : cible %s, avancement %d %%\n", g.Name, eurosFR(g.Target), g.Percent)
	}
	if s.Independence.Target > 0 {
		if s.Independence.Reached {
			fmt.Fprintf(&b, "- Indépendance financière : dans %d mois (cible %s)\n",
				s.Independence.Months, eurosFR(s.Independence.Target))
		} else {
			fmt.Fprintf(&b, "- Indépendance financière : hors d'atteinte (cible %s)\n", eurosFR(s.Independence.Target))
		}
	}
	for _, r := range s.Risks {
		fmt.Fprintf(&b, "- Risque (%s) : %s — %s\n", r.Severity, r.Title, r.Detail)
	}
	return b.String()
}

// eurosFR : montant exact (usage interne N2, pas cloud).
func eurosFR(c money.Cents) string {
	return c.String() + " €"
}

// sortedKinds : itération déterministe de la carte des types d'actifs.
func sortedKinds(m map[string]money.Cents) []string {
	kinds := make([]string, 0, len(m))
	for k := range m {
		kinds = append(kinds, k)
	}
	// tri simple par insertion (peu d'éléments)
	for i := 1; i < len(kinds); i++ {
		for j := i; j > 0 && kinds[j] < kinds[j-1]; j-- {
			kinds[j], kinds[j-1] = kinds[j-1], kinds[j]
		}
	}
	return kinds
}
