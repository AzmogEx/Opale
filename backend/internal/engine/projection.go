// Package engine est le MOTEUR FINANCIER DÉTERMINISTE d'Opale (EIA-040/042).
//
// Règles absolues :
//   - Tous les calculs sont en ENTIERS : centimes (money.Cents) et taux en
//     points de base (bps, 1 bps = 0,01 %). Jamais de float (ENF-007).
//   - Chaque fonction est pure et testée unitairement (CA-2).
//   - L'IA ne calcule jamais : elle explique les résultats de ce moteur
//     (EIA-041).
package engine

import (
	"errors"

	"github.com/opale-app/opale/internal/money"
)

// ErrInvalidInput est renvoyé quand un paramètre est hors bornes.
var ErrInvalidInput = errors.New("engine: paramètre invalide")

const (
	// bpsScale : 10 000 bps = 100 %.
	bpsScale = 10_000
	// MaxProjectionMonths borne toute projection (100 ans).
	MaxProjectionMonths = 1200
)

// ProjectionPoint — un point mensuel de la projection.
type ProjectionPoint struct {
	// Month : nombre de mois depuis aujourd'hui (0 = situation actuelle).
	Month int `json:"month"`
	// Net : patrimoine net projeté (centimes).
	Net money.Cents `json:"net_cents"`
}

// Project fait croître un patrimoine `start` pendant `months` mois :
// chaque mois, intérêts composés au taux mensuel dérivé de `annualReturnBps`
// (approximation linéaire annualBps/12, documentée et déterministe),
// puis versement de `monthlyContribution`.
//
// Un taux NÉGATIF est accepté (EF-043 : projections en euros constants —
// rendement réel = nominal − inflation, qui peut passer sous zéro).
//
// Renvoie months+1 points (le point 0 est la situation de départ).
func Project(start, monthlyContribution money.Cents, annualReturnBps, months int) ([]ProjectionPoint, error) {
	if months < 0 || months > MaxProjectionMonths {
		return nil, ErrInvalidInput
	}
	if annualReturnBps < -bpsScale || annualReturnBps > bpsScale {
		// Hors de ±100 %/an : pas un scénario supporté.
		return nil, ErrInvalidInput
	}

	monthlyBps := int64(annualReturnBps) / 12

	points := make([]ProjectionPoint, 0, months+1)
	points = append(points, ProjectionPoint{Month: 0, Net: start})

	v := int64(start)
	for m := 1; m <= months; m++ {
		// Intérêts du mois : v * mbps / 10000, en arithmétique entière.
		// (troncature vers zéro : légèrement conservateur, déterministe)
		interest := v / bpsScale * monthlyBps
		// Terme résiduel pour limiter la perte de précision de la division :
		interest += (v % bpsScale) * monthlyBps / bpsScale

		next := v + interest + int64(monthlyContribution)
		if next < v && monthlyContribution >= 0 && interest >= 0 {
			return nil, money.ErrOverflow
		}
		v = next
		points = append(points, ProjectionPoint{Month: m, Net: money.Cents(v)})
	}
	return points, nil
}

// Independence — résultat du calcul de date d'indépendance financière (EF-040).
type Independence struct {
	// Reached : la cible est atteinte dans la fenêtre de projection.
	Reached bool `json:"reached"`
	// Months : nombre de mois avant l'indépendance (0 si déjà atteinte).
	Months int `json:"months"`
	// Target : patrimoine cible (centimes).
	Target money.Cents `json:"target_cents"`
}

// IndependenceTarget calcule le patrimoine cible selon la règle du taux de
// retrait sûr : cible = dépenses annuelles × (10000 / swrBps).
// Exemple : retrait 4 % (400 bps) → cible = 25 × dépenses annuelles.
func IndependenceTarget(monthlyExpenses money.Cents, swrBps int) (money.Cents, error) {
	if monthlyExpenses <= 0 || swrBps <= 0 || swrBps > bpsScale {
		return 0, ErrInvalidInput
	}
	annual := int64(monthlyExpenses) * 12
	target := annual * bpsScale / int64(swrBps)
	if target < 0 {
		return 0, money.ErrOverflow
	}
	return money.Cents(target), nil
}

// ComputeIndependence projette le patrimoine mois par mois (mêmes règles que
// Project) et renvoie le premier mois où la cible est atteinte, dans la
// limite de MaxProjectionMonths.
func ComputeIndependence(
	start, monthlyContribution, monthlyExpenses money.Cents,
	annualReturnBps, swrBps int,
) (Independence, error) {
	target, err := IndependenceTarget(monthlyExpenses, swrBps)
	if err != nil {
		return Independence{}, err
	}
	if start >= target {
		return Independence{Reached: true, Months: 0, Target: target}, nil
	}

	points, err := Project(start, monthlyContribution, annualReturnBps, MaxProjectionMonths)
	if err != nil {
		return Independence{}, err
	}
	for _, p := range points[1:] {
		if p.Net >= target {
			return Independence{Reached: true, Months: p.Month, Target: target}, nil
		}
	}
	return Independence{Reached: false, Months: 0, Target: target}, nil
}
