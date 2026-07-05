package engine

import (
	"sort"
	"time"

	"github.com/opale-app/opale/internal/money"
)

// UpcomingFlow — une échéance datée à venir (calendrier financier, EF-025).
type UpcomingFlow struct {
	Date   time.Time   `json:"date"`
	Label  string      `json:"label"`
	Amount money.Cents `json:"amount_cents"` // signé
}

// CashProjection — cash disponible projeté à une date (EF-027).
type CashProjection struct {
	StartCash Cents_       `json:"start_cash_cents"`
	EndCash   Cents_       `json:"end_cash_cents"`
	Until     time.Time    `json:"until"`
	Upcoming  []UpcomingFlow `json:"upcoming"`
}

// Cents_ : alias JSON local pour rester homogène avec money.Cents.
type Cents_ = money.Cents

// ProjectCash projette le cash disponible jusqu'à `until` (EF-027) :
// cash de départ + toutes les échéances récurrentes attendues (actives)
// − dépenses variables moyennes (dailyVariableSpend × jours).
//
// Déterministe, arithmétique entière uniquement (EIA-040).
func ProjectCash(
	startCash money.Cents,
	recurring []RecurringFlow,
	today, until time.Time,
	dailyVariableSpend money.Cents,
) CashProjection {
	if until.Before(today) {
		until = today
	}

	// Énumération des occurrences de chaque flux actif dans la fenêtre.
	var upcoming []UpcomingFlow
	for _, f := range recurring {
		if !f.Active || f.IntervalDays <= 0 {
			continue
		}
		next := f.NextDate
		// Rattrapage : si l'échéance est légèrement passée mais le flux actif,
		// on la compte à aujourd'hui.
		for next.Before(today) {
			next = next.AddDate(0, 0, f.IntervalDays)
		}
		for !next.After(until) {
			upcoming = append(upcoming, UpcomingFlow{Date: next, Label: f.Label, Amount: f.Amount})
			next = next.AddDate(0, 0, f.IntervalDays)
		}
	}
	sort.Slice(upcoming, func(i, j int) bool { return upcoming[i].Date.Before(upcoming[j].Date) })

	end := int64(startCash)
	for _, u := range upcoming {
		end += int64(u.Amount)
	}

	// Dépenses variables moyennes sur la période.
	days := int64(until.Sub(today).Hours() / 24)
	if days > 0 && dailyVariableSpend > 0 {
		end -= days * int64(dailyVariableSpend)
	}

	return CashProjection{
		StartCash: startCash,
		EndCash:   money.Cents(end),
		Until:     until,
		Upcoming:  upcoming,
	}
}
