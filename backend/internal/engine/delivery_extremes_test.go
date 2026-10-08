package engine

import (
	"errors"
	"github.com/opale-app/opale/internal/money"
	"math"
	"testing"
)

func TestDeliveryExtremeIndicatorsKeepThresholds(t *testing.T) {
	// Every input fits int64; multiplying it by 10000 used to reverse scores.
	large := money.Cents(math.MaxInt64)
	health := ComputeHealthScore(HealthInputs{Income3M: large, Expenses3M: large / 2, Cash: large, Assets: large, Liabilities: large, FixedMonthly: large, AssetKindValues: map[string]money.Cents{"real_estate": large, "cto": large / 10}})
	want := []int{25, 25, 0, 5, 0}
	for i, c := range health.Components {
		if c.Score != want[i] {
			t.Fatalf("component %s = %d want %d", c.Name, c.Score, want[i])
		}
	}
	risks := DetectRisks(RiskInputs{Cash: large, Assets: large, Liabilities: large, Income3M: large, Expenses3M: large / 2, FixedMonthly: large, AssetKindValues: map[string]money.Cents{"real_estate": large, "cto": large / 10}})
	for _, id := range []string{"debt", "fixed_costs", "concentration", "illiquidity"} {
		if findRisk(risks, id) == nil {
			t.Errorf("missing threshold risk %s: %+v", id, risks)
		}
	}
	if findRisk(risks, "emergency_fund") != nil {
		t.Fatal("overflow fabricated emergency shortage")
	}
	// Gross composition is meaningful even if its sum exceeds int64; no money
	// amount is emitted or saturated by a score calculation.
	divers := ComputeHealthScore(HealthInputs{AssetKindValues: map[string]money.Cents{"cto": large, "real_estate": large, "savings": large, "valuable": large}})
	if divers.Components[3].Score != 15 {
		t.Fatalf("large diversified assets %v", divers)
	}
}

func TestDeliveryRecurringExtremeAmounts(t *testing.T) {
	now := day("2026-03-05")
	for _, amount := range []money.Cents{math.MaxInt64, math.MinInt64} {
		obs := []TxObs{{MerchantKey: "stable", Label: "Synthetic", Amount: amount, OccurredOn: day("2026-01-01")}, {MerchantKey: "stable", Label: "Synthetic", Amount: amount, OccurredOn: day("2026-02-01")}}
		flows := DetectRecurring(obs, now)
		if len(flows) != 1 || flows[0].Amount != amount {
			t.Fatalf("stable large recurrence %+v", flows)
		}
	}
	// Values separated by far more than25% used to pass a wrapped diff*4.
	obs := []TxObs{{MerchantKey: "variable", Amount: 1, OccurredOn: day("2026-01-01")}, {MerchantKey: "variable", Amount: math.MaxInt64, OccurredOn: day("2026-02-01")}}
	if flows := DetectRecurring(obs, now); len(flows) != 0 {
		t.Fatalf("unstable accepted %+v", flows)
	}
}

func TestDeliveryCashProjectionRefusesUnrepresentableMoney(t *testing.T) {
	today := day("2026-01-01")
	flows := []RecurringFlow{{Amount: 1, NextDate: today, IntervalDays: 30, Active: true}}
	if _, e := ProjectCash(math.MaxInt64, flows, today, today, 0); !errors.Is(e, money.ErrOverflow) {
		t.Fatalf("cash overflow %v", e)
	}
	if _, e := ProjectCash(0, nil, today, today.AddDate(0, 0, 2), math.MaxInt64); !errors.Is(e, money.ErrOverflow) {
		t.Fatalf("variable expense overflow %v", e)
	}
	flows = append(flows, RecurringFlow{Amount: -1, NextDate: today, IntervalDays: 30, Active: true})
	if p, e := ProjectCash(math.MaxInt64, flows, today, today, 0); e != nil || p.EndCash != math.MaxInt64 {
		t.Fatalf("exact cancellation %v %v", p, e)
	}
}

func TestDeliveryDecisionRefusesOverflowBeforeScenarios(t *testing.T) {
	for _, in := range []DecisionInputs{
		{NetWorth: math.MaxInt64, OneTimeCost: -1, SwrBps: 400},
		{MonthlySavings: math.MinInt64, MonthlyCost: 1, SwrBps: 400},
		{MonthlyExpenses: math.MaxInt64, MonthlyCost: 1, SwrBps: 400},
	} {
		if _, e := EvaluateDecision(in); !errors.Is(e, money.ErrOverflow) {
			t.Fatalf("decision overflow %v for %+v", e, in)
		}
	}
	in := DecisionInputs{Cash: math.MaxInt64, OneTimeCost: math.MaxInt64/2 + 1}
	impact := DecisionImpact{AffordableCash: true, Scenarios: []DecisionScenario{{}, {}, {}}}
	if level, _ := decisionVerdict(in, impact); level != "modéré" {
		t.Fatalf("half-cash comparison overflow %s", level)
	}
}

func TestDeliveryEmergencyFundMilestoneRequiresKnownExpenses(t *testing.T) {
	for _, tc := range []struct {
		cash, expenses money.Cents
		want           bool
	}{
		{math.MaxInt64, 0, false}, {299, 150, false}, {300, 150, true}, {1, 1, false}, {2, 1, true},
		{math.MaxInt64, math.MaxInt64/2 + 1, false}, {math.MaxInt64, math.MaxInt64 / 2, true},
	} {
		score := ComputeHealthScore(HealthInputs{Cash: tc.cash, Expenses3M: tc.expenses})
		if score.EmergencyFundReady != tc.want {
			t.Fatalf("cash%d expenses%d ready%v want%v", tc.cash, tc.expenses, score.EmergencyFundReady, tc.want)
		}
	}
}
