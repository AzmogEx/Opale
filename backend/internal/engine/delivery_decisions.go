package engine

import (
	"github.com/opale-app/opale/internal/money"
	"math/big"
)

// ComparisonInput compares alternatives with the same initial liquid capital
// and monthly budget BEFORE housing/debt costs. Asset price is not a expense:
// it replaces cash, and only fees/interest/market movements change net wealth.
type ComparisonInput struct {
	Kind                 string      `json:"kind"`
	HorizonMonths        int         `json:"horizon_months"`
	InitialCash          money.Cents `json:"initial_cash_cents"`
	MonthlyBudget        money.Cents `json:"monthly_budget_cents"`
	AssetPrice           money.Cents `json:"asset_price_cents"`
	DownPayment          money.Cents `json:"down_payment_cents"`
	PurchaseFees         money.Cents `json:"purchase_fees_cents"`
	MonthlyRent          money.Cents `json:"monthly_rent_cents"`
	MonthlyOwnershipCost money.Cents `json:"monthly_ownership_cost_cents"`
	LoanPrincipal        money.Cents `json:"loan_principal_cents"`
	LoanRateBps          int         `json:"loan_rate_bps"`
	LoanMonths           int         `json:"loan_months"`
	Repayment            money.Cents `json:"repayment_cents"`
	RepaymentFee         money.Cents `json:"repayment_fee_cents"`
	InvestmentReturnBps  int         `json:"investment_return_bps"`
	AssetGrowthBps       int         `json:"asset_growth_bps"`
	RentGrowthBps        int         `json:"rent_growth_bps"`
	InflationBps         int         `json:"inflation_bps"`
	SaleFeeBps           int         `json:"sale_fee_bps"`
}
type Alternative struct {
	Label          string      `json:"label"`
	InitialOutflow money.Cents `json:"initial_outflow_cents"`
	MonthlyCost    money.Cents `json:"monthly_cost_cents"`
	FinalLiquid    money.Cents `json:"final_liquid_cents"`
	AssetValue     money.Cents `json:"asset_value_cents"`
	RemainingDebt  money.Cents `json:"remaining_debt_cents"`
	TotalInterest  money.Cents `json:"total_interest_cents"`
	TotalFees      money.Cents `json:"total_fees_cents"`
	FinalNet       money.Cents `json:"final_net_cents"`
	RealFinalNet   money.Cents `json:"real_final_net_cents"`
	MinimumLiquid  money.Cents `json:"minimum_liquid_cents"`
	Feasible       bool        `json:"feasible"`
}
type ComparisonResult struct {
	Kind           string                   `json:"kind"`
	HorizonMonths  int                      `json:"horizon_months"`
	A              Alternative              `json:"a"`
	B              Alternative              `json:"b"`
	Delta          money.Cents              `json:"delta_b_minus_a_cents"`
	Assumptions    []string                 `json:"assumptions"`
	Sensitivity    []ComparisonSensitivity  `json:"sensitivity"`
	Timeline       []ComparisonCheckpoint   `json:"timeline"`
	Scenarios      []ComparisonScenario     `json:"scenarios"`
	Recommendation ComparisonRecommendation `json:"recommendation"`
	Risks          []string                 `json:"risks"`
}
type ComparisonSensitivity struct {
	InvestmentReturnBps int         `json:"investment_return_bps"`
	Delta               money.Cents `json:"delta_b_minus_a_cents"`
}

func ratioMoney(v money.Cents, numerator, denominator int64) (money.Cents, error) {
	b := new(big.Int).Mul(big.NewInt(int64(v)), big.NewInt(numerator))
	b.Quo(b, big.NewInt(denominator))
	if !b.IsInt64() {
		return 0, money.ErrOverflow
	}
	return money.Cents(b.Int64()), nil
}

