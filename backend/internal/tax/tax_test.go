package tax

import (
	"testing"

	"github.com/opale-app/opale/internal/money"
)

func euros(e int64) money.Cents { return money.Cents(e * 100) }

func TestComputeSinglePart(t *testing.T) {
	// 40 000 € imposable, 1 part :
	// tranche 0 %  : 11 600 → 0
	// tranche 11 % : 29 579 − 11 600 = 17 979 → 1 977,69
	// tranche 30 % : 40 000 − 29 579 = 10 421 → 3 126,30
	// total ≈ 5 103,99 €
	est := Compute(euros(40_000), 10)
	if est.MarginalRateBps != 3_000 {
		t.Fatalf("TMI %d bps, attendu 3000 (30 %%)", est.MarginalRateBps)
	}
	if est.Tax != money.Cents(510399) {
		t.Fatalf("impôt %d, attendu 510399", est.Tax)
	}
	if est.NetIncome != euros(40_000)-est.Tax {
		t.Fatal("net incohérent")
	}
}

func TestComputeFamilyQuotient(t *testing.T) {
	// Même revenu, 2,5 parts : quotient 16 000 €/part → TMI 11 %.
	est := Compute(euros(40_000), 25)
	if est.MarginalRateBps != 1_100 {
		t.Fatalf("TMI %d bps, attendu 1100", est.MarginalRateBps)
	}
	// L'impôt doit être nettement plus faible qu'à 1 part.
	single := Compute(euros(40_000), 10)
	if est.Tax >= single.Tax {
		t.Fatal("le quotient familial doit réduire l'impôt")
	}
}

func TestComputeZeroAndLow(t *testing.T) {
	if tax := Compute(euros(10_000), 10).Tax; tax != 0 {
		t.Fatalf("sous la première tranche : impôt %d, attendu 0", tax)
	}
	if tax := Compute(0, 10).Tax; tax != 0 {
		t.Fatal("revenu nul : impôt nul")
	}
}

func TestPEREffect(t *testing.T) {
	// À 40 000 € (TMI 30 %), 2 000 € de PER ≈ 600 € d'économie.
	effect := ComputePEREffect(euros(40_000), euros(2_000), 10)
	if effect.Savings < euros(590) || effect.Savings > euros(610) {
		t.Fatalf("économie %d, attendu ≈ 60000", effect.Savings)
	}
	if effect.RealCost != effect.Contribution-effect.Savings {
		t.Fatal("coût réel incohérent")
	}
}
