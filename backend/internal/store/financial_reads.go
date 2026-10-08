package store

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/opale-app/opale/internal/money"
	"strings"
	"time"
)

func financialFilter(profileID string, f TransactionFilter) (string, []any) {
	where := []string{"t.profile_id=$1"}
	args := []any{profileID}
	add := func(expr string, v any) { args = append(args, v); where = append(where, fmt.Sprintf(expr, len(args))) }
	if f.From != nil {
		add("t.occurred_on >= $%d", *f.From)
	}
	if f.To != nil {
		add("t.occurred_on <= $%d", *f.To)
	}
	if f.Query != "" {
		add("(t.label ILIKE $%[1]d OR t.raw_label ILIKE $%[1]d)", "%"+f.Query+"%")
	}
	if f.CategoryID != "" {
		add("t.category_id=$%d", f.CategoryID)
	}
	if f.AssetID != "" {
		add("t.asset_id=$%d", f.AssetID)
	}
	return strings.Join(where, " AND "), args
}

// AggregateTransactions is intentionally independent of display pagination.
func (s *Store) AggregateTransactions(ctx context.Context, profileID string, f TransactionFilter, income bool) (int64, int, *Transaction, error) {
	where, args := financialFilter(profileID, f)
	sign := "<"
	if income {
		sign = ">"
	}
	where += " AND t.amount_cents " + sign + " 0"
	var total int64
	var count int
	err := s.pool.QueryRow(ctx, `SELECT COALESCE(SUM(abs(t.eur_cents::numeric)),0)::bigint,COUNT(*) FROM financial_transactions t WHERE `+where, args...).Scan(&total, &count)
	if err != nil {
		return 0, 0, nil, err
	}
	if count == 0 {
		return 0, 0, nil, nil
	}
	var t Transaction
	err = s.pool.QueryRow(ctx, `SELECT t.id,t.label,t.eur_cents,t.occurred_on FROM financial_transactions t WHERE `+where+` ORDER BY abs(t.eur_cents::numeric) DESC,t.id LIMIT 1`, args...).Scan(&t.ID, &t.Label, &t.Amount, &t.OccurredOn)
	if err != nil {
		return 0, 0, nil, err
	}
	return total, count, &t, nil
}

// Wrapped uses SQL grouping rather than loading a capped or unbounded transaction list.
func (s *Store) Wrapped(ctx context.Context, profileID string, year int) (map[string]any, error) {
	result := map[string]any{"year": year, "currency": "EUR"}
	err := s.Snapshot(ctx, func(st *Store) error {
		var income, expenses int64
		var count, months int
		err := st.pool.QueryRow(ctx, `SELECT COALESCE(SUM(eur_cents) FILTER(WHERE eur_cents>0),0),COALESCE(-SUM(eur_cents) FILTER(WHERE eur_cents<0),0),COUNT(*),COUNT(DISTINCT date_trunc('month',occurred_on)) FROM financial_transactions WHERE profile_id=$1 AND occurred_on>=make_date($2,1,1) AND occurred_on<make_date($2+1,1,1)`, profileID, year).Scan(&income, &expenses, &count, &months)
		if err != nil {
			return err
		}
		saved, err := money.Sub(money.Cents(income), money.Cents(expenses))
		if err != nil {
			return err
		}
		result["income_cents"] = income
		result["expenses_cents"] = expenses
		result["saved_cents"] = saved
		result["transaction_count"] = count
		result["active_months"] = months
		rate := 0
		if income > 0 {
			var r int64
			err = st.pool.QueryRow(ctx, "SELECT ($1::numeric*10000/$2)::bigint", saved, income).Scan(&r)
			if err != nil {
				return err
			}
			rate = int(r)
		}
		result["savings_rate_bps"] = rate
		for _, group := range []struct{ key, expr, join string }{{"top_categories", "COALESCE(c.name,'Sans catégorie')", "LEFT JOIN categories c ON c.id=t.category_id"}, {"top_merchants", "t.label", ""}} {
			rows, err := st.pool.Query(ctx, `SELECT `+group.expr+`,SUM(-t.eur_cents)::bigint FROM financial_transactions t `+group.join+` WHERE t.profile_id=$1 AND t.occurred_on>=make_date($2,1,1) AND t.occurred_on<make_date($2+1,1,1) AND t.eur_cents<0 GROUP BY 1 ORDER BY 2 DESC,1 LIMIT 5`, profileID, year)
			if err != nil {
				return err
			}
			entries := []map[string]any{}
			for rows.Next() {
				var name string
				var total int64
				if err = rows.Scan(&name, &total); err != nil {
					rows.Close()
					return err
				}
				entries = append(entries, map[string]any{"name": name, "total_cents": total})
			}
			err = rows.Err()
			rows.Close()
			if err != nil {
				return err
			}
			result[group.key] = entries
		}
		var label string
		var amount int64
		var date time.Time
		err = st.pool.QueryRow(ctx, `SELECT label,-eur_cents,occurred_on FROM financial_transactions WHERE profile_id=$1 AND occurred_on>=make_date($2,1,1) AND occurred_on<make_date($2+1,1,1) AND eur_cents<0 ORDER BY eur_cents,id LIMIT 1`, profileID, year).Scan(&label, &amount, &date)
		if err == nil {
			result["biggest_expense"] = map[string]any{"label": label, "total_cents": amount, "occurred_on": date.Format("2006-01-02")}
		} else if err != pgx.ErrNoRows {
			return err
		}
		for _, bound := range []struct{ key, order string }{{"net_worth_start_cents", "ASC"}, {"net_worth_end_cents", "DESC"}} {
			var value int64
			err = st.pool.QueryRow(ctx, `SELECT net_worth_cents FROM monthly_snapshots WHERE profile_id=$1 AND month>=make_date($2,1,1) AND month<make_date($2+1,1,1) ORDER BY month `+bound.order+` LIMIT 1`, profileID, year).Scan(&value)
			if err == nil {
				result[bound.key] = value
			} else if err != pgx.ErrNoRows {
				return err
			}
		}
		return nil
	})
	return result, err
}