// LoanPayment derives the cent-exact amortizing payment, rounding monthly
// interest down to a cent. Last payment is capped at debt plus interest.
func LoanPayment(principal money.Cents, rate, months int) (money.Cents, error) {
	if principal < 0 || principal > 1_000_000_000_000 || rate < 0 || rate > 2000 || months < 1 || months > 600 {
		return 0, ErrInvalidInput
	}
	if principal == 0 {
		return 0, nil
	}
	interest, e := ratioMoney(principal, int64(rate), 120000)
	if e != nil {
		return 0, e
	}
	high, e := money.Add(principal, interest)
	if e != nil {
		return 0, e
	}
	low := money.Cents(0)
	for low < high {
		mid := low + (high-low)/2
		balance := principal
		for m := 0; m < months && balance > 0; m++ {
			i, e := ratioMoney(balance, int64(rate), 120000)
			if e != nil {
				return 0, e
			}
			due, e := money.Add(balance, i)
			if e != nil {
				return 0, e
			}
			balance = due - mid
		}
		if balance > 0 {
			low = mid + 1
		} else {
			high = mid
		}
	}
	return low, nil
}
func CompareDecisions(in ComparisonInput) (ComparisonResult, error) {
	if in.HorizonMonths < 1 {
		return ComparisonResult{}, ErrInvalidInput
	}
	out, e := compareCore(in)
	if e != nil {
		return out, e
	}
	for i, rate := range []int{max(-5000, in.InvestmentReturnBps-200), in.InvestmentReturnBps, min(2000, in.InvestmentReturnBps+200)} {
		alt := in
		alt.InvestmentReturnBps = rate
		r, e := compareCore(alt)
		if e != nil {
			return out, e
		}
		out.Scenarios = append(out.Scenarios, ComparisonScenario{Name: []string{"prudent", "normal", "ambitieux"}[i], InvestmentReturnBps: rate, A: r.A, B: r.B, Delta: r.Delta})
		if i != 1 {
			out.Sensitivity = append(out.Sensitivity, ComparisonSensitivity{rate, r.Delta})
		}
	}
	for _, months := range []int{0, 60, 120} {
		checkpoint := in
		checkpoint.HorizonMonths = months
		r, e := compareCore(checkpoint)
		if e != nil {
			return out, e
		}
		out.Timeline = append(out.Timeline, ComparisonCheckpoint{Months: months, A: r.A, B: r.B, Delta: r.Delta})
	}
	out.Assumptions = append(out.Assumptions,
		"Jalons à 0, 5 et 10 ans : mêmes hypothèses centrales, même durée contractuelle de prêt ; à 0, opérations initiales et frais inclus, aucun rendement ni mensualité écoulée. Les valeurs nettes incluent les frais de revente hypothétiques à chaque jalon.",
		"Scénarios prudent, normal et ambitieux à l’horizon demandé : seul le rendement annuel des liquidités placées varie de −200, 0 et +200 points de base, dans les bornes −50 % à +20 %. Ces noms ne désignent ni probabilités ni garanties ; les autres risques de marché ne sont pas modélisés.")
	out.Recommendation, out.Risks = recommendComparison(out)
	return out, nil
}
func compareCore(in ComparisonInput) (ComparisonResult, error) {
	valid := map[string]bool{"buy_rent": true, "cash_credit": true, "repay_invest": true}
	if !valid[in.Kind] || in.HorizonMonths < 0 || in.HorizonMonths > 600 || in.LoanMonths < 1 || in.LoanMonths > 600 || in.LoanRateBps < 0 || in.LoanRateBps > 2000 || in.InvestmentReturnBps < -5000 || in.InvestmentReturnBps > 2000 || in.AssetGrowthBps < -5000 || in.AssetGrowthBps > 2000 || in.RentGrowthBps < -5000 || in.RentGrowthBps > 2000 || in.InflationBps < 0 || in.InflationBps > 2000 || in.SaleFeeBps < 0 || in.SaleFeeBps > 10000 {
		return ComparisonResult{}, ErrInvalidInput
	}
	for _, v := range []money.Cents{in.InitialCash, in.MonthlyBudget, in.AssetPrice, in.DownPayment, in.PurchaseFees, in.MonthlyRent, in.MonthlyOwnershipCost, in.LoanPrincipal, in.Repayment, in.RepaymentFee} {
		if v < 0 || v > 1_000_000_000_000 {
			return ComparisonResult{}, ErrInvalidInput
		}
	}
	if in.DownPayment > in.AssetPrice || (in.Kind != "repay_invest" && in.AssetPrice == 0) || (in.Kind == "repay_invest" && (in.LoanPrincipal == 0 || in.Repayment == 0 || in.Repayment > in.LoanPrincipal)) {
		return ComparisonResult{}, ErrInvalidInput
	}
	type plan struct {
		label                                                string
		outflow, asset, debt, rent, ownership, fees, payment money.Cents
	}
	a, b := plan{}, plan{}
	switch in.Kind {
	case "buy_rent":
		a = plan{label: "Acheter", outflow: in.DownPayment + in.PurchaseFees, asset: in.AssetPrice, debt: in.AssetPrice - in.DownPayment, ownership: in.MonthlyOwnershipCost, fees: in.PurchaseFees}
		b = plan{label: "Louer", rent: in.MonthlyRent}
	case "cash_credit":
		a = plan{label: "Payer comptant", outflow: in.AssetPrice + in.PurchaseFees, asset: in.AssetPrice, ownership: in.MonthlyOwnershipCost, fees: in.PurchaseFees}
		b = plan{label: "Financer à crédit", outflow: in.DownPayment + in.PurchaseFees, asset: in.AssetPrice, debt: in.AssetPrice - in.DownPayment, ownership: in.MonthlyOwnershipCost, fees: in.PurchaseFees}
	case "repay_invest":
		a = plan{label: "Rembourser", outflow: in.Repayment + in.RepaymentFee, debt: in.LoanPrincipal - in.Repayment, fees: in.RepaymentFee}
		b = plan{label: "Investir", debt: in.LoanPrincipal}
	}
	var e error
	a.payment, e = LoanPayment(a.debt, in.LoanRateBps, in.LoanMonths)
	if e != nil {
		return ComparisonResult{}, e
	}
	b.payment, e = LoanPayment(b.debt, in.LoanRateBps, in.LoanMonths)
	if e != nil {
		return ComparisonResult{}, e
	}
	// Repayment reduces term while preserving the contractual payment.
	if in.Kind == "repay_invest" {
		a.payment = b.payment
	}
	run := func(p plan) (Alternative, error) {
		r := Alternative{Label: p.label, InitialOutflow: p.outflow, MonthlyCost: p.payment + p.rent + p.ownership, TotalFees: p.fees, FinalLiquid: in.InitialCash - p.outflow, AssetValue: p.asset, RemainingDebt: p.debt}
		r.MinimumLiquid = r.FinalLiquid
		for m := 0; m < in.HorizonMonths; m++ {
			liquidGrowth, e := ratioMoney(max(r.FinalLiquid, 0), int64(in.InvestmentReturnBps), 120000)
			if e != nil {
				return r, e
			}
			assetGrowth, e := ratioMoney(r.AssetValue, int64(in.AssetGrowthBps), 120000)
			if e != nil {
				return r, e
			}
			r.AssetValue, e = money.Add(r.AssetValue, assetGrowth)
			if e != nil {
				return r, e
			}
			interest, e := ratioMoney(r.RemainingDebt, int64(in.LoanRateBps), 120000)
			if e != nil {
				return r, e
			}
			payment := min(p.payment, r.RemainingDebt+interest)
			r.RemainingDebt -= payment - interest
			r.TotalInterest += interest
			cash, e := money.Add(r.FinalLiquid, liquidGrowth)
			if e != nil {
				return r, e
			}
			cash, e = money.Add(cash, in.MonthlyBudget-payment-p.rent-p.ownership)
			if e != nil {
				return r, e
			}
			r.FinalLiquid = cash
			r.MinimumLiquid = min(r.MinimumLiquid, cash)
			rentGrowth, e := ratioMoney(p.rent, int64(in.RentGrowthBps), 120000)
			if e != nil {
				return r, e
			}
			p.rent += rentGrowth
		}
		saleFees, e := ratioMoney(r.AssetValue, int64(in.SaleFeeBps), 10000)
		if e != nil {
			return r, e
		}
		r.TotalFees += saleFees
		r.FinalNet, e = money.Add(r.FinalLiquid, r.AssetValue)
		if e != nil {
			return r, e
		}
		r.FinalNet, e = money.Sub(r.FinalNet, r.RemainingDebt+saleFees)
		if e != nil {
			return r, e
		}
		r.RealFinalNet = r.FinalNet
		for m := 0; m < in.HorizonMonths; m++ {
			r.RealFinalNet, e = ratioMoney(r.RealFinalNet, 120000, 120000+int64(in.InflationBps))
			if e != nil {
				return r, e
			}
		}
		r.Feasible = r.MinimumLiquid >= 0
		return r, nil
	}
	ar, e := run(a)
	if e != nil {
		return ComparisonResult{}, e
	}
	br, e := run(b)
	if e != nil {
		return ComparisonResult{}, e
	}
	delta, e := money.Sub(br.FinalNet, ar.FinalNet)
	if e != nil {
		return ComparisonResult{}, e
	}
	return ComparisonResult{Kind: in.Kind, HorizonMonths: in.HorizonMonths, A: ar, B: br, Delta: delta, Assumptions: []string{"Capital liquide initial et budget mensuel avant logement/crédit identiques pour les deux alternatives.", "Rendements hypothétiques, intérêts mensuels taux annuel/12 ; aucun résultat garanti.", "Valeur terminale après frais de revente et capital restant dû. Fiscalité des gains et assurance de prêt à renseigner dans les coûts.", "Un solde liquide négatif signale un financement manquant, sans crédit fictif ajouté.", "Euros constants : déflation mensuelle selon l’inflation saisie. Remboursement anticipé : mensualité conservée, durée réduite."}}, nil
}
