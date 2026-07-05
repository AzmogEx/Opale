package engine

import (
	"testing"

	"github.com/opale-app/opale/internal/money"
)

// euros converts whole euros to cents — keeps test literals readable.
func euros(e int64) money.Cents { return money.Cents(e * 100) }

func findRisk(risks []Risk, id string) *Risk {
	for i := range risks {
		if risks[i].ID == id {
			return &risks[i]
		}
	}
	return nil
}

func TestDetectRisksHealthySituation(t *testing.T) {
	// Situation saine : 6 mois de réserve, 2 revenus, dettes faibles,
	// patrimoine diversifié → aucun risque.
	risks := DetectRisks(RiskInputs{
		Cash:          euros(12_000), // 6 mois de dépenses
		Assets:        euros(100_000),
		Liabilities:   euros(5_000), // 5 % des actifs
		Income3M:      euros(9_600),
		Expenses3M:    euros(6_000), // 2 000 €/mois
		FixedMonthly:  euros(1_000), // ~31 % des revenus
		IncomeSources: 2,
		AssetKindValues: map[string]money.Cents{
			"checking": euros(12_000), "savings": euros(30_000),
			"stocks": euros(38_000), "crypto": euros(20_000),
		},
	})
	if len(risks) != 0 {
		t.Fatalf("situation saine : aucun risque attendu, obtenu %+v", risks)
	}
}

func TestDetectRisksCriticalStack(t *testing.T) {
	// Situation dégradée : épargne négative, pas de réserve, dettes lourdes,
	// un seul revenu, charges écrasantes, découvert prévu.
	risks := DetectRisks(RiskInputs{
		Cash:             euros(500), // < 1 mois
		Assets:           euros(10_000),
		Liabilities:      euros(8_000), // 80 % des actifs
		Income3M:         euros(6_000),
		Expenses3M:       euros(7_500), // 2 500 €/mois > revenus
		FixedMonthly:     euros(1_700), // 85 % des revenus
		IncomeSources:    1,
		ProjectedCash30d: euros(-200),
		HasProjection:    true,
		AssetKindValues:  map[string]money.Cents{"checking": euros(500)},
	})

	for _, id := range []string{"negative_savings", "emergency_fund", "debt", "single_income", "fixed_costs", "low_cash_ahead"} {
		r := findRisk(risks, id)
		if r == nil {
			t.Fatalf("risque %q attendu, absent de %+v", id, risks)
		}
		if r.Severity != SeverityCritical && id != "single_income" {
			t.Errorf("risque %q : sévérité %q, attendu critical", id, r.Severity)
		}
	}
	// Le tri met les critiques d'abord.
	if risks[0].Severity != SeverityCritical {
		t.Errorf("premier risque %q, attendu un critique", risks[0].Severity)
	}
}

func TestDetectRisksIdleCashAndConcentration(t *testing.T) {
	// 40 000 € de cash pour 1 000 €/mois de dépenses (40 mois) et 50 % du
	// patrimoine → cash dormant. Immobilier 80 % → concentration + illiquidité.
	risks := DetectRisks(RiskInputs{
		Cash:          euros(40_000),
		Assets:        euros(80_000),
		Income3M:      euros(9_000),
		Expenses3M:    euros(3_000),
		IncomeSources: 2,
		AssetKindValues: map[string]money.Cents{
			"real_estate": euros(64_000), // 80 %
			"checking":    euros(16_000),
		},
	})
	if findRisk(risks, "idle_cash") == nil {
		t.Errorf("cash dormant attendu : %+v", risks)
	}
	if findRisk(risks, "concentration") == nil {
		t.Errorf("concentration attendue : %+v", risks)
	}
	if findRisk(risks, "illiquidity") == nil {
		t.Errorf("illiquidité attendue : %+v", risks)
	}
}

func TestEvaluateDecisionPureCost(t *testing.T) {
	// Achat 12 000 € cash, sans charge mensuelle. Rendement 0 % pour un
	// calcul de tête : à 5 et 10 ans, l'écart est exactement le coût.
	impact, err := EvaluateDecision(DecisionInputs{
		NetWorth:        euros(50_000),
		Cash:            euros(20_000),
		MonthlySavings:  euros(500),
		MonthlyExpenses: euros(2_000),
		AnnualReturnBps: 0,
		SwrBps:          400,
		OneTimeCost:     euros(12_000),
	})
	if err != nil {
		t.Fatal(err)
	}
	normal := impact.Scenarios[1]
	if normal.Delta5y != euros(-12_000) || normal.Delta10y != euros(-12_000) {
		t.Errorf("à 0 %% de rendement, l'écart doit être le coût : delta5y=%d delta10y=%d",
			normal.Delta5y, normal.Delta10y)
	}
	if impact.NetWorthAfter != euros(38_000) {
		t.Errorf("patrimoine après : %d, attendu %d", impact.NetWorthAfter, euros(38_000))
	}
	if !impact.AffordableCash {
		t.Error("12 000 € tiennent dans 20 000 € de cash")
	}
	if !impact.Scenarios[1].BaselineReached {
		// À 0 %, 500 €/mois vers une cible de 600 000 € (2 000 × 12 × 25)
		// depuis 50 000 € : (600000-50000)/500 = 1100 mois < 1200 → atteint.
		t.Error("la trajectoire de référence devrait atteindre la cible dans la fenêtre")
	}
	if normal.DelayMonths <= 0 {
		t.Errorf("un coût sec doit retarder l'indépendance : delay=%d", normal.DelayMonths)
	}
	// 12 000 € à 500 €/mois de rattrapage = 24 mois de retard exactement (0 %).
	if normal.DelayMonths != 24 {
		t.Errorf("retard attendu 24 mois à 0 %%, obtenu %d", normal.DelayMonths)
	}
	if impact.RiskLevel != "modéré" {
		// 12k > 20k/2 → entame plus de la moitié du cash.
		t.Errorf("risque attendu modéré, obtenu %q", impact.RiskLevel)
	}
}

