package engine

import (
	"testing"

	"github.com/opale-app/opale/internal/money"
)

func TestSimulateLoanKnownValue(t *testing.T) {
	// 200 000 € à 3,5 % sur 25 ans : mensualité de référence ≈ 1 001,25 €.
	res, err := SimulateLoan(money.Cents(20_000_000), 350, 300)
	if err != nil {
		t.Fatal(err)
	}
	if res.MonthlyPayment < money.Cents(99_900) || res.MonthlyPayment > money.Cents(100_300) {
		t.Fatalf("mensualité %d, attendu ≈ 100125", res.MonthlyPayment)
	}
	// Coût total = principal + intérêts (≈ 100 000 € d'intérêts).
	if res.TotalInterest < money.Cents(9_800_000) || res.TotalInterest > money.Cents(10_300_000) {
		t.Fatalf("intérêts %d, attendu ≈ 10 M centimes", res.TotalInterest)
	}
	if len(res.Schedule) != 25 {
		t.Fatalf("25 lignes annuelles attendues, obtenu %d", len(res.Schedule))
	}
	// Fin de tableau : tout est remboursé.
	if last := res.Schedule[len(res.Schedule)-1].Remaining; last != 0 {
		t.Fatalf("restant dû final %d, attendu 0", last)
	}
}

func TestSimulateLoanZeroRate(t *testing.T) {
	// Taux 0 : mensualité = principal / mois, zéro intérêt.
	res, err := SimulateLoan(money.Cents(1_200_000), 0, 12)
	if err != nil {
		t.Fatal(err)
	}
	if res.MonthlyPayment != money.Cents(100_000) {
		t.Fatalf("mensualité %d, attendu 100000", res.MonthlyPayment)
	}
	if res.TotalInterest != 0 {
		t.Fatalf("intérêts %d, attendu 0", res.TotalInterest)
	}
}

func TestLoanCapacityRoundTrip(t *testing.T) {
	// La capacité pour la mensualité d'un prêt connu ≈ son principal.
	res, _ := SimulateLoan(money.Cents(20_000_000), 350, 300)
	capacity, err := LoanCapacity(res.MonthlyPayment, 350, 300)
	if err != nil {
		t.Fatal(err)
	}
	diff := int64(capacity) - 20_000_000
	if diff < -10_000 || diff > 10_000 { // ± 100 €
		t.Fatalf("capacité %d, attendu ≈ 20000000", capacity)
	}
}
