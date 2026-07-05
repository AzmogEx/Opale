package engine

import (
	"testing"

	"github.com/opale-app/opale/internal/money"
)

func TestProjectZeroRateZeroContribution(t *testing.T) {
	pts, err := Project(money.Cents(1_000_00), 0, 0, 12)
	if err != nil {
		t.Fatal(err)
	}
	if len(pts) != 13 {
		t.Fatalf("attendu 13 points, obtenu %d", len(pts))
	}
	for _, p := range pts {
		if p.Net != money.Cents(1_000_00) {
			t.Fatalf("mois %d : le patrimoine ne doit pas bouger, obtenu %d", p.Month, p.Net)
		}
	}
}

func TestProjectContributionOnly(t *testing.T) {
	// 0 € de départ, 100 €/mois, 0 % : après 12 mois → 1 200 €.
	pts, err := Project(0, money.Cents(100_00), 0, 12)
	if err != nil {
		t.Fatal(err)
	}
	if got := pts[12].Net; got != money.Cents(1_200_00) {
		t.Fatalf("attendu 120000 centimes, obtenu %d", got)
	}
}

func TestProjectInterestDeterministic(t *testing.T) {
	// 10 000 € à 12 %/an (1 %/mois exactement, 1200 bps/12 = 100 bps/mois).
	// Mois 1 : 10 000 × 1,01 = 10 100 € pile (1 010 000 centimes).
	pts, err := Project(money.Cents(1_000_000), 0, 1200, 1)
	if err != nil {
		t.Fatal(err)
	}
	if got := pts[1].Net; got != money.Cents(1_010_000) {
		t.Fatalf("attendu 1010000, obtenu %d", got)
	}
}

func TestProjectRejectsInvalid(t *testing.T) {
	if _, err := Project(0, 0, -1, 12); err == nil {
		t.Fatal("taux négatif : erreur attendue")
	}
	if _, err := Project(0, 0, 0, MaxProjectionMonths+1); err == nil {
		t.Fatal("durée excessive : erreur attendue")
	}
	if _, err := Project(0, 0, bpsScale+1, 12); err == nil {
		t.Fatal("taux > 100 % : erreur attendue")
	}
}

func TestIndependenceTargetRule4Percent(t *testing.T) {
	// Dépenses 2 000 €/mois, retrait 4 % (400 bps) :
	// cible = 2 000 × 12 × 25 = 600 000 € = 60 000 000 centimes.
	target, err := IndependenceTarget(money.Cents(2_000_00), 400)
	if err != nil {
		t.Fatal(err)
	}
	if target != money.Cents(60_000_000) {
		t.Fatalf("attendu 60000000, obtenu %d", target)
	}
}

func TestComputeIndependenceAlreadyReached(t *testing.T) {
	ind, err := ComputeIndependence(
		money.Cents(700_000_00), // 700 k€ de départ
		0,
		money.Cents(2_000_00), // 2 000 €/mois de dépenses → cible 600 k€
		0, 400,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !ind.Reached || ind.Months != 0 {
		t.Fatalf("indépendance déjà atteinte attendue, obtenu %+v", ind)
	}
}

func TestComputeIndependenceProgression(t *testing.T) {
	// 0 € de départ, 1 000 €/mois à 0 %, dépenses 100 €/mois (cible 30 000 €).
	// 30 000 € / 1 000 €/mois = 30 mois exactement.
	ind, err := ComputeIndependence(0, money.Cents(1_000_00), money.Cents(100_00), 0, 400)
	if err != nil {
		t.Fatal(err)
	}
	if !ind.Reached {
		t.Fatal("la cible doit être atteinte")
	}
	if ind.Months != 30 {
		t.Fatalf("attendu 30 mois, obtenu %d", ind.Months)
	}
}

func TestComputeIndependenceNeverReached(t *testing.T) {
	// Aucune épargne, aucun rendement : jamais atteint.
	ind, err := ComputeIndependence(0, 0, money.Cents(2_000_00), 0, 400)
	if err != nil {
		t.Fatal(err)
	}
	if ind.Reached {
		t.Fatal("la cible ne doit pas être atteinte")
	}
}

func TestProjectMonotoneWithPositiveInputs(t *testing.T) {
	// Propriété : avec rendement et versements positifs, la courbe ne
	// redescend jamais.
	pts, err := Project(money.Cents(48_300_00), money.Cents(500_00), 500, 360)
	if err != nil {
		t.Fatal(err)
	}
	for i := 1; i < len(pts); i++ {
		if pts[i].Net < pts[i-1].Net {
			t.Fatalf("mois %d : la projection décroît (%d → %d)",
				pts[i].Month, pts[i-1].Net, pts[i].Net)
		}
	}
}
