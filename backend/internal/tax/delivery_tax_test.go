package tax

import (
	"github.com/opale-app/opale/internal/money"
	"testing"
)

// Independent worked example published by Service Public on 15 April 2026.
func TestOfficial2026WorkedExample(t *testing.T) {
	r := Compute(money.Cents(3000000), 10)
	if r.Tax != 210399 {
		t.Fatalf("official gross tax 210399, got %d", r.Tax)
	}
	if Compute(money.Cents(1160000), 10).Tax != 0 {
		t.Fatal("first bracket")
	}
}
