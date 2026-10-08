package store

import (
	"context"
	"github.com/opale-app/opale/internal/engine"
	"github.com/opale-app/opale/internal/money"
	"time"
)

type CalendarRule struct {
	ID          string      `json:"id"`
	AssetID     string      `json:"asset_id"`
	Label       string      `json:"label"`
	Amount      money.Cents `json:"amount_cents"`
	Date        string      `json:"date"`
	Frequency   string      `json:"frequency"`
	EndDate     *string     `json:"end_date,omitempty"`
	Active      bool        `json:"active"`
	MerchantKey string      `json:"merchant_key"`
}
type CalendarOccurrence struct {
	RuleID        string      `json:"rule_id"`
	AssetID       string      `json:"asset_id"`
	Label         string      `json:"label"`
	Date          string      `json:"date"`
	Amount        money.Cents `json:"amount_cents"`
	EUR           money.Cents `json:"eur_cents"`
	Currency      string      `json:"currency"`
	Status        string      `json:"status"`
	TransactionID *string     `json:"transaction_id,omitempty"`
}

func (s *Store) ListCalendarRules(ctx context.Context, profile string) ([]CalendarRule, error) {
	rows, err := s.pool.Query(ctx, `SELECT id,asset_id,label,amount_cents,starts_on::text,frequency,ends_on::text,active,merchant_key FROM calendar_rules WHERE profile_id=$1 ORDER BY starts_on,id`, profile)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []CalendarRule{}
	for rows.Next() {
		var c CalendarRule
		if err = rows.Scan(&c.ID, &c.AssetID, &c.Label, &c.Amount, &c.Date, &c.Frequency, &c.EndDate, &c.Active, &c.MerchantKey); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
func (s *Store) SaveCalendarRule(ctx context.Context, profile string, c CalendarRule) (CalendarRule, error) {
	var own bool
	if err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM assets WHERE profile_id=$1 AND id=$2 AND NOT archived)`, profile, c.AssetID).Scan(&own); err != nil {
		return c, err
	}
	if !own {
		return c, ErrNotFound
	}
	var err error
	if c.ID == "" {
		err = s.pool.QueryRow(ctx, `INSERT INTO calendar_rules(profile_id,asset_id,label,amount_cents,starts_on,frequency,ends_on,active,merchant_key) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING id`, profile, c.AssetID, c.Label, c.Amount, c.Date, c.Frequency, c.EndDate, c.Active, c.MerchantKey).Scan(&c.ID)
	} else {
		tag, e := s.pool.Exec(ctx, `UPDATE calendar_rules SET asset_id=$3,label=$4,amount_cents=$5,starts_on=$6,frequency=$7,ends_on=$8,active=$9,merchant_key=$10 WHERE id=$1 AND profile_id=$2`, c.ID, profile, c.AssetID, c.Label, c.Amount, c.Date, c.Frequency, c.EndDate, c.Active, c.MerchantKey)
		err = e
		if err == nil && tag.RowsAffected() == 0 {
			err = ErrNotFound
		}
	}
	return c, err
}
func (s *Store) DeleteCalendarRule(ctx context.Context, profile, id string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM calendar_rules WHERE id=$1 AND profile_id=$2`, id, profile)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}

