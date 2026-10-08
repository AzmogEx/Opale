package engine

import "github.com/opale-app/opale/internal/money"

type InvestmentCashFlow struct {
	Kind   string
	Amount money.Cents
}
type InvestmentPerformance struct {
	InitialCapital money.Cents `json:"initial_capital_cents"`
	Contributions  money.Cents `json:"contributions_cents"`
	Withdrawals    money.Cents `json:"withdrawals_cents"`
	Distributions  money.Cents `json:"distributions_cents"`
	Fees           money.Cents `json:"fees_cents"`
	Gain           money.Cents `json:"gain_cents"`
	ReturnBps      *int64      `json:"return_bps,omitempty"`
	Known          bool        `json:"known"`
	Reason         string      `json:"reason"`
}

// Cash-on-capital return is cumulative, not annualized or time weighted.
// Distributions/fees are flows OUTSIDE the quoted portfolio value. Charges
// already netted into a valuation must not be entered again as external fees.
func ComputeInvestmentPerformance(first, last money.Cents, flows []InvestmentCashFlow, complete bool) (InvestmentPerformance, error) {
	out := InvestmentPerformance{InitialCapital: first, Known: complete, Reason: "Rendement cumulé sur capital initial et apports, non annualisé ; flux externes après la première valorisation"}
	for _, f := range flows {
		if f.Amount <= 0 {
			return out, ErrInvalidInput
		}
		var p *money.Cents
		switch f.Kind {
		case "contribution":
			p = &out.Contributions
		case "withdrawal":
			p = &out.Withdrawals
		case "distribution":
			p = &out.Distributions
		case "fee":
			p = &out.Fees
		default:
			return out, ErrInvalidInput
		}
		v, e := money.Add(*p, f.Amount)
		if e != nil {
			return out, e
		}
		*p = v
	}
	if !complete {
		out.Reason = "Historique des apports/retraits non confirmé : performance indéterminée"
		return out, nil
	}
	gain, e := money.Sub(last, first)
	if e != nil {
		return out, e
	}
	for _, v := range []money.Cents{-out.Contributions, out.Withdrawals, out.Distributions, -out.Fees} {
		gain, e = money.Add(gain, v)
		if e != nil {
			return out, e
		}
	}
	out.Gain = gain
	denominator, e := money.Add(first, out.Contributions)
	if e != nil {
		return out, e
	}
	if denominator > 0 {
		rate, e := ratioMoney(gain, 10000, int64(denominator))
		if e != nil {
			return out, e
		}
		n := int64(rate)
		out.ReturnBps = &n
	} else {
		out.Reason = "Capital de référence nul : taux indéterminé"
	}
	return out, nil
}
