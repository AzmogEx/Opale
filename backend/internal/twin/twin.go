// Package twin construit le FINANCIAL TWIN (EF-060) : le double financier
// complet d'un profil — revenus, charges, actifs, dettes, objectifs,
// habitudes, risques — assemblé depuis le moteur déterministe.
//
// Ce snapshot a deux usages :
//   - il est renvoyé tel quel à l'app (GET /v1/twin) ;
//   - il sert de contexte à l'IA, en version complète (N2 homelab, privé)
//     ou ANONYMISÉE (N3 cloud, EIA-031/033).
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
	NetWorth    money.Cents            `json:"net_worth_cents"`
	Assets      money.Cents            `json:"assets_cents"`
	Liabilities money.Cents            `json:"liabilities_cents"`
	Cash        money.Cents            `json:"cash_cents"`
	AssetKinds  map[string]money.Cents `json:"asset_kinds"`

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

// compactK arrondit un montant au millier d'euros le plus proche et le
// formate façon « 42k » / « 3.2k » / « 250 » (EIA-031 : montants agrégés).
// L'arrondi EST l'anonymisation : on ne transmet jamais le centime près.
func compactK(c money.Cents) string {
	euros := int64(c) / 100
	neg := ""
	if euros < 0 {
		neg = "-"
		euros = -euros
	}
	switch {
	case euros >= 10_000:
		// ≥ 10 k€ : au millier près.
		return fmt.Sprintf("%s%dk", neg, (euros+500)/1_000)
	case euros >= 1_000:
		// 1–10 k€ : à la centaine près, notation 3.2k.
		hundreds := (euros + 50) / 100 // en centaines
		if hundreds%10 == 0 {
			return fmt.Sprintf("%s%dk", neg, hundreds/10)
		}
		return fmt.Sprintf("%s%d.%dk", neg, hundreds/10, hundreds%10)
	default:
		// < 1 k€ : à la dizaine près.
		return fmt.Sprintf("%s%d", neg, (euros+5)/10*10)
	}
}

// Anonymize produit le contexte N2-safe pour le cloud (EIA-033) : profil
// anonyme, montants agrégés/arrondis, ni nom, ni banque, ni transaction.
func Anonymize(s Snapshot) string {
	var b strings.Builder
	b.WriteString("Profil A :\n")
	fmt.Fprintf(&b, "  patrimoine net %s\n", compactK(s.NetWorth))
	fmt.Fprintf(&b, "  cash %s\n", compactK(s.Cash))
	for _, kind := range sortedKinds(s.AssetKinds) {
		if s.AssetKinds[kind] > 0 && kind != "checking" && kind != "savings" {
			fmt.Fprintf(&b, "  %s %s\n", kindLabel(kind), compactK(s.AssetKinds[kind]))
		}
	}
	if s.Liabilities > 0 {
		fmt.Fprintf(&b, "  dettes %s\n", compactK(s.Liabilities))
	}
	fmt.Fprintf(&b, "  revenu mensuel %s\n", compactK(s.MonthlyIncome))
	fmt.Fprintf(&b, "  dépenses mensuelles %s\n", compactK(s.MonthlyExpenses))
	if s.FixedMonthly > 0 {
		fmt.Fprintf(&b, "  charges fixes %s\n", compactK(s.FixedMonthly))
	}
	fmt.Fprintf(&b, "  taux d'épargne %d %%\n", s.SavingsRateBps/100)
	fmt.Fprintf(&b, "  score de santé %d/100\n", s.Health.Score)
	for _, g := range s.Goals {
		// Le nom d'objectif est de la donnée N2 (« objectifs » autorisés) —
		// mais on ne transmet que nom générique + montant arrondi.
		fmt.Fprintf(&b, "  objectif %s %s (%d %%)\n", strings.ToLower(g.Name), compactK(g.Target), g.Percent)
	}
	if s.Independence.Target > 0 {
		if s.Independence.Reached {
			fmt.Fprintf(&b, "  indépendance financière dans %d mois (cible %s)\n",
				s.Independence.Months, compactK(s.Independence.Target))
		} else {
			fmt.Fprintf(&b, "  indépendance financière hors d'atteinte (cible %s)\n",
				compactK(s.Independence.Target))
		}
	}
	for _, r := range s.Risks {
		fmt.Fprintf(&b, "  risque détecté : %s (%s)\n", r.Title, r.Severity)
	}
	return b.String()
}

// Describe produit le contexte complet pour le homelab (N2 — privé, pas
// d'anonymisation nécessaire) : mêmes données, montants exacts en euros.
func Describe(s Snapshot) string {
	var b strings.Builder
	b.WriteString("Situation financière actuelle (chiffres exacts du moteur) :\n")
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

// eurosFR : montant exact en euros entiers (usage interne N2, pas cloud).
func eurosFR(c money.Cents) string {
	return fmt.Sprintf("%d €", int64(c)/100)
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