// CalendarDates retains the original day-of-month, including after February.
// It is bounded independently from untrusted dates and a 100-year rule history.
func CalendarDates(c CalendarRule, from, until time.Time) []time.Time {
	start, err := time.Parse("2006-01-02", c.Date)
	if err != nil || !c.Active {
		return nil
	}
	end := until
	if c.EndDate != nil {
		d, e := time.Parse("2006-01-02", *c.EndDate)
		if e == nil && d.Before(end) {
			end = d
		}
	}
	dates := []time.Time{}
	for i := 0; i <= 36600; i++ {
		var d time.Time
		switch c.Frequency {
		case "once":
			if i > 0 {
				return dates
			}
			d = start
		case "weekly":
			d = start.AddDate(0, 0, 7*i)
		case "monthly":
			d = engine.CivilMonth(start, i)
		case "quarterly":
			d = engine.CivilMonth(start, 3*i)
		case "yearly":
			d = engine.CivilMonth(start, 12*i)
		default:
			return dates
		}
		if d.After(end) {
			break
		}
		if !d.Before(from) {
			dates = append(dates, d)
		}
	}
	return dates
}
func (s *Store) CalendarOccurrences(ctx context.Context, profile string, from, until time.Time) ([]CalendarOccurrence, error) {
	rules, err := s.ListCalendarRules(ctx, profile)
	if err != nil {
		return nil, err
	}
	out := []CalendarOccurrence{}
	for _, rule := range rules {
		for _, d := range CalendarDates(rule, from, until) {
			o := CalendarOccurrence{RuleID: rule.ID, AssetID: rule.AssetID, Label: rule.Label, Date: d.Format("2006-01-02"), Amount: rule.Amount, Status: "planned"}
			err = s.pool.QueryRow(ctx, `SELECT COALESCE(e.status,'planned'),COALESCE(e.amount_cents,$4),e.transaction_id,a.currency,amount_eur(COALESCE(e.amount_cents,$4),a.currency,LEAST($3::date,CURRENT_DATE),$1::uuid) FROM assets a LEFT JOIN calendar_occurrences e ON e.profile_id=$1 AND e.rule_id=$2 AND e.occurs_on=$3 WHERE a.profile_id=$1 AND a.id=$5`, profile, rule.ID, o.Date, rule.Amount, rule.AssetID).Scan(&o.Status, &o.Amount, &o.TransactionID, &o.Currency, &o.EUR)
			if err != nil {
				return nil, err
			}
			out = append(out, o)
		}
	}
	return out, nil
}
func (s *Store) SetCalendarOccurrence(ctx context.Context, profile, rule, date, status string, amount *int64, transaction *string) error {
	parsed, e := time.Parse("2006-01-02", date)
	if e != nil {
		return ErrInvalid
	}
	rules, e := s.ListCalendarRules(ctx, profile)
	if e != nil {
		return e
	}
	matched := false
	for _, c := range rules {
		if c.ID == rule {
			c.Active = true
			matched = len(CalendarDates(c, parsed, parsed)) == 1
			break
		}
	}
	if !matched {
		return ErrNotFound
	}
	// Verify both referenced resources before the upsert. Transaction ownership
	// and account equality are checked even when the caller knows its UUID.
	var ok bool
	err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM calendar_rules r WHERE r.id=$1 AND r.profile_id=$2 AND ($3::uuid IS NULL OR EXISTS(SELECT 1 FROM transactions t WHERE t.id=$3 AND t.profile_id=$2 AND t.asset_id=r.asset_id)))`, rule, profile, transaction).Scan(&ok)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotFound
	}
	_, err = s.pool.Exec(ctx, `INSERT INTO calendar_occurrences(profile_id,rule_id,occurs_on,status,amount_cents,transaction_id) VALUES($1,$2,$3,$4,$5,$6) ON CONFLICT(rule_id,occurs_on) DO UPDATE SET status=$4,amount_cents=$5,transaction_id=$6`, profile, rule, date, status, amount, transaction)
	return err
}

func (s *Store) CalendarMerchantKeys(ctx context.Context, profile string) ([]string, error) {
	rows, e := s.pool.Query(ctx, `SELECT DISTINCT merchant_key FROM calendar_rules WHERE profile_id=$1 AND active AND merchant_key<>''`, profile)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	keys := []string{}
	for rows.Next() {
		var k string
		if e = rows.Scan(&k); e != nil {
			return nil, e
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}
func (s *Store) RecurringExcludedKeys(ctx context.Context, profile string) (map[string]bool, error) {
	rows, err := s.pool.Query(ctx, `SELECT merchant_key FROM recurring_exclusions WHERE profile_id=$1 UNION SELECT merchant_key FROM calendar_rules WHERE profile_id=$1 AND active AND merchant_key<>''`, profile)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var k string
		if err = rows.Scan(&k); err != nil {
			return nil, err
		}
		out[k] = true
	}
	return out, rows.Err()
}
func (s *Store) ExcludeRecurring(ctx context.Context, profile, key string, excluded bool) error {
	if excluded {
		_, e := s.pool.Exec(ctx, `INSERT INTO recurring_exclusions(profile_id,merchant_key) VALUES($1,$2) ON CONFLICT DO NOTHING`, profile, key)
		return e
	}
	_, e := s.pool.Exec(ctx, `DELETE FROM recurring_exclusions WHERE profile_id=$1 AND merchant_key=$2`, profile, key)
	return e
}
