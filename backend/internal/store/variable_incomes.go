package store

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/opale-app/opale/internal/categorize"
	"github.com/opale-app/opale/internal/money"
	"strings"
)

type VariableIncome struct {
	ID             string      `json:"id"`
	Revision       int64       `json:"revision"`
	Name           string      `json:"name"`
	Kind           string      `json:"kind"`
	Currency       string      `json:"currency"`
	Low            money.Cents `json:"low_cents"`
	Usual          money.Cents `json:"usual_cents"`
	High           money.Cents `json:"high_cents"`
	Frequency      string      `json:"frequency"`
	NextDate       string      `json:"next_date"`
	Forecast       string      `json:"forecast"`
	AssetID        string      `json:"asset_id"`
	CalendarRuleID string      `json:"calendar_rule_id"`
	MerchantKey    string      `json:"merchant_key"`
	Active         bool        `json:"active"`
	Note           string      `json:"note"`
}

const incomeColumns = `id,revision,name,kind,currency,low_cents,usual_cents,high_cents,frequency,next_date::text,forecast,COALESCE(asset_id::text,''),COALESCE(calendar_rule_id::text,''),merchant_key,active,note`

func scanIncome(row pgx.Row) (VariableIncome, error) {
	var v VariableIncome
	e := row.Scan(&v.ID, &v.Revision, &v.Name, &v.Kind, &v.Currency, &v.Low, &v.Usual, &v.High, &v.Frequency, &v.NextDate, &v.Forecast, &v.AssetID, &v.CalendarRuleID, &v.MerchantKey, &v.Active, &v.Note)
	return v, e
}
func (s *Store) incomeRaw(ctx context.Context, profile, id string) (VariableIncome, error) {
	v, e := scanIncome(s.pool.QueryRow(ctx, `SELECT `+incomeColumns+` FROM variable_incomes WHERE profile_id=$1 AND id=$2`, profile, id))
	return v, domainNotFound(e)
}
func (s *Store) ListVariableIncomes(ctx context.Context, profile string) ([]VariableIncome, error) {
	rows, e := s.pool.Query(ctx, `SELECT `+incomeColumns+` FROM variable_incomes WHERE profile_id=$1 ORDER BY active DESC,name,id`, profile)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []VariableIncome{}
	for rows.Next() {
		v, e := scanIncome(rows)
		if e != nil {
			return nil, e
		}
		v.NextDate = nextFinancialDate(v.NextDate, v.Frequency, v.Active, onboardingToday())
		out = append(out, v)
	}
	return out, rows.Err()
}
func (s *Store) SaveVariableIncome(ctx context.Context, profile string, v VariableIncome) (VariableIncome, error) {
	if !financialUUID(v.ID) || v.Revision < 0 || strings.TrimSpace(v.Name) == "" || len(v.Name) > 200 || len(v.Note) > 2000 || len(v.MerchantKey) > 200 || v.Low < 0 || v.Usual <= 0 || v.High > maxOnboardingCents || v.Low > v.Usual || v.Usual > v.High || !financialCurrency(v.Currency) || !financialFrequency(v.Frequency, true) || !financialDay(v.NextDate, false) || !strings.Contains("|freelance|bonus|rental|dividend|other|", "|"+v.Kind+"|") || v.Kind == "" {
		return v, setupInvalid("Revenu : nom, dates et montants valides requis (0 ≤ minimum ≤ habituel ≤ maximum).")
	}
	if v.Forecast != "off" && v.Forecast != "prudent" && v.Forecast != "usual" {
		return v, ErrInvalid
	}
	if v.Forecast != "off" && v.AssetID == "" {
		return v, setupInvalid("Un compte est nécessaire pour ajouter la prévision au calendrier.")
	}
	v.Name = strings.TrimSpace(v.Name)
	v.MerchantKey = categorize.MerchantKey(v.MerchantKey)
	var out VariableIncome
	e := s.Atomic(ctx, func(st *Store) error {
		if e := st.lockFinancialProfile(ctx, profile); e != nil {
			return e
		}
		old, e := st.incomeRaw(ctx, profile, v.ID)
		create := errors.Is(e, ErrNotFound)
		if e != nil && !create {
			return e
		}
		if create && v.Revision != 0 || !create && v.Revision != old.Revision {
			return ErrFinancialConflict
		}
		if e = st.financialAccount(ctx, profile, v.AssetID, v.Currency, v.Active && v.Forecast != "off"); e != nil {
			return e
		}
		amount := v.Low
		if v.Forecast == "usual" {
			amount = v.Usual
		}
		previous := ""
		if !create {
			previous = old.CalendarRuleID
		}
		rule, e := st.syncFinancialRule(ctx, profile, previous, v.AssetID, v.Name, v.NextDate, v.Frequency, v.MerchantKey, amount, v.Active && v.Forecast != "off")
		if e != nil {
			return e
		}
		if create {
			_, e = st.pool.Exec(ctx, `INSERT INTO variable_incomes(id,profile_id,name,kind,currency,low_cents,usual_cents,high_cents,frequency,next_date,forecast,asset_id,calendar_rule_id,merchant_key,active,note) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,NULLIF($12,'')::uuid,NULLIF($13,'')::uuid,$14,$15,$16)`, v.ID, profile, v.Name, v.Kind, v.Currency, v.Low, v.Usual, v.High, v.Frequency, v.NextDate, v.Forecast, v.AssetID, rule, v.MerchantKey, v.Active, v.Note)
		} else {
			_, e = st.pool.Exec(ctx, `UPDATE variable_incomes SET revision=revision+1,name=$3,kind=$4,currency=$5,low_cents=$6,usual_cents=$7,high_cents=$8,frequency=$9,next_date=$10,forecast=$11,asset_id=NULLIF($12,'')::uuid,calendar_rule_id=NULLIF($13,'')::uuid,merchant_key=$14,active=$15,note=$16,updated_at=now() WHERE id=$1 AND profile_id=$2`, v.ID, profile, v.Name, v.Kind, v.Currency, v.Low, v.Usual, v.High, v.Frequency, v.NextDate, v.Forecast, v.AssetID, rule, v.MerchantKey, v.Active, v.Note)
		}
		if e != nil {
			return e
		}
		out, e = st.incomeRaw(ctx, profile, v.ID)
		return e
	})
	return out, e
}
func (s *Store) DeleteVariableIncome(ctx context.Context, profile, id string, revision int64) error {
	return s.Atomic(ctx, func(st *Store) error {
		if e := st.lockFinancialProfile(ctx, profile); e != nil {
			return e
		}
		v, e := st.incomeRaw(ctx, profile, id)
		if e != nil {
			return e
		}
		if v.Revision != revision {
			return ErrFinancialConflict
		}
		if e = st.stopFinancialRule(ctx, profile, v.CalendarRuleID, onboardingToday().AddDate(0, 0, -1).Format("2006-01-02")); e != nil {
			return e
		}
		_, e = st.pool.Exec(ctx, `DELETE FROM variable_incomes WHERE profile_id=$1 AND id=$2`, profile, id)
		return e
	})
}
