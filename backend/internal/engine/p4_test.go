package engine

import (
	"testing"
	"time"

	"github.com/opale-app/opale/internal/money"
)

func day(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}

// ── Récurrence (EF-026) ─────────────────────────────────────────────────────

func TestDetectRecurringMonthly(t *testing.T) {
	txs := []TxObs{
		{"NETFLIX", "Netflix", day("2026-04-03"), -1399},
		{"NETFLIX", "Netflix", day("2026-05-03"), -1399},
		{"NETFLIX", "Netflix", day("2026-06-03"), -1399},
		{"NETFLIX", "Netflix", day("2026-07-03"), -1399},
	}
	flows := DetectRecurring(txs, day("2026-07-05"))
	if len(flows) != 1 {
		t.Fatalf("attendu 1 flux, obtenu %d", len(flows))
	}
	f := flows[0]
	if f.Periodicity != "monthly" || !f.Active {
		t.Fatalf("attendu mensuel actif, obtenu %+v", f)
	}
	if f.Amount != money.Cents(-1399) {
		t.Fatalf("montant %d", f.Amount)
	}
	if f.NextDate != day("2026-08-02") && f.NextDate != day("2026-08-03") {
		t.Fatalf("prochaine échéance inattendue : %s", f.NextDate)
	}
}

func TestDetectRecurringIgnoresIrregular(t *testing.T) {
	txs := []TxObs{
		{"CARREFOUR", "Carrefour", day("2026-06-01"), -4520},
		{"CARREFOUR", "Carrefour", day("2026-06-04"), -1230},
		{"CARREFOUR", "Carrefour", day("2026-06-19"), -8730},
		{"CARREFOUR", "Carrefour", day("2026-07-02"), -2100},
	}
	if flows := DetectRecurring(txs, day("2026-07-05")); len(flows) != 0 {
		t.Fatalf("les courses irrégulières ne sont pas un abonnement : %+v", flows)
	}
}

func TestDetectRecurringLapsed(t *testing.T) {
	// Dernière occurrence en mars, on est en juillet : résilié.
	txs := []TxObs{
		{"SPOTIFY", "Spotify", day("2026-02-10"), -1099},
		{"SPOTIFY", "Spotify", day("2026-03-10"), -1099},
	}
	flows := DetectRecurring(txs, day("2026-07-05"))
	if len(flows) != 1 || flows[0].Active {
		t.Fatalf("attendu flux inactif : %+v", flows)
	}
}

func TestDetectRecurringSalary(t *testing.T) {
	txs := []TxObs{
		{"SALAIRE ACME", "Salaire Acme", day("2026-05-01"), 340000},
		{"SALAIRE ACME", "Salaire Acme", day("2026-06-01"), 340000},
		{"SALAIRE ACME", "Salaire Acme", day("2026-07-01"), 340000},
	}
	flows := DetectRecurring(txs, day("2026-07-05"))
	if len(flows) != 1 || flows[0].Amount <= 0 {
		t.Fatalf("le salaire doit être détecté comme récurrent positif : %+v", flows)
	}
}

// ── Cashflow futur (EF-027) ─────────────────────────────────────────────────

func TestProjectCashDeterministic(t *testing.T) {
	recurring := []RecurringFlow{
		{Label: "Salaire", Amount: 340000, IntervalDays: 30, NextDate: day("2026-07-25"), Active: true},
		{Label: "Loyer", Amount: -80000, IntervalDays: 30, NextDate: day("2026-07-28"), Active: true},
		{Label: "Résilié", Amount: -9999, IntervalDays: 30, NextDate: day("2026-07-10"), Active: false},
	}
	p := ProjectCash(money.Cents(100000), recurring, day("2026-07-05"), day("2026-07-31"), 0)

	// 1 000 + 3 400 − 800 = 3 600 € ; le flux inactif est ignoré.
	if p.EndCash != money.Cents(360000) {
		t.Fatalf("EndCash = %d, attendu 360000", p.EndCash)
	}
	if len(p.Upcoming) != 2 {
		t.Fatalf("attendu 2 échéances, obtenu %d", len(p.Upcoming))
	}
	// Trié par date : salaire (25) avant loyer (28).
	if p.Upcoming[0].Label != "Salaire" {
		t.Fatalf("ordre du calendrier : %+v", p.Upcoming)
	}
}

func TestProjectCashVariableSpend(t *testing.T) {
	// 10 jours × 20 €/jour de dépenses variables = −200 €.
	p := ProjectCash(money.Cents(100000), nil, day("2026-07-05"), day("2026-07-15"), money.Cents(2000))
	if p.EndCash != money.Cents(80000) {
		t.Fatalf("EndCash = %d, attendu 80000", p.EndCash)
	}
}

// ── Score de santé (EF-015) ─────────────────────────────────────────────────

func TestHealthScoreHealthyProfile(t *testing.T) {
	score := ComputeHealthScore(HealthInputs{
		Income3M:     money.Cents(1_020_000), // 3 400 €/mois
		Expenses3M:   money.Cents(600_000),   // 2 000 €/mois → épargne 41 %
		Cash:         money.Cents(1_500_000), // 15 000 € → 7,5 mois
		Assets:       money.Cents(6_030_000),
		Liabilities:  money.Cents(300_000), // ~5 %
		FixedMonthly: money.Cents(100_000), // 1 000 € (~29 %)
		AssetKindValues: map[string]money.Cents{
			"checking": 1_000_000, "savings": 500_000,
			"pea": 2_000_000, "real_estate": 2_530_000,
		},
	})
	if score.Score < 90 {
		t.Fatalf("profil sain : score %d attendu ≥ 90 (%+v)", score.Score, score.Components)
	}
}

func TestHealthScoreStressedProfile(t *testing.T) {
	score := ComputeHealthScore(HealthInputs{
		Income3M:     money.Cents(600_000), // 2 000 €/mois
		Expenses3M:   money.Cents(660_000), // 2 200 €/mois → épargne négative
		Cash:         money.Cents(50_000),  // 500 € → 0,2 mois
		Assets:       money.Cents(1_000_000),
		Liabilities:  money.Cents(700_000),  // 70 %
		FixedMonthly: money.Cents(180_000),  // 1 800 € (90 %)
		AssetKindValues: map[string]money.Cents{
			"checking": 1_000_000, // 100 % concentré
		},
	})
	if score.Score > 20 {
		t.Fatalf("profil en difficulté : score %d attendu ≤ 20 (%+v)", score.Score, score.Components)
	}
}

func TestHealthScoreEmptyInputs(t *testing.T) {
	// Aucune donnée : le score doit rester calculable (pas de division par zéro).
	score := ComputeHealthScore(HealthInputs{})
	if score.Score < 0 || score.Score > 100 {
		t.Fatalf("score hors bornes : %d", score.Score)
	}
	if len(score.Components) != 5 {
		t.Fatalf("5 composantes attendues, obtenu %d", len(score.Components))
	}
}

func TestHealthScoreBounds(t *testing.T) {
	// La somme des maxima doit faire 100.
	score := ComputeHealthScore(HealthInputs{})
	total := 0
	for _, c := range score.Components {
		total += c.Max
	}
	if total != 100 {
		t.Fatalf("somme des maxima = %d, attendu 100", total)
	}
}
