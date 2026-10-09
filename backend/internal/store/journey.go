package store

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/opale-app/opale/internal/money"
	"strings"
)

type FinancialJourney struct {
	Revision       int64        `json:"revision"`
	Step           int          `json:"step"`
	Reviewed       []string     `json:"reviewed"`
	Skipped        []string     `json:"skipped"`
	DailyBudget    *money.Cents `json:"daily_budget_cents"`
	BudgetCurrency string       `json:"budget_currency"`
}

func (s *Store) FinancialJourney(ctx context.Context, profile string) (FinancialJourney, error) {
	j := FinancialJourney{Reviewed: []string{}, Skipped: []string{}, BudgetCurrency: "EUR"}
	e := s.pool.QueryRow(ctx, `SELECT revision,step,reviewed,skipped,daily_budget_cents,budget_currency FROM profile_journey WHERE profile_id=$1`, profile).Scan(&j.Revision, &j.Step, &j.Reviewed, &j.Skipped, &j.DailyBudget, &j.BudgetCurrency)
	if errors.Is(e, pgx.ErrNoRows) {
		// Carry the budget declared in the previous form forward, without
		// inventing progress or duplicating any existing financial record.
		var budget string
		e = s.pool.QueryRow(ctx, `SELECT COALESCE(draft->>'variable_budget','') FROM profile_onboarding WHERE profile_id=$1 AND status='completed'`, profile).Scan(&budget)
		if errors.Is(e, pgx.ErrNoRows) {
			return j, nil
		}
		if e != nil {
			return j, e
		}
		if strings.TrimSpace(budget) != "" {
			amount, err := setupAmount(budget, "Budget du quotidien", false, false)
			if err != nil {
				return j, err
			}
			j.DailyBudget = &amount
		}
		return j, nil
	}
	return j, e
}

func (s *Store) SaveFinancialJourney(ctx context.Context, profile string, j FinancialJourney) (FinancialJourney, error) {
	if j.Revision < 0 || j.Step < 0 || j.Step > 6 || !financialCurrency(j.BudgetCurrency) || (j.DailyBudget != nil && (*j.DailyBudget < 0 || *j.DailyBudget > maxOnboardingCents)) {
		return j, ErrInvalid
	}
	allowed := map[string]bool{"accounts": true, "income": true, "expenses": true, "subscriptions": true, "wealth": true, "budget": true, "review": true}
	seen := map[string]bool{}
	for _, ids := range [][]string{j.Reviewed, j.Skipped} {
		for _, id := range ids {
			if !allowed[id] || seen[id] {
				return j, ErrInvalid
			}
			seen[id] = true
		}
	}
	if j.Reviewed == nil {
		j.Reviewed = []string{}
	}
	if j.Skipped == nil {
		j.Skipped = []string{}
	}
	e := s.Atomic(ctx, func(st *Store) error {
		if e := st.lockFinancialProfile(ctx, profile); e != nil {
			return e
		}
		old, e := st.FinancialJourney(ctx, profile)
		if e != nil {
			return e
		}
		if old.Revision != j.Revision {
			return ErrFinancialConflict
		}
		_, e = st.pool.Exec(ctx, `INSERT INTO profile_journey(profile_id,step,reviewed,skipped,daily_budget_cents,budget_currency) VALUES($1,$2,$3,$4,$5,$6)
   ON CONFLICT(profile_id) DO UPDATE SET revision=profile_journey.revision+1,step=EXCLUDED.step,reviewed=EXCLUDED.reviewed,skipped=EXCLUDED.skipped,daily_budget_cents=EXCLUDED.daily_budget_cents,budget_currency=EXCLUDED.budget_currency,updated_at=now()`, profile, j.Step, j.Reviewed, j.Skipped, j.DailyBudget, j.BudgetCurrency)
		if e != nil {
			return e
		}
		j, e = st.FinancialJourney(ctx, profile)
		return e
	})
	return j, e
}
