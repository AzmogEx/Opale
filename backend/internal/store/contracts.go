package store

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/opale-app/opale/internal/categorize"
	"github.com/opale-app/opale/internal/money"
)

type FinancialContract struct {
	ID             string                    `json:"id"`
	Revision       int64                     `json:"revision"`
	Name           string                    `json:"name"`
	Category       string                    `json:"category"`
	Amount         money.Cents               `json:"amount_cents"`
	Currency       string                    `json:"currency"`
	Frequency      string                    `json:"frequency"`
	NextDueDate    string                    `json:"next_due_date"`
	AssetID        string                    `json:"asset_id"`
	CalendarRuleID string                    `json:"calendar_rule_id"`
	MerchantKey    string                    `json:"merchant_key"`
	TrialEnd       string                    `json:"trial_end"`
	CommitmentEnd  string                    `json:"commitment_end"`
	RenewalDate    string                    `json:"renewal_date"`
	AutoRenew      bool                      `json:"auto_renew"`
	NoticeDays     int                       `json:"notice_days"`
	ReminderDays   int                       `json:"reminder_days"`
	Active         bool                      `json:"active"`
	Note           string                    `json:"note"`
	PriceSince     string                    `json:"price_since"`
	PendingPrice   *ContractPriceObservation `json:"pending_price"`
}
type ContractPriceObservation struct {
	TransactionID string      `json:"transaction_id"`
	Date          string      `json:"date"`
	Amount        money.Cents `json:"amount_cents"`
	Previous      money.Cents `json:"previous_cents"`
	AnnualDelta   money.Cents `json:"annual_delta_cents"`
}
type ContractPrice struct {
	ID          string      `json:"id"`
	Amount      money.Cents `json:"amount_cents"`
	Currency    string      `json:"currency"`
	EffectiveOn string      `json:"effective_on"`
	Source      string      `json:"source"`
}

const contractColumns = `id,revision,name,category,amount_cents,currency,frequency,next_due_date::text,COALESCE(asset_id::text,''),COALESCE(calendar_rule_id::text,''),merchant_key,COALESCE(trial_end::text,''),COALESCE(commitment_end::text,''),COALESCE(renewal_date::text,''),auto_renew,notice_days,reminder_days,active,note,price_since::text`

