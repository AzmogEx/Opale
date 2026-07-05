package twin

import (
	"strings"
	"testing"

	"github.com/opale-app/opale/internal/engine"
	"github.com/opale-app/opale/internal/money"
)

func sample() Snapshot {
	return Snapshot{
		NetWorth:    money.Cents(6_030_000), // 60 300 €
		Assets:      money.Cents(6_530_000),
		Liabilities: money.Cents(500_000),
		Cash:        money.Cents(4_230_000), // 42 300 €
		AssetKinds: map[string]money.Cents{
			"checking": money.Cents(4_230_000),
			"stocks":   money.Cents(1_800_000), // 18 000 €
		},
		MonthlyIncome:   money.Cents(320_000), // 3 200 €
		MonthlyExpenses: money.Cents(240_000),
		FixedMonthly:    money.Cents(140_000), // 1 400 €
		MonthlySavings:  money.Cents(80_000),
		SavingsRateBps:  2_500,
		Health:          engine.HealthScore{Score: 86},
		Goals: []Goal{
			{Name: "Achat immobilier", Target: money.Cents(25_000_000), Percent: 12},
		},
	}
}

func TestCompactKRounding(t *testing.T) {
	cases := []struct {
		cents money.Cents
		want  string
	}{
		{money.Cents(4_230_000), "42k"},   // 42 300 € → au millier
		{money.Cents(320_000), "3.2k"},    // 3 200 € → à la centaine
		{money.Cents(25_000_000), "250k"}, // 250 000 €
		{money.Cents(84_700), "850"},      // 847 € → à la dizaine
		{money.Cents(-150_000), "-1.5k"},
		{money.Cents(1_000_000), "10k"},
	}
	for _, c := range cases {
		if got := compactK(c.cents); got != c.want {
			t.Errorf("compactK(%d) = %q, attendu %q", c.cents, got, c.want)
		}
	}
}

func TestAnonymizeNeverLeaksExactAmounts(t *testing.T) {
	out := Anonymize(sample())

	// Le format du cahier des charges (§6.5) : profil anonyme, montants en k.
	if !strings.HasPrefix(out, "Profil A :") {
		t.Errorf("le contexte anonymisé doit commencer par « Profil A : » :\n%s", out)
	}
	for _, want := range []string{"cash 42k", "placements actions 18k", "revenu mensuel 3.2k", "charges fixes 1.4k", "objectif achat immobilier 250k"} {
		if !strings.Contains(out, want) {
			t.Errorf("attendu %q dans :\n%s", want, out)
		}
	}
	// Aucun montant exact ne doit fuiter (EIA-033).
	for _, leak := range []string{"42300", "42 300", "3200", "3 200", "4230000"} {
		if strings.Contains(out, leak) {
			t.Errorf("montant exact %q dans le contexte anonymisé :\n%s", leak, out)
		}
	}
}

func TestDescribeKeepsExactAmounts(t *testing.T) {
	out := Describe(sample())
	for _, want := range []string{"60300 €", "42300 €", "3200 €", "Achat immobilier"} {
		if !strings.Contains(out, want) {
			t.Errorf("attendu %q dans le contexte homelab :\n%s", want, out)
		}
	}
}
