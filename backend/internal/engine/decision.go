package engine

import "github.com/opale-app/opale/internal/money"

// Mode Décision (EF-052) : « puis-je me permettre X ? »
// Le moteur projette la trajectoire AVEC et SANS la décision, en trois
// scénarios de rendement (prudent / normal / ambitieux), et mesure l'impact
// à 0 / 5 / 10 ans plus le retard d'indépendance financière.
// Tout est entier et déterministe (ENF-007, EIA-040) — l'IA ne fait
// qu'expliquer ce résultat.

// DecisionInputs — la situation actuelle et la décision envisagée.
type DecisionInputs struct {
	// Situation actuelle.
	NetWorth        money.Cents // patrimoine net de départ
	Cash            money.Cents // cash disponible (test de faisabilité)
	MonthlySavings  money.Cents // épargne mensuelle actuelle
	MonthlyExpenses money.Cents // dépenses mensuelles (cible d'indépendance)
	AnnualReturnBps int         // rendement annuel de référence
	SwrBps          int         // taux de retrait sûr (règle des 4 % = 400)

	// La décision.
	OneTimeCost money.Cents // coût immédiat (négatif = rentrée d'argent)
	MonthlyCost money.Cents // charge mensuelle ajoutée (négatif = économie)
}

// DecisionScenario — l'impact de la décision sous une hypothèse de rendement.
type DecisionScenario struct {
	Name      string `json:"name"` // prudent | normal | ambitieux
	ReturnBps int    `json:"return_bps"`

	// Patrimoine projeté AVEC la décision.
	In5y  money.Cents `json:"in_5y_cents"`
	In10y money.Cents `json:"in_10y_cents"`
	// Écart vs la trajectoire SANS la décision (négatif = ça coûte).
	Delta5y  money.Cents `json:"delta_5y_cents"`
	Delta10y money.Cents `json:"delta_10y_cents"`

	// Indépendance financière : atteinte, et retard induit (mois).
	BaselineReached bool `json:"baseline_reached"`
	DecisionReached bool `json:"decision_reached"`
	// DelayMonths n'a de sens que si les deux trajectoires atteignent la
	// cible (positif = la décision retarde la liberté).
	DelayMonths int `json:"delay_months"`
}

// DecisionImpact — le verdict complet du Mode Décision.
type DecisionImpact struct {
	// Impact immédiat : patrimoine juste après la décision.
	NetWorthAfter money.Cents `json:"net_worth_after_cents"`
	// Faisabilité cash : le coût immédiat tient dans le cash disponible.
	AffordableCash bool `json:"affordable_cash"`
	// Épargne mensuelle après décision (négative = la décision te met à découvert chaque mois).
	SavingsAfter money.Cents `json:"savings_after_cents"`

	RiskLevel      string             `json:"risk_level"` // faible | modéré | élevé
	Recommendation string             `json:"recommendation"`
	Scenarios      []DecisionScenario `json:"scenarios"`
}

// clampBps borne un rendement dans [0, 1200] (0–12 %/an), la plage
// acceptée par Project pour des scénarios réalistes.
func clampBps(bps int) int {
	if bps < 0 {
		return 0
	}
	if bps > 1_200 {
		return 1_200
	}
	return bps
}

// netAt renvoie le patrimoine projeté au mois donné.
func netAt(points []ProjectionPoint, month int) money.Cents {
	if month < len(points) {
		return points[month].Net
	}
	return points[len(points)-1].Net
}

