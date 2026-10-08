package api

import (
	"github.com/opale-app/opale/internal/money"
	"math/big"
)

// Financial ratios use an unbounded intermediate: a valid monetary value can
// overflow int64 when multiplied by basis points before the division.
func scaleCents(value money.Cents, numerator, denominator int64) (money.Cents, error) {
	if denominator <= 0 {
		return 0, money.ErrOverflow
	}
	n := new(big.Int).Mul(big.NewInt(int64(value)), big.NewInt(numerator))
	n.Quo(n, big.NewInt(denominator))
	if !n.IsInt64() {
		return 0, money.ErrOverflow
	}
	return money.Cents(n.Int64()), nil
}
