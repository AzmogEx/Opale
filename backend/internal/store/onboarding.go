package store

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/opale-app/opale/internal/money"
)

var ErrOnboardingConflict = errors.New("onboarding revision conflict")
var ErrOnboardingUnavailable = errors.New("onboarding unavailable")

// OnboardingValidationError contains only a fixed, safe message, never draft data.
type OnboardingValidationError struct{ Message string }

func (e *OnboardingValidationError) Error() string { return e.Message }

func setupInvalid(message string) error { return &OnboardingValidationError{Message: message} }

type OnboardingIncome struct {
	Enabled  bool   `json:"enabled"`
	Amount   string `json:"amount"`
	NextDate string `json:"next_date"`
}

type OnboardingAccount struct {
	Enabled         bool   `json:"enabled"`
	ExistingAssetID string `json:"existing_asset_id"`
	Name            string `json:"name"`
	Balance         string `json:"balance"`
	BalanceDate     string `json:"balance_date"`
}

type OnboardingExpense struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	Amount    string `json:"amount"`
	Date      string `json:"date"`
	Frequency string `json:"frequency"`
}

type OnboardingGoal struct {
	Enabled bool   `json:"enabled"`
	Name    string `json:"name"`
	Target  string `json:"target"`
}

type OnboardingDraft struct {
	Income         OnboardingIncome    `json:"income"`
	Account        OnboardingAccount   `json:"account"`
	Expenses       []OnboardingExpense `json:"expenses"`
	Subscriptions  []OnboardingExpense `json:"subscriptions"`
	Goal           OnboardingGoal      `json:"goal"`
	VariableBudget string              `json:"variable_budget"`
}

type OnboardingResult struct {
	AssetID             string   `json:"asset_id"`
	IncomeRuleID        string   `json:"income_rule_id"`
	ExpenseRuleIDs      []string `json:"expense_rule_ids"`
	SubscriptionRuleIDs []string `json:"subscription_rule_ids"`
	GoalID              string   `json:"goal_id"`
}

type OnboardingState struct {
	Version      int               `json:"version"`
	Status       string            `json:"status"`
	Step         int               `json:"step"`
	Revision     int64             `json:"revision"`
	CanStart     bool              `json:"can_start"`
	ShouldPrompt bool              `json:"should_prompt"`
	Draft        OnboardingDraft   `json:"draft"`
	Result       *OnboardingResult `json:"result"`
}

func onboardingToday() time.Time {
	loc, _ := time.LoadLocation("Europe/Paris")
	now := time.Now().In(loc)
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
}

func defaultOnboardingDraft() OnboardingDraft {
	today := onboardingToday()
	return OnboardingDraft{
		Income:   OnboardingIncome{Enabled: true, NextDate: today.Format("2006-01-02")},
		Account:  OnboardingAccount{Enabled: true, Name: "Compte courant", BalanceDate: today.AddDate(0, 0, -1).Format("2006-01-02")},
		Expenses: []OnboardingExpense{}, Subscriptions: []OnboardingExpense{},
	}
}

// GetOnboarding does not create a row. Existing populated profiles and demo
// profiles never receive an automatic setup gate; regular profiles can opt in.
func (s *Store) GetOnboarding(ctx context.Context, profile string) (OnboardingState, error) {
	out := OnboardingState{Version: 1, Status: "not_started", Draft: defaultOnboardingDraft()}
	var demo, populated bool
	err := s.pool.QueryRow(ctx, `SELECT is_demo,
        EXISTS(SELECT 1 FROM assets WHERE profile_id=p.id)
        OR EXISTS(SELECT 1 FROM liabilities WHERE profile_id=p.id)
        OR EXISTS(SELECT 1 FROM transactions WHERE profile_id=p.id)
        OR EXISTS(SELECT 1 FROM goals WHERE profile_id=p.id)
        OR EXISTS(SELECT 1 FROM envelopes WHERE profile_id=p.id)
        OR EXISTS(SELECT 1 FROM bank_links WHERE profile_id=p.id)
        OR EXISTS(SELECT 1 FROM documents WHERE profile_id=p.id)
        OR EXISTS(SELECT 1 FROM contacts WHERE profile_id=p.id)
        FROM profiles p WHERE id=$1`, profile).Scan(&demo, &populated)
	if errors.Is(err, pgx.ErrNoRows) {
		return out, ErrNotFound
	}
	if err != nil {
		return out, err
	}
	var draft, result []byte
	err = s.pool.QueryRow(ctx, `SELECT version,status,step,revision,draft,result FROM profile_onboarding WHERE profile_id=$1`, profile).Scan(&out.Version, &out.Status, &out.Step, &out.Revision, &draft, &result)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return out, err
	}
	if err == nil {
		if err = json.Unmarshal(draft, &out.Draft); err != nil {
			return out, err
		}
		if result != nil {
			if err = json.Unmarshal(result, &out.Result); err != nil {
				return out, err
			}
		}
	}
	normalizeOnboardingLists(&out.Draft)
	out.CanStart = !demo && out.Status != "completed"
	out.ShouldPrompt = out.CanStart && !populated && (out.Status == "not_started" || out.Status == "draft")
	return out, nil
}

