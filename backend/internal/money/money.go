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
	"math/big"
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
	v := new(big.Int).SetInt64(int64(c))
	negative := v.Sign() < 0
	v.Abs(v)
	whole, frac := new(big.Int), new(big.Int)
	whole.QuoRem(v, big.NewInt(100), frac)
	result := fmt.Sprintf("%s.%02d", whole.String(), frac.Int64())
	if negative {
		return "-" + result
	}
	return result
}

// Parse convertit une chaîne décimale ("123.45", "-0,50", "1000") en Cents,
// sans passer par un float. Accepte le point ou la virgule comme séparateur.
func Parse(s string) (Cents, error) { return ParseMinor(s, 2) }

// ParseMinor parses an ISO minor-unit decimal exactly, rejecting overflow and
// additional decimals rather than silently rounding an imported amount.
func ParseMinor(s string, exponent int) (Cents, error) {
	if exponent < 0 || exponent > 4 {
		return 0, errors.New("money: invalid exponent")
	}
	s = strings.ReplaceAll(strings.TrimSpace(s), ",", ".")
	negative := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(strings.TrimPrefix(s, "-"), "+")
	parts := strings.Split(s, ".")
	if len(parts) > 2 || parts[0] == "" {
		return 0, errors.New("money: invalid decimal")
	}
	fraction := ""
	if len(parts) == 2 {
		fraction = parts[1]
	}
	if len(fraction) > exponent {
		return 0, errors.New("money: too many decimals")
	}
	digits := parts[0] + fraction + strings.Repeat("0", exponent-len(fraction))
	for _, c := range digits {
		if c < '0' || c > '9' {
			return 0, errors.New("money: invalid digit")
		}
	}
	if negative {
		digits = "-" + digits
	}
	v, e := strconv.ParseInt(digits, 10, 64)
	if e != nil {
		return 0, ErrOverflow
	}
	return Cents(v), nil
}