func scanContract(row pgx.Row) (FinancialContract, error) {
	var c FinancialContract
	e := row.Scan(&c.ID, &c.Revision, &c.Name, &c.Category, &c.Amount, &c.Currency, &c.Frequency, &c.NextDueDate, &c.AssetID, &c.CalendarRuleID, &c.MerchantKey, &c.TrialEnd, &c.CommitmentEnd, &c.RenewalDate, &c.AutoRenew, &c.NoticeDays, &c.ReminderDays, &c.Active, &c.Note, &c.PriceSince)
	return c, e
}
func validateContract(c FinancialContract) error {
	if !financialUUID(c.ID) || c.Revision < 0 || strings.TrimSpace(c.Name) == "" || len(c.Name) > 200 || len(c.Note) > 2000 || len(c.MerchantKey) > 200 || c.Amount <= 0 || c.Amount > maxOnboardingCents || !financialCurrency(c.Currency) || !financialFrequency(c.Frequency, false) || !financialDay(c.NextDueDate, false) || c.NoticeDays < 0 || c.NoticeDays > 365 || c.ReminderDays < 0 || c.ReminderDays > 90 {
		return setupInvalid("Contrat : nom, montant positif, devise, dates et périodicité valides requis.")
	}
	if !strings.Contains("|subscription|insurance|housing|utilities|other|", "|"+c.Category+"|") || c.Category == "" {
		return ErrInvalid
	}
	for _, d := range []string{c.TrialEnd, c.CommitmentEnd, c.RenewalDate} {
		if !financialDay(d, true) {
			return setupInvalid("Les dates du contrat doivent être comprises entre 1900 et dans dix ans.")
		}
	}
	if c.MerchantKey != "" && c.AssetID == "" {
		return setupInvalid("Sélectionne le compte débité pour surveiller les hausses de ce marchand.")
	}
	return nil
}
func (s *Store) contractRaw(ctx context.Context, profile, id string) (FinancialContract, error) {
	c, e := scanContract(s.pool.QueryRow(ctx, `SELECT `+contractColumns+` FROM financial_contracts WHERE profile_id=$1 AND id=$2`, profile, id))
	return c, domainNotFound(e)
}
func (s *Store) ListFinancialContracts(ctx context.Context, profile string) ([]FinancialContract, error) {
	rows, e := s.pool.Query(ctx, `SELECT `+contractColumns+` FROM financial_contracts WHERE profile_id=$1 ORDER BY active DESC,name,id`, profile)
	if e != nil {
		return nil, e
	}
	out := []FinancialContract{}
	for rows.Next() {
		c, e := scanContract(rows)
		if e != nil {
			rows.Close()
			return nil, e
		}
		out = append(out, c)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return nil, e
	}
	for i := range out {
		c := &out[i]
		c.NextDueDate = nextFinancialDate(c.NextDueDate, c.Frequency, c.Active, onboardingToday())
		c.PendingPrice, e = s.pendingContractPrice(ctx, profile, *c)
		if e != nil {
			return nil, e
		}
	}
	return out, nil
}
func (s *Store) pendingContractPrice(ctx context.Context, profile string, c FinancialContract) (*ContractPriceObservation, error) {
	if !c.Active || c.AssetID == "" || c.MerchantKey == "" {
		return nil, nil
	}
	var p ContractPriceObservation
	// Match an explicitly declared merchant AND account, only booked expenses.
	// Compare the latest payment (not an older high payment before a later correction).
	e := s.pool.QueryRow(ctx, `SELECT t.id,t.occurred_on::text,(-t.amount_cents::numeric)::bigint FROM transactions t
 JOIN assets a ON a.id=t.asset_id AND a.profile_id=t.profile_id
 WHERE t.profile_id=$1 AND t.asset_id=$2 AND t.merchant_key=$3 AND a.currency=$4
 AND t.flow_kind='expense_income' AND t.bank_status='booked' AND t.amount_cents<0 AND -t.amount_cents::numeric<=999999999999
 AND t.occurred_on>=$5::date AND t.occurred_on<=CURRENT_DATE
 ORDER BY t.occurred_on DESC,t.created_at DESC,t.id DESC LIMIT 1`, profile, c.AssetID, c.MerchantKey, c.Currency, c.PriceSince).Scan(&p.TransactionID, &p.Date, &p.Amount)
	if errors.Is(e, pgx.ErrNoRows) {
		return nil, nil
	}
	if e != nil {
		return nil, e
	}
	if p.Amount <= c.Amount {
		return nil, nil
	}
	var dismissed bool
	e = s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM contract_price_dismissals WHERE profile_id=$1 AND contract_id=$2 AND transaction_id=$3)`, profile, c.ID, p.TransactionID).Scan(&dismissed)
	if e != nil || dismissed {
		return nil, e
	}
	p.Previous = c.Amount
	multiplier := int64(12)
	if c.Frequency == "quarterly" {
		multiplier = 4
	}
	if c.Frequency == "yearly" {
		multiplier = 1
	}
	p.AnnualDelta = money.Cents(int64(p.Amount-c.Amount) * multiplier)
	return &p, nil
}
func (s *Store) SaveFinancialContract(ctx context.Context, profile string, c FinancialContract) (FinancialContract, error) {
	if e := validateContract(c); e != nil {
		return c, e
	}
	c.Name = strings.TrimSpace(c.Name)
	c.MerchantKey = categorize.MerchantKey(c.MerchantKey)
	var out FinancialContract
	e := s.Atomic(ctx, func(st *Store) error {
		if e := st.lockFinancialProfile(ctx, profile); e != nil {
			return e
		}
		old, e := st.contractRaw(ctx, profile, c.ID)
		create := errors.Is(e, ErrNotFound)
		if e != nil && !create {
			return e
		}
		if create && c.Revision != 0 || !create && c.Revision != old.Revision {
			return ErrFinancialConflict
		}
		if e = st.financialAccount(ctx, profile, c.AssetID, c.Currency, c.Active); e != nil {
			return e
		}
		previous := ""
		if !create {
			previous = old.CalendarRuleID
		}
		rule, e := st.syncFinancialRule(ctx, profile, previous, c.AssetID, c.Name, c.NextDueDate, c.Frequency, c.MerchantKey, -c.Amount, c.Active)
		if e != nil {
			return e
		}
		priceChanged := create || old.Amount != c.Amount || old.Currency != c.Currency
		c.PriceSince = old.PriceSince
		if priceChanged {
			c.PriceSince = onboardingToday().Format("2006-01-02")
		}
		if create {
			_, e = st.pool.Exec(ctx, `INSERT INTO financial_contracts(id,profile_id,name,category,amount_cents,currency,frequency,next_due_date,asset_id,calendar_rule_id,merchant_key,trial_end,commitment_end,renewal_date,auto_renew,notice_days,reminder_days,active,note,price_since)
    VALUES($1,$2,$3,$4,$5,$6,$7,$8,NULLIF($9,'')::uuid,NULLIF($10,'')::uuid,$11,NULLIF($12,'')::date,NULLIF($13,'')::date,NULLIF($14,'')::date,$15,$16,$17,$18,$19,$20)`, c.ID, profile, c.Name, c.Category, c.Amount, c.Currency, c.Frequency, c.NextDueDate, c.AssetID, rule, c.MerchantKey, c.TrialEnd, c.CommitmentEnd, c.RenewalDate, c.AutoRenew, c.NoticeDays, c.ReminderDays, c.Active, c.Note, c.PriceSince)
		} else {
			_, e = st.pool.Exec(ctx, `UPDATE financial_contracts SET revision=revision+1,name=$3,category=$4,amount_cents=$5,currency=$6,frequency=$7,next_due_date=$8,asset_id=NULLIF($9,'')::uuid,calendar_rule_id=NULLIF($10,'')::uuid,merchant_key=$11,trial_end=NULLIF($12,'')::date,commitment_end=NULLIF($13,'')::date,renewal_date=NULLIF($14,'')::date,auto_renew=$15,notice_days=$16,reminder_days=$17,active=$18,note=$19,price_since=$20,updated_at=now() WHERE id=$1 AND profile_id=$2`, c.ID, profile, c.Name, c.Category, c.Amount, c.Currency, c.Frequency, c.NextDueDate, c.AssetID, rule, c.MerchantKey, c.TrialEnd, c.CommitmentEnd, c.RenewalDate, c.AutoRenew, c.NoticeDays, c.ReminderDays, c.Active, c.Note, c.PriceSince)
		}
		if e != nil {
			return e
		}
		if priceChanged {
			_, e = st.pool.Exec(ctx, `INSERT INTO contract_prices(profile_id,contract_id,amount_cents,currency,effective_on,source) VALUES($1,$2,$3,$4,$5,'declared')`, profile, c.ID, c.Amount, c.Currency, c.PriceSince)
			if e != nil {
				return e
			}
		}
		out, e = st.contractRaw(ctx, profile, c.ID)
		return e
	})
	return out, e
}
func (s *Store) ContractPrices(ctx context.Context, profile, id string) ([]ContractPrice, error) {
	if _, e := s.contractRaw(ctx, profile, id); e != nil {
		return nil, e
	}
	rows, e := s.pool.Query(ctx, `SELECT id,amount_cents,currency,effective_on::text,source FROM contract_prices WHERE profile_id=$1 AND contract_id=$2 ORDER BY effective_on DESC,created_at DESC,id DESC`, profile, id)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []ContractPrice{}
	for rows.Next() {
		var p ContractPrice
		if e = rows.Scan(&p.ID, &p.Amount, &p.Currency, &p.EffectiveOn, &p.Source); e != nil {
			return nil, e
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
func (s *Store) ResolveContractPrice(ctx context.Context, profile, id, transaction string, revision int64, accept bool) error {
	return s.Atomic(ctx, func(st *Store) error {
		if e := st.lockFinancialProfile(ctx, profile); e != nil {
			return e
		}
		c, e := st.contractRaw(ctx, profile, id)
		if e != nil {
			return e
		}
		if c.Revision != revision {
			return ErrFinancialConflict
		}
		p, e := st.pendingContractPrice(ctx, profile, c)
		if e != nil {
			return e
		}
		if p == nil || p.TransactionID != transaction {
			return ErrFinancialConflict
		}
		if !accept {
			_, e = st.pool.Exec(ctx, `INSERT INTO contract_price_dismissals(profile_id,contract_id,transaction_id) VALUES($1,$2,$3) ON CONFLICT DO NOTHING`, profile, id, transaction)
			return e
		}
		c.Amount = p.Amount
		c.NextDueDate = nextFinancialDate(c.NextDueDate, c.Frequency, true, onboardingToday())
		saved, saveErr := st.SaveFinancialContract(ctx, profile, c)
		if saveErr != nil {
			return saveErr
		}
		// Confirming a booked payment on the forecast date must not count it twice.
		if p.Date == saved.NextDueDate && saved.CalendarRuleID != "" {
			var used bool
			if e = st.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM calendar_occurrences WHERE profile_id=$1 AND transaction_id=$2)`, profile, p.TransactionID).Scan(&used); e != nil {
				return e
			}
			status := "realized"
			transactionID := &p.TransactionID
			if used {
				status = "excluded"
				transactionID = nil
			}
			amount := -int64(p.Amount)
			if e = st.SetCalendarOccurrence(ctx, profile, saved.CalendarRuleID, p.Date, status, &amount, transactionID); e != nil {
				return e
			}
		}
		_, e = st.pool.Exec(ctx, `UPDATE financial_contracts SET price_since=$3 WHERE profile_id=$1 AND id=$2`, profile, id, p.Date)
		if e != nil {
			return e
		}
		// SaveFinancialContract recorded the new baseline; mark the matching entry observed.
		_, e = st.pool.Exec(ctx, `UPDATE contract_prices SET source='observed',effective_on=$3 WHERE id=(SELECT id FROM contract_prices WHERE profile_id=$1 AND contract_id=$2 ORDER BY created_at DESC,id DESC LIMIT 1)`, profile, id, p.Date)
		return e
	})
}
func (s *Store) DeleteFinancialContract(ctx context.Context, profile, id string, revision int64) error {
	return s.Atomic(ctx, func(st *Store) error {
		if e := st.lockFinancialProfile(ctx, profile); e != nil {
			return e
		}
		c, e := st.contractRaw(ctx, profile, id)
		if e != nil {
			return e
		}
		if c.Revision != revision {
			return ErrFinancialConflict
		}
		if e = st.stopFinancialRule(ctx, profile, c.CalendarRuleID, onboardingToday().AddDate(0, 0, -1).Format("2006-01-02")); e != nil {
			return e
		}
		_, e = st.pool.Exec(ctx, `DELETE FROM financial_contracts WHERE profile_id=$1 AND id=$2`, profile, id)
		return e
	})
}