func normalizeOnboardingLists(d *OnboardingDraft) {
	if d.Expenses == nil {
		d.Expenses = []OnboardingExpense{}
	}
	if d.Subscriptions == nil {
		d.Subscriptions = []OnboardingExpense{}
	}
}

// Locking the profile serializes the first draft as well as all later saves,
// skips and completions; no race is possible before the onboarding row exists.
func (s *Store) lockOnboarding(ctx context.Context, profile string) (OnboardingState, error) {
	var demo bool
	err := s.pool.QueryRow(ctx, `SELECT is_demo FROM profiles WHERE id=$1 FOR UPDATE`, profile).Scan(&demo)
	if errors.Is(err, pgx.ErrNoRows) {
		return OnboardingState{}, ErrNotFound
	}
	if err != nil {
		return OnboardingState{}, err
	}
	if demo {
		return OnboardingState{}, ErrOnboardingUnavailable
	}
	return s.GetOnboarding(ctx, profile)
}

// SaveOnboarding retains incomplete input. Validation of financial meaning only
// runs at completion, so returning to a partially filled page is always safe.
func (s *Store) SaveOnboarding(ctx context.Context, profile string, expected int64, step int, draft OnboardingDraft) (OnboardingState, error) {
	var out OnboardingState
	if expected < 0 || step < 0 || step > 5 {
		return out, setupInvalid("Étape ou révision invalide.")
	}
	if err := validateOnboardingBounds(draft); err != nil {
		return out, err
	}
	normalizeOnboardingLists(&draft)
	err := s.Atomic(ctx, func(st *Store) error {
		state, err := st.lockOnboarding(ctx, profile)
		if err != nil {
			return err
		}
		if state.Status == "completed" || state.Revision != expected {
			return ErrOnboardingConflict
		}
		raw, err := json.Marshal(draft)
		if err != nil {
			return err
		}
		_, err = st.pool.Exec(ctx, `INSERT INTO profile_onboarding(profile_id,status,step,revision,draft) VALUES($1,'draft',$2,1,$3)
          ON CONFLICT(profile_id) DO UPDATE SET status='draft',step=$2,revision=profile_onboarding.revision+1,draft=$3,updated_at=now()`, profile, step, raw)
		if err != nil {
			return err
		}
		out, err = st.GetOnboarding(ctx, profile)
		return err
	})
	return out, err
}

func (s *Store) SkipOnboarding(ctx context.Context, profile string, expected int64) (OnboardingState, error) {
	var out OnboardingState
	if expected < 0 {
		return out, setupInvalid("Révision invalide.")
	}
	err := s.Atomic(ctx, func(st *Store) error {
		state, err := st.lockOnboarding(ctx, profile)
		if err != nil {
			return err
		}
		if state.Status == "completed" || state.Revision != expected {
			return ErrOnboardingConflict
		}
		raw, err := json.Marshal(state.Draft)
		if err != nil {
			return err
		}
		_, err = st.pool.Exec(ctx, `INSERT INTO profile_onboarding(profile_id,status,step,revision,draft) VALUES($1,'skipped',$2,1,$3)
          ON CONFLICT(profile_id) DO UPDATE SET status='skipped',revision=profile_onboarding.revision+1,updated_at=now()`, profile, state.Step, raw)
		if err != nil {
			return err
		}
		out, err = st.GetOnboarding(ctx, profile)
		return err
	})
	return out, err
}