func TestEvaluateDecisionUnaffordable(t *testing.T) {
	impact, err := EvaluateDecision(DecisionInputs{
		NetWorth:        euros(30_000),
		Cash:            euros(5_000),
		MonthlySavings:  euros(400),
		MonthlyExpenses: euros(1_800),
		AnnualReturnBps: 500,
		SwrBps:          400,
		OneTimeCost:     euros(15_000), // > cash
	})
	if err != nil {
		t.Fatal(err)
	}
	if impact.AffordableCash {
		t.Error("15 000 € ne tiennent pas dans 5 000 € de cash")
	}
	if impact.RiskLevel != "élevé" {
		t.Errorf("risque attendu élevé, obtenu %q", impact.RiskLevel)
	}
}

func TestEvaluateDecisionSavingsKiller(t *testing.T) {
	// Charge mensuelle qui dépasse l'épargne → épargne négative → risque élevé.
	impact, err := EvaluateDecision(DecisionInputs{
		NetWorth:        euros(40_000),
		Cash:            euros(10_000),
		MonthlySavings:  euros(300),
		MonthlyExpenses: euros(2_000),
		AnnualReturnBps: 500,
		SwrBps:          400,
		MonthlyCost:     euros(450),
	})
	if err != nil {
		t.Fatal(err)
	}
	if impact.SavingsAfter != euros(-150) {
		t.Errorf("épargne après : %d, attendu %d", impact.SavingsAfter, euros(-150))
	}
	if impact.RiskLevel != "élevé" {
		t.Errorf("risque attendu élevé, obtenu %q", impact.RiskLevel)
	}
}

func TestEvaluateDecisionImprovement(t *testing.T) {
	// Une économie mensuelle (MonthlyCost négatif) améliore la trajectoire.
	impact, err := EvaluateDecision(DecisionInputs{
		NetWorth:        euros(50_000),
		Cash:            euros(15_000),
		MonthlySavings:  euros(500),
		MonthlyExpenses: euros(2_000),
		AnnualReturnBps: 500,
		SwrBps:          400,
		MonthlyCost:     euros(-200), // 200 €/mois d'économie
	})
	if err != nil {
		t.Fatal(err)
	}
	normal := impact.Scenarios[1]
	if normal.Delta10y <= 0 {
		t.Errorf("une économie mensuelle doit améliorer le patrimoine à 10 ans : %d", normal.Delta10y)
	}
	if impact.RiskLevel != "faible" {
		t.Errorf("risque attendu faible, obtenu %q", impact.RiskLevel)
	}
	if normal.DelayMonths >= 0 && normal.BaselineReached && normal.DecisionReached {
		if normal.DelayMonths > 0 {
			t.Errorf("une économie ne doit pas retarder l'indépendance : %d", normal.DelayMonths)
		}
	}
}

func TestEvaluateDecisionScenarioOrdering(t *testing.T) {
	impact, err := EvaluateDecision(DecisionInputs{
		NetWorth:        euros(60_000),
		Cash:            euros(20_000),
		MonthlySavings:  euros(600),
		MonthlyExpenses: euros(2_200),
		AnnualReturnBps: 500,
		SwrBps:          400,
		OneTimeCost:     euros(5_000),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(impact.Scenarios) != 3 {
		t.Fatalf("3 scénarios attendus, obtenu %d", len(impact.Scenarios))
	}
	names := []string{"prudent", "normal", "ambitieux"}
	for i, s := range impact.Scenarios {
		if s.Name != names[i] {
			t.Errorf("scénario %d : %q, attendu %q", i, s.Name, names[i])
		}
	}
	// Plus le rendement est haut, plus le patrimoine à 10 ans est haut.
	if !(impact.Scenarios[0].In10y < impact.Scenarios[1].In10y &&
		impact.Scenarios[1].In10y < impact.Scenarios[2].In10y) {
		t.Errorf("ordre des scénarios incohérent : %d < %d < %d attendu",
			impact.Scenarios[0].In10y, impact.Scenarios[1].In10y, impact.Scenarios[2].In10y)
	}
}
