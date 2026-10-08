package store

import (
	"context"
	"github.com/opale-app/opale/internal/bank"
	"github.com/opale-app/opale/internal/money"
	"time"
)

type BankAccount struct {
	ID                string       `json:"id"`
	LinkID            string       `json:"link_id"`
	ProviderAccountID string       `json:"provider_account_id"`
	AssetID           *string      `json:"asset_id,omitempty"`
	Currency          string       `json:"currency"`
	Name              string       `json:"name"`
	Status            string       `json:"status"`
	LastSyncedAt      *time.Time   `json:"last_synced_at,omitempty"`
	Balance           *money.Cents `json:"balance_cents,omitempty"`
	BalanceDate       *string      `json:"balance_date,omitempty"`
	BalanceType       string       `json:"balance_type"`
	BalanceReconciled bool         `json:"balance_reconciled"`
	LastError         string       `json:"last_error"`
}

type BankPending struct {
	AccountID string      `json:"account_id"`
	Amount    money.Cents `json:"amount_cents"`
	Currency  string      `json:"currency"`
	Date      *string     `json:"occurred_on,omitempty"`
	Label     string      `json:"label"`
}

func (s *Store) ListBankPending(ctx context.Context, profile string) ([]BankPending, error) {
	rows, err := s.pool.Query(ctx, `SELECT p.bank_account_id,p.amount_cents,a.currency,p.occurred_on::text,p.label FROM bank_pending p JOIN bank_accounts a ON a.profile_id=p.profile_id AND a.id=p.bank_account_id WHERE p.profile_id=$1 ORDER BY p.occurred_on NULLS LAST,p.id`, profile)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []BankPending{}
	for rows.Next() {
		var p BankPending
		if err = rows.Scan(&p.AccountID, &p.Amount, &p.Currency, &p.Date, &p.Label); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) ListBankAccounts(ctx context.Context, profile string) ([]BankAccount, error) {
	rows, e := s.pool.Query(ctx, `SELECT id,link_id,provider_account_id,asset_id,currency,name,status,last_synced_at,balance_cents,balance_date::text,balance_type,balance_reconciled,last_error FROM bank_accounts WHERE profile_id=$1 ORDER BY link_id,id`, profile)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []BankAccount{}
	for rows.Next() {
		var a BankAccount
		if e = rows.Scan(&a.ID, &a.LinkID, &a.ProviderAccountID, &a.AssetID, &a.Currency, &a.Name, &a.Status, &a.LastSyncedAt, &a.Balance, &a.BalanceDate, &a.BalanceType, &a.BalanceReconciled, &a.LastError); e != nil {
			return nil, e
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
func (s *Store) DiscoverBankAccounts(ctx context.Context, profile, link string, ids []string) error {
	return s.Atomic(ctx, func(st *Store) error {
		for _, id := range ids {
			_, e := st.pool.Exec(ctx, `INSERT INTO bank_accounts(profile_id,link_id,provider_account_id,name) VALUES($1,$2,$3,'Compte bancaire') ON CONFLICT(profile_id,provider_account_id) DO UPDATE SET link_id=$2,status=CASE WHEN bank_accounts.asset_id IS NULL THEN 'needs_mapping' ELSE 'ready' END`, profile, link, id)
			if e != nil {
				return e
			}
		}
		_, e := st.pool.Exec(ctx, `UPDATE bank_accounts SET status='closed' WHERE profile_id=$1 AND link_id=$2 AND NOT(provider_account_id=ANY($3))`, profile, link, ids)
		return e
	})
}
func (s *Store) MapBankAccount(ctx context.Context, profile, id string, asset *string) error {
	return s.Atomic(ctx, func(st *Store) error {
		var previous, bound *string
		err := st.pool.QueryRow(ctx, `SELECT a.asset_id,b.asset_id FROM bank_accounts a LEFT JOIN bank_account_bindings b ON b.profile_id=a.profile_id AND b.provider_account_id=a.provider_account_id WHERE a.id=$1 AND a.profile_id=$2 FOR UPDATE OF a`, id, profile).Scan(&previous, &bound)
		if err != nil {
			return domainNotFound(err)
		}
		if asset != nil && *asset != "" && bound != nil && *bound != *asset {
			return ErrInvalid
		}
		if asset != nil && *asset != "" {
			var valid bool
			e := st.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM assets a JOIN bank_accounts b ON b.id=$3 AND b.profile_id=$1 WHERE a.profile_id=$1 AND a.id=$2 AND NOT a.archived AND a.kind IN('checking','savings') AND (b.currency='' OR b.currency=a.currency))`, profile, *asset, id).Scan(&valid)
			if e != nil {
				return e
			}
			if !valid {
				return ErrNotFound
			}
		} else {
			asset = nil
		}
		if (previous == nil && asset == nil) || (previous != nil && asset != nil && *previous == *asset) {
			return nil
		}
		tag, e := st.pool.Exec(ctx, `UPDATE bank_accounts SET asset_id=$3,status=CASE WHEN $3::uuid IS NULL THEN 'needs_mapping' ELSE 'ready' END WHERE id=$1 AND profile_id=$2`, id, profile, asset)
		if e != nil {
			return e
		}
		if tag.RowsAffected() == 0 {
			return ErrNotFound
		}
		_, e = st.pool.Exec(ctx, `UPDATE bank_links SET next_sync_at=now() WHERE profile_id=$1 AND id=(SELECT link_id FROM bank_accounts WHERE id=$2 AND profile_id=$1)`, profile, id)
		return e
	})
}
func (s *Store) ClaimBankLink(ctx context.Context, profile, id string) (bool, error) {
	tag, e := s.pool.Exec(ctx, `UPDATE bank_links SET lease_until=now()+interval '5 minutes',sync_status='syncing' WHERE id=$1 AND profile_id=$2 AND next_sync_at<=now() AND (lease_until IS NULL OR lease_until<now())`, id, profile)
	return tag.RowsAffected() > 0, e
}
func (s *Store) FinishBankLink(ctx context.Context, profile, id, status, message string, retry time.Duration) error {
	_, e := s.pool.Exec(ctx, `UPDATE bank_links SET sync_status=$3,last_error=$4,lease_until=NULL,next_sync_at=now()+$5*interval '1 second',attempts=CASE WHEN $3='synced' THEN 0 ELSE attempts+1 END,last_synced_at=CASE WHEN $3='synced' THEN now() ELSE last_synced_at END,status=CASE WHEN $3='synced' THEN 'linked' ELSE status END WHERE profile_id=$1 AND id=$2`, profile, id, status, message, int64(retry/time.Second))
	return e
}
func (s *Store) BankLink(ctx context.Context, profile, id string) (BankLink, error) {
	all, e := s.ListBankLinks(ctx, profile)
	if e != nil {
		return BankLink{}, e
	}
	for _, l := range all {
		if l.ID == id {
			return l, nil
		}
	}
	return BankLink{}, ErrNotFound
}
func (s *Store) RenewBankLink(ctx context.Context, profile, id, requisition string) error {
	tag, e := s.pool.Exec(ctx, `UPDATE bank_links SET requisition_id=$3,sync_status='pending_consent',last_error='',next_sync_at=now(),lease_until=NULL WHERE profile_id=$1 AND id=$2`, profile, id, requisition)
	if e == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return e
}
func (s *Store) MarkBankAccountError(ctx context.Context, profile, id, message string) error {
	_, e := s.pool.Exec(ctx, `UPDATE bank_accounts SET status='error',last_error=$3 WHERE profile_id=$1 AND id=$2`, profile, id, message)
	return e
}

func (s *Store) SaveBankAccountDetails(ctx context.Context, profile, id, currency, name string) error {
	_, e := s.pool.Exec(ctx, `UPDATE bank_accounts SET currency=$3,name=$4 WHERE profile_id=$1 AND id=$2`, profile, id, currency, name)
	return e
}
func (s *Store) CurrencyExponent(ctx context.Context, code string) (int, error) {
	var n int
	e := s.pool.QueryRow(ctx, `SELECT currency_exponent($1)`, code).Scan(&n)
	return n, e
}

// ApplyBankAccount atomically persists booked imports, replaces pending
// observations, and reconciles a dated booked balance. A pending operation is
// never a ledger transaction: its later booking therefore cannot double cash.
func (s *Store) ApplyBankAccount(ctx context.Context, profile string, a BankAccount, booked []NewTransaction, pending []NewTransaction, balance *bank.Balance, balanceValue money.Cents) (ImportResult, error) {
	result := ImportResult{}
	if a.AssetID == nil {
		return result, ErrInvalid
	}
	e := s.Atomic(ctx, func(st *Store) error {
		var current string
		if err := st.pool.QueryRow(ctx, `SELECT asset_id FROM bank_accounts WHERE id=$1 AND profile_id=$2 AND asset_id=$3 FOR UPDATE`, a.ID, profile, *a.AssetID).Scan(&current); err != nil {
			return domainNotFound(err)
		}
		var bound string
		if err := st.pool.QueryRow(ctx, `INSERT INTO bank_account_bindings(profile_id,provider_account_id,asset_id) VALUES($1,$2,$3) ON CONFLICT(profile_id,provider_account_id) DO UPDATE SET provider_account_id=EXCLUDED.provider_account_id RETURNING asset_id`, profile, a.ProviderAccountID, *a.AssetID).Scan(&bound); err != nil {
			return err
		}
		if bound != *a.AssetID {
			return ErrInvalid
		}
		var err error
		result, err = st.ImportTransactions(ctx, profile, booked)
		if err != nil {
			return err
		}
		if _, err = st.pool.Exec(ctx, `DELETE FROM bank_pending WHERE profile_id=$1 AND bank_account_id=$2`, profile, a.ID); err != nil {
			return err
		}
		for _, p := range pending {
			var date *time.Time
			if !p.OccurredOn.IsZero() {
				d := p.OccurredOn
				date = &d
			}
			_, err = st.pool.Exec(ctx, `INSERT INTO bank_pending(profile_id,bank_account_id,amount_cents,occurred_on,label) VALUES($1,$2,$3,$4,$5)`, profile, a.ID, p.Amount, date, p.Label)
			if err != nil {
				return err
			}
		}
		reconciled := false
		if balance != nil && !balance.CreditIncluded && (balance.Kind == "closingBooked" || balance.Kind == "interimBooked") {
			day, err := time.Parse("2006-01-02", balance.Date)
			if err != nil {
				return err
			}
			value := balanceValue
			if balance.Kind == "interimBooked" {
				for _, t := range booked {
					if t.OccurredOn.Equal(day) {
						value, err = money.Sub(value, t.Amount)
						if err != nil {
							return err
						}
					}
				}
				day = day.AddDate(0, 0, -1)
			}
			_, err = st.pool.Exec(ctx, `INSERT INTO valuations(profile_id,asset_id,value_cents,as_of,note) VALUES($1,$2,$3,$4,'Solde bancaire comptabilisé, clôture de journée')`, profile, *a.AssetID, value, day)
			if err != nil {
				return err
			}
			reconciled = true
		}
		var date *string
		var amount *money.Cents
		kind := ""
		if balance != nil {
			date = &balance.Date
			amount = &balanceValue
			kind = balance.Kind
		}
		_, err = st.pool.Exec(ctx, `UPDATE bank_accounts SET status='synced',last_synced_at=now(),last_error='',currency=$3,balance_cents=$4,balance_date=$5,balance_type=$6,balance_reconciled=$7 WHERE profile_id=$1 AND id=$2`, profile, a.ID, a.Currency, amount, date, kind, reconciled)
		return err
	})
	return result, e
}
