package engine

import (
	"github.com/opale-app/opale/internal/money"
	"math"
	"testing"
	"time"
)

func TestGoalProjectionCivilAndBoundaries(t *testing.T) {
	now := time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)
	g, e := ProjectGoal(150000, 100000, 50000, nil, now)
	if e != nil || g.EstimatedDate == nil || *g.EstimatedDate != "2026-02-28" || g.Remaining != 50000 {
		t.Fatalf("%+v %v", g, e)
	}
	g, e = ProjectGoal(100, 0, 0, nil, now)
	if e != nil || g.EstimatedDate != nil {
		t.Fatal(g, e)
	}
	g, e = ProjectGoal(100, 100, 0, nil, now)
	if e != nil || g.Percent != 100 || g.Remaining != 0 {
		t.Fatal(g, e)
	}
	_, e = ProjectGoal(money.Cents(math.MaxInt64), money.Cents(math.MinInt64), 1, nil, now)
	if e == nil {
		t.Fatal("overflow accepted")
	}
	g, e = ProjectGoal(1000000, 0, 1, nil, now)
	if e != nil || g.EstimatedDate != nil {
		t.Fatal(g, e)
	}
}