// CompleteOnboarding applies exactly one saved draft. A lost HTTP response or
// concurrent completion returns the durable result without recreating anything.
func (s *Store) CompleteOnboarding(ctx context.Context, profile string, expected int64) (OnboardingState, error) {
	var out OnboardingState
	if expected < 0 {
		return out, setupInvalid("Révision invalide.")
	}
	err := s.Atomic(ctx, func(st *Store) error {
		state, err := st.lockOnboarding(ctx, profile)
		if err != nil {
			return err
		}
		if state.Status == "completed" {
			out = state
			return nil
		}
		if state.Revision != expected || state.Status == "not_started" {
			return ErrOnboardingConflict
		}
		parsed, err := validateOnboarding(state.Draft, onboardingToday())
		if err != nil {
			return err
		}
		d := state.Draft
		result := &OnboardingResult{ExpenseRuleIDs: []string{}, SubscriptionRuleIDs: []string{}}
		if d.Account.Enabled {
			if d.Account.ExistingAssetID != "" {
				// An existing account keeps its current name and valuations. It must
				// be an active EUR cash account, owned by the authenticated profile.
				err = st.pool.QueryRow(ctx, `SELECT id FROM assets WHERE id=$1 AND profile_id=$2 AND NOT archived AND currency='EUR' AND kind IN ('checking','savings') FOR UPDATE`, d.Account.ExistingAssetID, profile).Scan(&result.AssetID)
				if errors.Is(err, pgx.ErrNoRows) {
					return ErrNotFound
				}
				if err != nil {
					return err
				}
			} else {
				a, e := st.CreateAsset(ctx, profile, strings.TrimSpace(d.Account.Name), "checking", "EUR", "")
				if e != nil {
					return e
				}
				result.AssetID = a.ID
				if parsed.balance != nil {
					if _, e = st.AddAssetValuation(ctx, profile, a.ID, *parsed.balance, parsed.balanceDate, "Solde de clôture déclaré lors de la première configuration"); e != nil {
						return e
					}
				}
			}
		}
		if d.Income.Enabled {
			rule, e := st.SaveCalendarRule(ctx, profile, CalendarRule{AssetID: result.AssetID, Label: "Salaire net", Amount: parsed.income, Date: d.Income.NextDate, Frequency: "monthly", Active: true})
			if e != nil {
				return e
			}
			result.IncomeRuleID = rule.ID
		}
		for i, group := range [][]OnboardingExpense{d.Expenses, d.Subscriptions} {
			for j, entry := range group {
				rule, e := st.SaveCalendarRule(ctx, profile, CalendarRule{AssetID: result.AssetID, Label: strings.TrimSpace(entry.Label), Amount: -parsed.expenses[i][j], Date: entry.Date, Frequency: entry.Frequency, Active: true})
				if e != nil {
					return e
				}
				if i == 0 {
					result.ExpenseRuleIDs = append(result.ExpenseRuleIDs, rule.ID)
				} else {
					result.SubscriptionRuleIDs = append(result.SubscriptionRuleIDs, rule.ID)
					if e = st.TrackCalendarContract(ctx, profile, rule.ID); e != nil {
						return e
					}
				}
			}
		}
		if d.Goal.Enabled {
			// A declared intention is not observed savings: no monthly allocation
			// and no link to all of the money in the principal account.
			g, e := st.SaveGoal(ctx, profile, "", strings.TrimSpace(d.Goal.Name), "target", parsed.goal, nil, nil, 0)
			if e != nil {
				return e
			}
			result.GoalID = g.ID
		}
		raw, err := json.Marshal(result)
		if err != nil {
			return err
		}
		_, err = st.pool.Exec(ctx, `UPDATE profile_onboarding SET status='completed',step=5,revision=revision+1,result=$2,completed_at=now(),updated_at=now() WHERE profile_id=$1`, profile, raw)
		if err != nil {
			return err
		}
		out, err = st.GetOnboarding(ctx, profile)
		return err
	})
	return out, err
}

// Bound draft data without requiring completed fields or persisting arbitrary
// JSON. Dates and amounts are validated exactly when the user confirms.
func validateOnboardingBounds(d OnboardingDraft) error {
	if len(d.Expenses) > 100 || len(d.Subscriptions) > 100 {
		return setupInvalid("Maximum 100 charges et 100 abonnements.")
	}
	for _, v := range []string{d.Income.Amount, d.Account.Balance, d.Goal.Target, d.VariableBudget} {
		if len(v) > 64 {
			return setupInvalid("Montant trop long.")
		}
	}
	for _, v := range []string{d.Income.NextDate, d.Account.BalanceDate, d.Account.ExistingAssetID} {
		if len(v) > 64 {
			return setupInvalid("Date ou identifiant trop long.")
		}
	}
	if len(d.Account.Name) > 200 || len(d.Goal.Name) > 200 {
		return setupInvalid("Nom limité à 200 caractères.")
	}
	for _, entries := range [][]OnboardingExpense{d.Expenses, d.Subscriptions} {
		for _, e := range entries {
			if len(e.ID) > 64 || len(e.Label) > 200 || len(e.Amount) > 64 || len(e.Date) > 64 || len(e.Frequency) > 20 {
				return setupInvalid("Échéance trop longue.")
			}
		}
	}
	return nil
}

type parsedOnboarding struct {
	income      money.Cents
	balance     *money.Cents
	balanceDate time.Time
	expenses    [2][]money.Cents
	goal        money.Cents
}

const maxOnboardingCents money.Cents = 999_999_999_999

