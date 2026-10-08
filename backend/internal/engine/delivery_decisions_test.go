package engine

import (
	"testing"

	"github.com/opale-app/opale/internal/money"
)

func TestCashCreditNoInterestSameWealth(t *testing.T) {
	in := ComparisonInput{Kind: "cash_credit", HorizonMonths: 120, InitialCash: 20000000, MonthlyBudget: 200000, AssetPrice: 10000000, DownPayment: 2000000, LoanMonths: 120}
	r, e := CompareDecisions(in)
	if e != nil {
		t.Fatal(e)
	}
	if r.A.FinalNet != 44000000 || r.B.FinalNet != r.A.FinalNet || r.B.RemainingDebt != 0 {
		t.Fatalf("%+v", r)
	}
}
func TestBuyingRetainsAssetAndFees(t *testing.T) {
	r, e := CompareDecisions(ComparisonInput{Kind: "buy_rent", HorizonMonths: 1, InitialCash: 10000000, AssetPrice: 10000000, DownPayment: 10000000, PurchaseFees: 10000, LoanMonths: 120})
	if e != nil {
		t.Fatal(e)
	}
	if r.A.FinalNet != 9990000 || r.A.Feasible {
		t.Fatalf("asset double counted or missing funds hidden: %+v", r.A)
	}
}
func TestRepaymentReducesInterestAndInflationDeflates(t *testing.T) {
	r, e := CompareDecisions(ComparisonInput{Kind: "repay_invest", HorizonMonths: 120, InitialCash: 5000000, MonthlyBudget: 100000, LoanPrincipal: 8000000, LoanMonths: 120, LoanRateBps: 400, Repayment: 4000000, InflationBps: 200})
	if e != nil {
		t.Fatal(e)
	}
	if r.A.TotalInterest >= r.B.TotalInterest || r.A.RealFinalNet >= r.A.FinalNet {
		t.Fatalf("%+v", r)
	}
}
func TestLoanHighRateLongTermAmortizes(t *testing.T) {
	p, e := LoanPayment(10000000, 2000, 600)
	if e != nil || p < 160000 {
		t.Fatal(p, e)
	}

	r, e := SimulateLoan(10000000, 2000, 600)
	if e != nil || r.Schedule[len(r.Schedule)-1].Remaining != 0 || r.TotalPaid != 10000000+r.TotalInterest {
		t.Fatal(r, e)
	}
}

func TestDecisionTimelineConservesWealthAndInitialFees(t *testing.T) {
	in := ComparisonInput{Kind: "cash_credit", HorizonMonths: 24, InitialCash: 20000000, MonthlyBudget: 200000, AssetPrice: 10000000, DownPayment: 2000000, PurchaseFees: 100000, SaleFeeBps: 100, LoanMonths: 120, InflationBps: 200}
	r, err := CompareDecisions(in)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Timeline) != 3 {
		t.Fatal(r.Timeline)
	}
	for i, expected := range []struct {
		months int
		net    money.Cents
	}{{0, 19800000}, {60, 31800000}, {120, 43800000}} {
		p := r.Timeline[i]
		if p.Months != expected.months || p.A.FinalNet != expected.net || p.B.FinalNet != expected.net || p.Delta != 0 {
			t.Fatalf("checkpoint %d: %+v", i, p)
		}
		if p.A.TotalFees != 200000 || p.B.TotalFees != 200000 || p.A.TotalInterest != 0 || p.B.TotalInterest != 0 {
			t.Fatal(p)
		}
	}
	initial := r.Timeline[0]
	if initial.A.FinalLiquid != 9900000 || initial.B.FinalLiquid != 17900000 || initial.B.RemainingDebt != 8000000 || initial.B.RealFinalNet != initial.B.FinalNet {
		t.Fatalf("initial operations must not elapse a month: %+v", initial)
	}
	if r.Timeline[2].B.RemainingDebt != 0 || r.Timeline[1].A.RealFinalNet >= r.Timeline[1].A.FinalNet {
		t.Fatal(r.Timeline)
	}
	if r.HorizonMonths != 24 {
		t.Fatal("timeline replaced requested horizon")
	}
	// Repayment converts liquid capital into a lower debt, losing only its fee.
	r, err = CompareDecisions(ComparisonInput{Kind: "repay_invest", HorizonMonths: 12, InitialCash: 5000000, MonthlyBudget: 100000, LoanPrincipal: 8000000, LoanMonths: 120, LoanRateBps: 400, Repayment: 4000000, RepaymentFee: 10000})
	if err != nil {
		t.Fatal(err)
	}
	initial = r.Timeline[0]
	if initial.A.FinalNet != -3010000 || initial.B.FinalNet != -3000000 || initial.A.RemainingDebt != 4000000 || initial.A.TotalInterest != 0 {
		t.Fatal(initial)
	}
	// The public request still requires a positive horizon; only its initial checkpoint is zero.
	in.HorizonMonths = 0
	if _, err := CompareDecisions(in); err == nil {
		t.Fatal("zero request horizon accepted")
	}
}

func TestDecisionScenariosAndConditionalRecommendation(t *testing.T) {
	in := ComparisonInput{Kind: "cash_credit", HorizonMonths: 120, InitialCash: 20000000, MonthlyBudget: 200000, AssetPrice: 10000000, DownPayment: 2000000, LoanMonths: 120}
	r, err := CompareDecisions(in)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Scenarios) != 3 || len(r.Sensitivity) != 2 || len(r.Risks) == 0 {
		t.Fatal(r)
	}
	for i, name := range []string{"prudent", "normal", "ambitieux"} {
		if r.Scenarios[i].Name != name || r.Scenarios[i].InvestmentReturnBps != []int{-200, 0, 200}[i] {
			t.Fatal(r.Scenarios)
		}
	}
	if r.Scenarios[0].Delta >= 0 || r.Scenarios[1].Delta != 0 || r.Scenarios[2].Delta <= 0 || r.Recommendation.Preferred != "none" {
		t.Fatalf("winner reversal must not recommend: %+v", r)
	}
	if r.Scenarios[1].A != r.A || r.Scenarios[1].B != r.B {
		t.Fatal("normal scenario diverges from central result")
	}
	// When all feasible scenarios agree, the advice can express that conditional preference.
	in.InvestmentReturnBps = -1000
	r, err = CompareDecisions(in)
	if err != nil || r.Recommendation.Preferred != "a" || r.Recommendation.Message == "" {
		t.Fatal(r, err)
	}
	// Bounds are displayed honestly instead of inventing out-of-range assumptions.
	in.InvestmentReturnBps = 2000
	r, err = CompareDecisions(in)
	if err != nil || r.Scenarios[2].InvestmentReturnBps != 2000 {
		t.Fatal(r, err)
	}
}

func TestDecisionRecommendationNeverChoosesUnfundedAlternative(t *testing.T) {
	in := ComparisonInput{Kind: "cash_credit", HorizonMonths: 12, InitialCash: 200000, MonthlyBudget: 100000, AssetPrice: 1000000, DownPayment: 100000, LoanMonths: 120, LoanRateBps: 400, InvestmentReturnBps: 400}
	r, err := CompareDecisions(in)
	if err != nil || r.A.Feasible || !r.B.Feasible || r.Recommendation.Preferred != "b" {
		t.Fatal(r, err)
	}
	in.InitialCash = 0
	in.MonthlyBudget = 0
	r, err = CompareDecisions(in)
	if err != nil || r.A.Feasible || r.B.Feasible || r.Recommendation.Preferred != "none" {
		t.Fatal(r, err)
	}
}