// Helpers use civil dates and integer minor units; no market estimates.
type ContractAlert struct {
	ID         string `json:"id"`
	ContractID string `json:"contract_id"`
	Kind       string `json:"kind"`
	Severity   string `json:"severity"`
	Title      string `json:"title"`
	Detail     string `json:"detail"`
	Date       string `json:"date"`
}

func FinancialContractAlerts(contracts []FinancialContract, today time.Time) []ContractAlert {
	out := []ContractAlert{}
	for _, c := range contracts {
		if !c.Active {
			continue
		}
		add := func(kind, title, detail, date string) {
			if date != "" {
				out = append(out, ContractAlert{ID: c.ID + "|" + kind + "|" + date, ContractID: c.ID, Kind: kind, Severity: "warning", Title: title, Detail: c.Name + " — " + detail, Date: date})
			}
		}
		due := func(date string) bool {
			d, e := time.Parse("2006-01-02", date)
			return e == nil && !d.Before(today) && !d.After(today.AddDate(0, 0, c.ReminderDays))
		}
		if due(c.TrialEnd) {
			add("contract_trial", "Fin d’essai gratuit", "essai se terminant le "+c.TrialEnd, c.TrialEnd)
		}
		if due(c.CommitmentEnd) {
			add("contract_commitment", "Fin d’engagement", "engagement se terminant le "+c.CommitmentEnd, c.CommitmentEnd)
		}
		if c.AutoRenew && c.RenewalDate != "" {
			d, _ := time.Parse("2006-01-02", c.RenewalDate)
			deadline := d.AddDate(0, 0, -c.NoticeDays).Format("2006-01-02")
			if due(deadline) {
				add("contract_notice", "Préavis à envoyer", "date limite de préavis : "+deadline, deadline)
			}
			if due(c.RenewalDate) {
				add("contract_renewal", "Renouvellement à venir", "renouvellement le "+c.RenewalDate, c.RenewalDate)
			}
		}
		if p := c.PendingPrice; p != nil {
			out = append(out, ContractAlert{ID: c.ID + "|price|" + p.TransactionID, ContractID: c.ID, Kind: "subscription_price_increase", Severity: "warning", Title: "Hausse de prix à vérifier", Detail: c.Name + " — le dernier prélèvement dépasse le tarif déclaré. Vérifie le montant dans Contrats.", Date: p.Date})
		}
	}
	return out
}

// TrackCalendarContract enriches an existing forecast without duplicating it.
func (s *Store) TrackCalendarContract(ctx context.Context, profile, ruleID string) error {
	_, e := s.pool.Exec(ctx, `WITH inserted AS (
 INSERT INTO financial_contracts(id,profile_id,name,category,amount_cents,currency,frequency,next_due_date,asset_id,calendar_rule_id,merchant_key,active)
 SELECT gen_random_uuid(),r.profile_id,r.label,'subscription',-r.amount_cents,a.currency,r.frequency,r.starts_on,r.asset_id,r.id,r.merchant_key,r.active
 FROM calendar_rules r JOIN assets a ON a.id=r.asset_id AND a.profile_id=r.profile_id
 WHERE r.profile_id=$1 AND r.id=$2 AND r.amount_cents BETWEEN -999999999999 AND -1 AND r.frequency IN ('monthly','quarterly','yearly')
 ON CONFLICT(calendar_rule_id) DO NOTHING RETURNING profile_id,id,amount_cents,currency,price_since)
 INSERT INTO contract_prices(profile_id,contract_id,amount_cents,currency,effective_on,source)
 SELECT profile_id,id,amount_cents,currency,price_since,'declared' FROM inserted`, profile, ruleID)
	return e
}
