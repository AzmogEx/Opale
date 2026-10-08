package engine

import (
	"math/big"
	"time"

	"github.com/opale-app/opale/internal/money"
)

// GoalProjection uses an explicitly allocated monthly saving, without assuming
// investment returns. Dates are civil months; a month-end remains a month-end.
type GoalProjection struct {
	Remaining      money.Cents `json:"remaining_cents"`
	Percent        int         `json:"percent"`
	EstimatedDate  *string     `json:"estimated_date,omitempty"`
	EstimateReason string      `json:"estimate_reason"`
	OnTrack        *bool       `json:"on_track,omitempty"`
}

func CivilMonth(d time.Time, months int) time.Time {
	first := time.Date(d.Year(), d.Month()+time.Month(months), 1, 0, 0, 0, 0, d.Location())
	last := first.AddDate(0, 1, -1).Day()
	day := min(d.Day(), last)
	return time.Date(first.Year(), first.Month(), day, 0, 0, 0, 0, d.Location())
}

func ProjectGoal(target, progress, saving money.Cents, deadline *time.Time, now time.Time) (GoalProjection, error) {
	if target <= 0 || saving < 0 {
		return GoalProjection{}, ErrInvalidInput
	}
	out := GoalProjection{}
	pct := new(big.Int).Mul(big.NewInt(int64(progress)), big.NewInt(100))
	pct.Quo(pct, big.NewInt(int64(target)))
	if pct.Sign() > 0 {
		if pct.Cmp(big.NewInt(100)) >= 0 {
			out.Percent = 100
		} else {
			out.Percent = int(pct.Int64())
		}
	}
	if progress >= target {
		date := now.Format("2006-01-02")
		out.EstimatedDate = &date
		out.EstimateReason = "Objectif atteint"
		yes := true
		out.OnTrack = &yes
		return out, nil
	}
	remaining := new(big.Int).Sub(big.NewInt(int64(target)), big.NewInt(int64(progress)))
	if !remaining.IsInt64() {
		return GoalProjection{}, money.ErrOverflow
	}
	out.Remaining = money.Cents(remaining.Int64())
	if saving == 0 {
		out.EstimateReason = "Aucune épargne mensuelle affectée"
		if deadline != nil && !deadline.After(now) {
			no := false
			out.OnTrack = &no
		}
		return out, nil
	}
	months := new(big.Int).Sub(remaining, big.NewInt(1))
	months.Quo(months, big.NewInt(int64(saving)))
	months.Add(months, big.NewInt(1))
	if months.Cmp(big.NewInt(MaxProjectionMonths)) > 0 {
		out.EstimateReason = "Atteinte au-delà de l’horizon de 100 ans"
		if deadline != nil {
			no := false
			out.OnTrack = &no
		}
		return out, nil
	}
	date := CivilMonth(now, int(months.Int64()))
	text := date.Format("2006-01-02")
	out.EstimatedDate = &text
	out.EstimateReason = "Épargne affectée constante, sans rendement supposé"
	if deadline != nil {
		on := !date.After(*deadline)
		out.OnTrack = &on
	}
	return out, nil
}
