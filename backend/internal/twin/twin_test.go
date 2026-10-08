package twin

import (
	"strings"
	"testing"

	"github.com/opale-app/opale/internal/engine"
	"github.com/opale-app/opale/internal/money"
)

func sample() Snapshot {
	return Snapshot{
		Complete:    true,
		NetWorth:    money.Cents(6_030_001), // 60 300,01 €
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

// Outbound privacy is tested at the actual HTTP provider in ai/cloud_http_test.go.
func TestDescribeKeepsExactAmounts(t *testing.T) {
	out := Describe(sample())
	for _, want := range []string{"60300.01 €", "42300.00 €", "3200.00 €", "Achat immobilier"} {
		if !strings.Contains(out, want) {
			t.Errorf("attendu %q dans le contexte homelab :\n%s", want, out)
		}
	}
}
