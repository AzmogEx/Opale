package engine

import "testing"

func TestContributionIsNotMarketGain(t *testing.T) {
	r, e := ComputeInvestmentPerformance(100000, 150000, []InvestmentCashFlow{{"contribution", 50000}}, true)
	if e != nil || r.Gain != 0 || r.ReturnBps == nil || *r.ReturnBps != 0 {
		t.Fatal(r, e)
	}
}
func TestIncompleteHistoryDoesNotInventReturn(t *testing.T) {
	r, e := ComputeInvestmentPerformance(100000, 150000, nil, false)
	if e != nil || r.Known || r.ReturnBps != nil {
		t.Fatal(r, e)
	}
}
func TestWithdrawalDistributionAndExternalFee(t *testing.T) {
	r, e := ComputeInvestmentPerformance(100000, 90000, []InvestmentCashFlow{{"withdrawal", 20000}, {"distribution", 1000}, {"fee", 500}}, true)
	if e != nil || r.Gain != 10500 {
		t.Fatal(r, e)
	}
}
