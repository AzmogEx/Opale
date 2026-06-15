// Package money définit le type monétaire d'Opale.
//
// RÈGLE D'OR (ENF-007, CONCEPTION §8) : l'argent ne doit JAMAIS être stocké ni
// calculé en float/double. Tous les montants sont des entiers, exprimés en
// centimes de l'unité (ex. 12 345 = 123,45 €). Ce package est la seule porte
// d'entrée autorisée pour manipuler des montants ; il est testé unitairement
// (cf. CA-2).
package money

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Cents représente un montant en centimes (entier signé). Positif = avoir,
// négatif = dette/sortie.
type Cents int64

// ErrOverflow est renvoyé quand une opération dépasse la capacité d'un int64.
var ErrOverflow = errors.New("money: dépassement de capacité (overflow)")

// FromUnits construit un montant à partir d'un nombre entier d'unités (euros).
func FromUnits(units int64) Cents { return Cents(units * 100) }

// Add additionne deux montants en détectant les débordements.
func Add(a, b Cents) (Cents, error) {
	sum := a + b
	// Détection d'overflow sur l'addition d'entiers signés.
	if (b > 0 && sum < a) || (b < 0 && sum > a) {
		return 0, ErrOverflow
	}
	return sum, nil
}

// Sub soustrait b de a en détectant les débordements.
func Sub(a, b Cents) (Cents, error) {
	diff := a - b
	if (b < 0 && diff < a) || (b > 0 && diff > a) {
		return 0, ErrOverflow
	}
	return diff, nil
}

// Sum additionne une série de montants ; renvoie une erreur en cas d'overflow.
func Sum(values ...Cents) (Cents, error) {
	var total Cents
	for _, v := range values {
		t, err := Add(total, v)
		if err != nil {
			return 0, err
		}
		total = t
	}
	return total, nil
}

// Abs renvoie la valeur absolue d'un montant.
func Abs(a Cents) Cents {
	if a < 0 {
		return -a
	}
	return a
}

// Euros renvoie la partie entière (unités) du montant — sans arrondi monétaire,
// uniquement pour de l'affichage ou des calculs non monétaires.
func (c Cents) Euros() int64 { return int64(c) / 100 }

// String formate le montant en chaîne décimale à deux décimales (ex. "-12.05").
// Aucune conversion en float n'est utilisée.
func (c Cents) String() string {
	neg := c < 0
	v := int64(c)
	if neg {
		v = -v
	}
	whole := v / 100
	frac := v % 100
	s := fmt.Sprintf("%d.%02d", whole, frac)
	if neg {
		return "-" + s
	}
	return s
}

// Parse convertit une chaîne décimale ("123.45", "-0,50", "1000") en Cents,
// sans passer par un float. Accepte le point ou la virgule comme séparateur.
func Parse(s string) (Cents, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, errors.New("money: chaîne vide")
	}
	s = strings.ReplaceAll(s, ",", ".")

	neg := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")
	s = strings.TrimPrefix(s, "+")

	parts := strings.SplitN(s, ".", 2)
	whole, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("money: partie entière invalide %q : %w", parts[0], err)
	}

	var frac int64
	if len(parts) == 2 {
		f := parts[1]
		switch {
		case len(f) == 1:
			f += "0"
		case len(f) > 2:
			return 0, fmt.Errorf("money: trop de décimales dans %q (max 2)", s)
		}
		frac, err = strconv.ParseInt(f, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("money: partie décimale invalide %q : %w", parts[1], err)
		}
	}

	if whole > (math.MaxInt64-frac)/100 {
		return 0, ErrOverflow
	}
	cents := whole*100 + frac
	if neg {
		cents = -cents
	}
	return Cents(cents), nil
}
