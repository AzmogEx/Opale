package store

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/opale-app/opale/internal/money"
)

var ErrFinancialConflict = errors.New("financial record revision conflict")
var ErrManagedCalendar = errors.New("calendar managed by contract or variable income")

func financialUUID(id string) bool { var u pgtype.UUID; return u.Scan(id) == nil && u.Valid }
func financialDay(day string, optional bool) bool {
	if day == "" {
		return optional
	}
	d, e := time.Parse("2006-01-02", day)
	return e == nil && d.Year() >= 1900 && !d.After(onboardingToday().AddDate(10, 0, 0))
}
func financialCurrency(c string) bool {
	return strings.Contains("|EUR|USD|GBP|CHF|JPY|KWD|CAD|AUD|", "|"+c+"|") && len(c) == 3
}
func financialFrequency(f string, once bool) bool {
	return f == "monthly" || f == "quarterly" || f == "yearly" || (once && f == "once")
}
func (s *Store) lockFinancialProfile(ctx context.Context, profile string) error {
	var id string
	e := s.pool.QueryRow(ctx, `SELECT id FROM profiles WHERE id=$1 FOR UPDATE`, profile).Scan(&id)
	return domainNotFound(e)
}
func (s *Store) financialAccount(ctx context.Context, profile, id, currency string, requireActive bool) error {
	if id == "" {
		return nil
	}
	if !financialUUID(id) {
		return ErrInvalid
	}
	var ok bool
	e := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM assets WHERE profile_id=$1 AND id=$2 AND currency=$3 AND (NOT $4 OR (NOT archived AND kind IN ('checking','savings'))))`, profile, id, currency, requireActive).Scan(&ok)
	if e != nil {
		return e
	}
	if !ok {
		return ErrNotFound
	}
	return nil
}
func (s *Store) stopFinancialRule(ctx context.Context, profile, id, end string) error {
	if id == "" {
		return nil
	}
	// Keep history. Rules starting in the future can simply be disabled.
	_, e := s.pool.Exec(ctx, `UPDATE calendar_rules SET ends_on=CASE WHEN starts_on<=$3::date THEN $3::date ELSE ends_on END,active=CASE WHEN starts_on>$3::date THEN false ELSE active END WHERE profile_id=$1 AND id=$2`, profile, id, end)
	return e
}
func (s *Store) syncFinancialRule(ctx context.Context, profile, previous, account, label, date, frequency, merchant string, amount money.Cents, active bool) (string, error) {
	today := onboardingToday()
	end := today.AddDate(0, 0, -1).Format("2006-01-02")
	if previous != "" {
		var c CalendarRule
		e := s.pool.QueryRow(ctx, `SELECT id,asset_id,label,amount_cents,starts_on::text,frequency,active,merchant_key FROM calendar_rules WHERE profile_id=$1 AND id=$2 FOR UPDATE`, profile, previous).Scan(&c.ID, &c.AssetID, &c.Label, &c.Amount, &c.Date, &c.Frequency, &c.Active, &c.MerchantKey)
		if e != nil && !errors.Is(e, pgx.ErrNoRows) {
			return "", e
		}
		if e == nil && active && account != "" && amount != 0 && c.AssetID == account && c.Amount == amount && c.Frequency == frequency && c.Date == date && c.Active {
			_, e = s.pool.Exec(ctx, `UPDATE calendar_rules SET label=$3,merchant_key=$4 WHERE profile_id=$1 AND id=$2`, profile, previous, label, merchant)
			return previous, e
		}
		// Replacement starts at a future declared date, without rewriting older occurrences.
		if active && account != "" && amount != 0 && date >= today.Format("2006-01-02") {
			d, _ := time.Parse("2006-01-02", date)
			end = d.AddDate(0, 0, -1).Format("2006-01-02")
		}
		if e = s.stopFinancialRule(ctx, profile, previous, end); e != nil {
			return "", e
		}
	}
	if !active || account == "" || amount == 0 {
		return "", nil
	}
	if date < today.Format("2006-01-02") {
		return "", setupInvalid("Choisis une prochaine échéance à partir d’aujourd’hui pour créer la prévision.")
	}
	c, e := s.SaveCalendarRule(ctx, profile, CalendarRule{AssetID: account, Label: label, Amount: amount, Date: date, Frequency: frequency, Active: true, MerchantKey: merchant})
	return c.ID, e
}
func nextFinancialDate(date, frequency string, active bool, today time.Time) string {
	if !active {
		return date
	}
	ds := CalendarDates(CalendarRule{Date: date, Frequency: frequency, Active: true}, today, today.AddDate(2, 0, 0))
	if len(ds) > 0 {
		return ds[0].Format("2006-01-02")
	}
	return date
}

// Managed forecasts must be edited from their originating contract/income.
func (s *Store) ManagedCalendarRule(ctx context.Context, profile, id string) (bool, error) {
	var ok bool
	e := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM financial_contracts WHERE profile_id=$1 AND calendar_rule_id=$2) OR EXISTS(SELECT 1 FROM variable_incomes WHERE profile_id=$1 AND calendar_rule_id=$2)`, profile, id).Scan(&ok)
	return ok, e
}

func (s *Store) SaveUserCalendarRule(ctx context.Context, profile string, c CalendarRule) (CalendarRule, error) {
	var out CalendarRule
	e := s.Atomic(ctx, func(st *Store) error {
		if e := st.lockFinancialProfile(ctx, profile); e != nil {
			return e
		}
		if c.ID != "" {
			managed, e := st.ManagedCalendarRule(ctx, profile, c.ID)
			if e != nil {
				return e
			}
			if managed {
				return ErrManagedCalendar
			}
		}
		var e error
		out, e = st.SaveCalendarRule(ctx, profile, c)
		return e
	})
	return out, e
}

func (s *Store) DeleteUserCalendarRule(ctx context.Context, profile, id string) error {
	return s.Atomic(ctx, func(st *Store) error {
		if e := st.lockFinancialProfile(ctx, profile); e != nil {
			return e
		}
		managed, e := st.ManagedCalendarRule(ctx, profile, id)
		if e != nil {
			return e
		}
		if managed {
			return ErrManagedCalendar
		}
		return st.DeleteCalendarRule(ctx, profile, id)
	})
}
