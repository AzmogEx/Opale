package store

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"time"
)

type QuoteMetadata struct {
	Symbol        string     `json:"symbol"`
	QuantityMicro int64      `json:"quantity_micro"`
	Source        string     `json:"source"`
	AsOf          *time.Time `json:"as_of,omitempty"`
	LastError     string     `json:"last_error,omitempty"`
	Enabled       bool       `json:"enabled"`
}

func (s *Store) QuoteMetadata(ctx context.Context, owner, id string) (QuoteMetadata, error) {
	var m QuoteMetadata
	err := s.pool.QueryRow(ctx, `SELECT quote_symbol,quote_quantity_micro,quote_refreshed_at,quote_error FROM assets WHERE profile_id=$1 AND id=$2`, owner, id).Scan(&m.Symbol, &m.QuantityMicro, &m.AsOf, &m.LastError)
	if errors.Is(err, pgx.ErrNoRows) {
		err = ErrNotFound
	}
	m.Source = "CoinGecko"
	m.Enabled = m.Symbol != "" && m.QuantityMicro > 0
	return m, err
}
func (s *Store) QuoteResult(ctx context.Context, owner, id string, success bool) error {
	if success {
		_, e := s.pool.Exec(ctx, `UPDATE assets SET quote_refreshed_at=now(),quote_error='' WHERE profile_id=$1 AND id=$2`, owner, id)
		return e
	}
	_, e := s.pool.Exec(ctx, `UPDATE assets SET quote_error='Cours indisponible ou conversion impossible ; dernière valeur conservée.' WHERE profile_id=$1 AND id=$2`, owner, id)
	return e
}
