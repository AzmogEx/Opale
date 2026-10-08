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
	"math/big"

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

	points := make([]ProjectionPoint, 0, months+1)
	points = append(points, ProjectionPoint{Month: 0, Net: start})

	v := int64(start)
	for m := 1; m <= months; m++ {
		next, e := projectMonth(money.Cents(v), monthlyContribution, annualReturnBps)
		if e != nil {
			return nil, e
		}
		v = int64(next)
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
	value := new(big.Int).Mul(big.NewInt(int64(monthlyExpenses)), big.NewInt(12*bpsScale))
	value.Quo(value, big.NewInt(int64(swrBps)))
	if !value.IsInt64() {
		return 0, money.ErrOverflow
	}
	return money.Cents(value.Int64()), nil
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

	if annualReturnBps < -bpsScale || annualReturnBps > bpsScale {
		return Independence{}, ErrInvalidInput
	}
	value := start
	for month := 1; month <= MaxProjectionMonths; month++ {
		value, err = projectMonth(value, monthlyContribution, annualReturnBps)
		if err != nil {
			return Independence{}, err
		}
		if value >= target {
			return Independence{Reached: true, Months: month, Target: target}, nil
		}
	}
	return Independence{Reached: false, Months: 0, Target: target}, nil
}

// Interest uses the annual nominal rate divided by twelve without truncating
// the rate itself to whole basis points. Each monthly amount truncates to cents.
func projectMonth(value, contribution money.Cents, annualBps int) (money.Cents, error) {
	interest := new(big.Int).Mul(big.NewInt(int64(value)), big.NewInt(int64(annualBps)))
	interest.Quo(interest, big.NewInt(120000))
	interest.Add(interest, big.NewInt(int64(value)))
	interest.Add(interest, big.NewInt(int64(contribution)))
	if !interest.IsInt64() {
		return 0, money.ErrOverflow
	}
	return money.Cents(interest.Int64()), nil
}