// EvaluateDecision calcule l'impact déterministe d'une décision.
func EvaluateDecision(in DecisionInputs) (DecisionImpact, error) {
	if in.SwrBps <= 0 || in.SwrBps > bpsScale {
		return DecisionImpact{}, ErrInvalidInput
	}

	startAfter := in.NetWorth - in.OneTimeCost
	savingsAfter := in.MonthlySavings - in.MonthlyCost
	expensesAfter := in.MonthlyExpenses + in.MonthlyCost

	impact := DecisionImpact{
		NetWorthAfter:  startAfter,
		AffordableCash: in.OneTimeCost <= in.Cash,
		SavingsAfter:   savingsAfter,
	}

	// Trois scénarios de rendement autour de l'hypothèse de référence.
	scenarios := []struct {
		name string
		bps  int
	}{
		{"prudent", clampBps(in.AnnualReturnBps - 200)},
		{"normal", clampBps(in.AnnualReturnBps)},
		{"ambitieux", clampBps(in.AnnualReturnBps + 200)},
	}

	for _, sc := range scenarios {
		baseline, err := Project(in.NetWorth, in.MonthlySavings, sc.bps, 120)
		if err != nil {
			return DecisionImpact{}, err
		}
		decided, err := Project(startAfter, savingsAfter, sc.bps, 120)
		if err != nil {
			return DecisionImpact{}, err
		}

		s := DecisionScenario{
			Name:      sc.name,
			ReturnBps: sc.bps,
			In5y:      netAt(decided, 60),
			In10y:     netAt(decided, 120),
			Delta5y:   netAt(decided, 60) - netAt(baseline, 60),
			Delta10y:  netAt(decided, 120) - netAt(baseline, 120),
		}

		// Retard d'indépendance : seulement si les dépenses restent > 0.
		if in.MonthlyExpenses > 0 && expensesAfter > 0 {
			base, err := ComputeIndependence(in.NetWorth, in.MonthlySavings, in.MonthlyExpenses, sc.bps, in.SwrBps)
			if err != nil {
				return DecisionImpact{}, err
			}
			after, err := ComputeIndependence(startAfter, savingsAfter, expensesAfter, sc.bps, in.SwrBps)
			if err != nil {
				return DecisionImpact{}, err
			}
			s.BaselineReached = base.Reached
			s.DecisionReached = after.Reached
			if base.Reached && after.Reached {
				s.DelayMonths = after.Months - base.Months
			}
		}

		impact.Scenarios = append(impact.Scenarios, s)
	}

	impact.RiskLevel, impact.Recommendation = decisionVerdict(in, impact)
	return impact, nil
}

// decisionVerdict classe le risque et formule une recommandation
// déterministe (gabarits) — l'IA pourra la reformuler, jamais la contredire.
func decisionVerdict(in DecisionInputs, impact DecisionImpact) (level, reco string) {
	normal := impact.Scenarios[1] // le scénario « normal »

	switch {
	case impact.SavingsAfter < 0:
		return "élevé", "Cette décision rend ton épargne mensuelle négative : chaque mois, tu t'appauvris. À éviter en l'état, ou compense en réduisant d'autres charges."
	case in.OneTimeCost > 0 && !impact.AffordableCash:
		return "élevé", "Le coût immédiat dépasse ton cash disponible : impossible sans emprunter ou vendre des actifs. Reporte ou finance autrement."
	case in.MonthlyCost > 0 && in.MonthlySavings > 0 && int64(in.MonthlyCost)*2 > int64(in.MonthlySavings):
		if normal.DelayMonths > 0 {
			return "modéré", "Faisable, mais la charge mensuelle absorbe plus de la moitié de ton épargne et retarde ton indépendance. À faire seulement si ça compte vraiment pour toi."
		}
		return "modéré", "Faisable, mais la charge mensuelle absorbe plus de la moitié de ton épargne actuelle. Garde un œil sur ton taux d'épargne."
	case in.OneTimeCost > 0 && in.Cash > 0 && int64(in.OneTimeCost)*2 > int64(in.Cash):
		return "modéré", "Le coût entame plus de la moitié de ton cash : ton fonds d'urgence en prend un coup. Reconstitue-le en priorité après l'achat."
	case impact.Scenarios[1].Delta10y >= 0:
		return "faible", "Cette décision améliore ta trajectoire : elle te rapporte plus qu'elle ne coûte à 10 ans. Fonce."
	default:
		return "faible", "Décision absorbable sans mettre le plan en danger : l'impact à long terme reste limité."
	}
}
