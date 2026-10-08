package store

import (
	"context"
	"errors"
)

var ErrHistoryRequired = errors.New("history must be preserved")

// Principal inserts acquire the same parent row lock. A valuation correction
// and a repayment therefore cannot validate against different debt states.
func (s *Store) lockLiability(ctx context.Context, owner, id string) error {
	var found string
	return domainNotFound(s.pool.QueryRow(ctx, `SELECT id FROM liabilities WHERE profile_id=$1 AND id=$2 FOR UPDATE`, owner, id).Scan(&found))
}
func (s *Store) validateLiabilityHistory(ctx context.Context, owner, id string) error {
	// A later snapshot can hide an earlier overpayment from the current total.
	// Balances can change only at snapshots or booked principal dates, so check
	// every such boundary, including future entries already present in the book.
	rows, err := s.pool.Query(ctx, `SELECT current_liability_value($1,$2,check_on) FROM (
	 SELECT as_of AS check_on FROM valuations WHERE profile_id=$1 AND liability_id=$2
	 UNION SELECT occurred_on FROM transactions WHERE profile_id=$1 AND linked_liability_id=$2 AND flow_kind='loan_principal' AND bank_status='booked'
	 UNION SELECT CURRENT_DATE
	) boundaries ORDER BY check_on`, owner, id)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var value int64
		if err = rows.Scan(&value); err != nil {
			return err
		}
	}
	return rows.Err()
}
func (s *Store) valuationLiability(ctx context.Context, owner, id string) (*string, error) {
	var liability *string
	e := s.pool.QueryRow(ctx, `SELECT liability_id FROM valuations WHERE profile_id=$1 AND id=$2`, owner, id).Scan(&liability)
	return liability, domainNotFound(e)
}
