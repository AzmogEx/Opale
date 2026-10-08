package engine

import (
	"github.com/opale-app/opale/internal/money"
	"math/big"
)

// boundedIndicator is only for scores/ratios, never monetary results. The
// unbounded intermediate keeps threshold decisions correct near int64 limits.
func boundedIndicator(n, d *big.Int, scale, low, high int64) int64 {
	if d.Sign() <= 0 {
		return 0
	}
	v := new(big.Int).Mul(n, big.NewInt(scale))
	v.Quo(v, d)
	if v.Cmp(big.NewInt(low)) < 0 {
		return low
	}
	if v.Cmp(big.NewInt(high)) > 0 {
		return high
	}
	return v.Int64()
}
func indicator(n, d, scale, low, high int64) int64 {
	return boundedIndicator(big.NewInt(n), big.NewInt(d), scale, low, high)
}
func positiveAssetTotal(values map[string]money.Cents) *big.Int {
	total := new(big.Int)
	for _, v := range values {
		if v > 0 {
			total.Add(total, big.NewInt(int64(v)))
		}
	}
	return total
}