var setupDecimal = regexp.MustCompile(`^[+-]?([0-9]+([.,][0-9]{0,2})?|[.,][0-9]{1,2})$`)

func setupAmount(raw, field string, positive, signed bool) (money.Cents, error) {
	// Match iOS Cents.parse: grouping whitespace and the EUR symbol are
	// presentation only; decimal conversion still uses integer arithmetic.
	raw = strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) || r == '€' {
			return -1
		}
		return r
	}, raw)
	if !setupDecimal.MatchString(raw) {
		return 0, setupInvalid(field + " : montant valide en euros requis, avec deux décimales au maximum.")
	}
	raw = strings.ReplaceAll(raw, ",", ".")
	for _, prefix := range []string{"", "+", "-"} {
		if strings.HasPrefix(raw, prefix+".") {
			raw = prefix + "0" + strings.TrimPrefix(raw, prefix)
			break
		}
	}
	value, err := money.Parse(raw)
	if err != nil || value > maxOnboardingCents || value < -maxOnboardingCents || (!signed && value < 0) || (positive && value <= 0) {
		return 0, setupInvalid(field + " : montant valide en euros requis, avec deux décimales au maximum.")
	}
	return value, nil
}

func setupNextDate(raw string, today time.Time) error {
	d, err := time.Parse("2006-01-02", raw)
	if err != nil || d.Before(today) || d.After(today.AddDate(10, 0, 0)) {
		return setupInvalid("La prochaine échéance doit être comprise entre aujourd’hui et les dix prochaines années.")
	}
	return nil
}

func validateOnboarding(d OnboardingDraft, today time.Time) (parsedOnboarding, error) {
	var out parsedOnboarding
	if err := validateOnboardingBounds(d); err != nil {
		return out, err
	}
	var err error
	if (d.Income.Enabled || len(d.Expenses) > 0 || len(d.Subscriptions) > 0) && !d.Account.Enabled {
		return out, setupInvalid("Ajoute ou sélectionne un compte pour enregistrer tes échéances.")
	}
	if d.Account.Enabled {
		if d.Account.ExistingAssetID != "" {
			var id pgtype.UUID
			if err = id.Scan(d.Account.ExistingAssetID); err != nil {
				return out, setupInvalid("Compte invalide.")
			}
		} else {
			if strings.TrimSpace(d.Account.Name) == "" {
				return out, setupInvalid("Donne un nom au compte principal.")
			}
			if strings.TrimSpace(d.Account.Balance) != "" {
				amount, e := setupAmount(d.Account.Balance, "Solde", false, true)
				if e != nil {
					return out, e
				}
				out.balance = &amount
				out.balanceDate, err = time.Parse("2006-01-02", d.Account.BalanceDate)
				if err != nil || out.balanceDate.Year() < 1900 || out.balanceDate.After(today) {
					return out, setupInvalid("La date du solde doit être comprise entre 1900 et aujourd’hui.")
				}
			}
		}
	}
	if d.Income.Enabled {
		if out.income, err = setupAmount(d.Income.Amount, "Salaire net", true, false); err != nil {
			return out, err
		}
		if err = setupNextDate(d.Income.NextDate, today); err != nil {
			return out, err
		}
	}
	ids := map[[16]byte]bool{}
	for i, entries := range [][]OnboardingExpense{d.Expenses, d.Subscriptions} {
		for _, e := range entries {
			var id pgtype.UUID
			if err = id.Scan(e.ID); err != nil || !id.Valid || ids[id.Bytes] {
				return out, setupInvalid("Chaque échéance doit avoir un identifiant unique valide.")
			}
			ids[id.Bytes] = true
			if strings.TrimSpace(e.Label) == "" {
				return out, setupInvalid("Donne un nom à chaque charge ou abonnement.")
			}
			if e.Frequency != "monthly" && e.Frequency != "quarterly" && e.Frequency != "yearly" {
				return out, setupInvalid("Périodicité mensuelle, trimestrielle ou annuelle requise.")
			}
			amount, err := setupAmount(e.Amount, "Charge ou abonnement", true, false)
			if err != nil {
				return out, err
			}
			if err = setupNextDate(e.Date, today); err != nil {
				return out, err
			}
			out.expenses[i] = append(out.expenses[i], amount)
		}
	}
	if d.Goal.Enabled {
		if strings.TrimSpace(d.Goal.Name) == "" {
			return out, setupInvalid("Donne un nom à ton objectif.")
		}
		if out.goal, err = setupAmount(d.Goal.Target, "Objectif", true, false); err != nil {
			return out, err
		}
	}
	if strings.TrimSpace(d.VariableBudget) != "" {
		if _, err = setupAmount(d.VariableBudget, "Budget variable", false, false); err != nil {
			return out, err
		}
	}
	return out, nil
}
