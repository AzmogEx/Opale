package engine

import "github.com/opale-app/opale/internal/money"

// Simulateur de crédit immobilier — mensualité, amortissement, capacité.
// Arithmétique entière au centime, intérêts arrondis mensuellement vers zéro.
// Même échéancier et convention que les comparaisons de décisions.

// LoanYear — le résumé d'une année d'amortissement.
type LoanYear struct {
	Year      int         `json:"year"`
	Interest  money.Cents `json:"interest_cents"`  // intérêts payés dans l'année
	Principal money.Cents `json:"principal_cents"` // capital remboursé dans l'année
	Remaining money.Cents `json:"remaining_cents"` // restant dû fin d'année
}

// LoanResult — la simulation complète d'un prêt.
type LoanResult struct {
	MonthlyPayment money.Cents `json:"monthly_payment_cents"`
	TotalPaid      money.Cents `json:"total_paid_cents"`
	TotalInterest  money.Cents `json:"total_interest_cents"`
	Schedule       []LoanYear  `json:"schedule"`
}

// SimulateLoan calcule la mensualité arrondie vers le haut qui amortit la
// dette, et plafonne le dernier paiement au capital restant plus intérêts.
func SimulateLoan(principal money.Cents, annualRateBps, months int) (LoanResult, error) {
	if principal <= 0 || principal > 1_000_000_000_000 {
		return LoanResult{}, ErrInvalidInput
	}
	payment, err := LoanPayment(principal, annualRateBps, months)
	if err != nil {
		return LoanResult{}, err
	}
	result := LoanResult{MonthlyPayment: payment}
	balance := principal
	var interestYear, principalYear money.Cents
	for m := 1; m <= months; m++ {
		interest, err := ratioMoney(balance, int64(annualRateBps), 120000)
		if err != nil {
			return LoanResult{}, err
		}
		paid := min(payment, balance+interest)
		capital := paid - interest
		balance -= capital
		interestYear += interest
		principalYear += capital
		result.TotalPaid += paid
		result.TotalInterest += interest
		if m%12 == 0 || m == months {
			result.Schedule = append(result.Schedule, LoanYear{Year: (m + 11) / 12, Interest: interestYear, Principal: principalYear, Remaining: balance})
			interestYear = 0
			principalYear = 0
		}
	}
	return result, nil
}

// LoanCapacity — combien on peut emprunter avec une mensualité donnée.
func LoanCapacity(monthlyPayment money.Cents, annualRateBps, months int) (money.Cents, error) {
	if monthlyPayment <= 0 || monthlyPayment > 1_000_000_000 || months <= 0 || months > 600 ||
		annualRateBps < 0 || annualRateBps > 2_000 {
		return 0, ErrInvalidInput
	}
	// Dichotomie sur le principal : le plus grand qui tient dans la mensualité.
	low, high := money.Cents(0), money.Cents(int64(monthlyPayment)*int64(months))
	for low < high {
		mid := (low + high + 1) / 2
		sim, err := SimulateLoan(mid, annualRateBps, months)
		if err != nil {
			return 0, err
		}
		if sim.MonthlyPayment <= monthlyPayment {
			low = mid
		} else {
			high = mid - 1
		}
	}
	return low, nil
}
