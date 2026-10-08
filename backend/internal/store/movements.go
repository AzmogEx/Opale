package store

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/opale-app/opale/internal/money"
	"time"
)

// Transfer records both sides of a movement atomically. Amounts are explicit
// native minor units; a cross-currency transfer never invents an exchange rate.
type TransferInput struct {
	RequestID  string      `json:"request_id"`
	FromID     string      `json:"from_asset_id"`
	ToID       string      `json:"to_asset_id"`
	FromAmount money.Cents `json:"from_amount_cents"`
	ToAmount   money.Cents `json:"to_amount_cents"`
	Fee        money.Cents `json:"fee_cents"`
	Date       string      `json:"occurred_on"`
}

func (s *Store) Transfer(ctx context.Context, owner string, n TransferInput) ([]Transaction, error) {
	var result []Transaction
	e := s.Atomic(ctx, func(st *Store) error {
		if _, e := st.pool.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, owner+":"+n.RequestID); e != nil {
			return e
		}
		rows, e := st.pool.Query(ctx, txSelect+` WHERE t.profile_id=$1 AND t.transfer_id=$2 ORDER BY t.created_at,t.id`, owner, n.RequestID)
		if e != nil {
			return e
		}
		for rows.Next() {
			t, e := scanTransaction(rows)
			if e != nil {
				rows.Close()
				return e
			}
			result = append(result, t)
		}
		e = rows.Err()
		rows.Close()
		if e != nil {
			return e
		}
		if len(result) > 0 {
			var debit, credit, fee money.Cents
			for _, t := range result {
				if t.OccurredOn.Format("2006-01-02") != n.Date {
					return ErrInvalid
				}
				if t.FlowKind == "fee" {
					fee -= t.Amount
				} else if t.AssetID == n.FromID {
					debit -= t.Amount
				} else if t.AssetID == n.ToID {
					credit += t.Amount
				} else {
					return ErrInvalid
				}
			}
			if debit != n.FromAmount || credit != n.ToAmount || fee != n.Fee {
				return ErrInvalid
			}
			return nil
		}
		from, e := st.GetAsset(ctx, owner, n.FromID)
		if e != nil {
			return e
		}
		to, e := st.GetAsset(ctx, owner, n.ToID)
		if e != nil {
			return e
		}
		if from.Archived || to.Archived || (from.Kind != "checking" && from.Kind != "savings") || (to.Kind != "checking" && to.Kind != "savings") {
			return fmt.Errorf("%w: virement réservé à deux comptes actifs", ErrInvalid)
		}
		if from.Currency == to.Currency && n.FromAmount != n.ToAmount {
			return fmt.Errorf("%w: montants différents dans la même devise", ErrInvalid)
		}
		d, e := time.Parse("2006-01-02", n.Date)
		if e != nil {
			return e
		}
		for _, part := range []struct {
			asset  string
			amount money.Cents
			kind   string
		}{{n.FromID, -n.FromAmount, "internal_transfer"}, {n.ToID, n.ToAmount, "internal_transfer"}, {n.FromID, -n.Fee, "fee"}} {
			if part.amount == 0 {
				continue
			}
			t, e := st.CreateTransaction(ctx, owner, NewTransaction{AssetID: part.asset, Amount: part.amount, OccurredOn: d, Label: "Virement entre comptes", RawLabel: "Virement entre comptes", FlowKind: part.kind})
			if e != nil {
				return e
			}
			if _, e = st.pool.Exec(ctx, `UPDATE transactions SET transfer_id=$1 WHERE profile_id=$2 AND id=$3`, n.RequestID, owner, t.ID); e != nil {
				return e
			}
			result = append(result, t)
		}
		return nil
	})
	return result, e
}
func (s *Store) DeleteTransfer(ctx context.Context, owner, id string) error {
	tag, e := s.pool.Exec(ctx, `DELETE FROM transactions WHERE profile_id=$1 AND transfer_id=$2`, owner, id)
	if e == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return e
}
func (s *Store) isLinkedMovement(ctx context.Context, owner, id string) (bool, error) {
	var linked bool
	e := s.pool.QueryRow(ctx, `SELECT transfer_id IS NOT NULL FROM transactions WHERE profile_id=$1 AND id=$2`, owner, id).Scan(&linked)
	if errors.Is(e, pgx.ErrNoRows) {
		return false, ErrNotFound
	}
	return linked, e
}
