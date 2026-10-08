// Package store gère l'accès à PostgreSQL : pool de connexions, migrations et
// requêtes typées. C'est l'unique couche de persistance du backend.
package store

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store enveloppe un pool de connexions PostgreSQL.
type database interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
	Begin(context.Context) (pgx.Tx, error)
}
type Store struct {
	pool       database
	connection *pgxpool.Pool
}

// Atomic scopes related mutations to one transaction (nested calls use savepoints).
func (s *Store) Atomic(ctx context.Context, fn func(*Store) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err = fn(&Store{pool: tx}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// Snapshot gives all export/aggregate reads the same MVCC database version.
func (s *Store) Snapshot(ctx context.Context, fn func(*Store) error) error {
	return s.Atomic(ctx, func(st *Store) error {
		if _, err := st.pool.Exec(ctx, "SET TRANSACTION ISOLATION LEVEL REPEATABLE READ READ ONLY"); err != nil {
			return err
		}
		return fn(st)
	})
}

// New ouvre un pool de connexions et vérifie la connectivité.
func New(ctx context.Context, databaseURL string) (*Store, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("store: URL de base invalide : %w", err)
	}
	cfg.ConnConfig.RuntimeParams["timezone"] = "Europe/Paris"
	cfg.MaxConns = 10
	cfg.MaxConnLifetime = time.Hour

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("store: ouverture du pool : %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("store: connexion PostgreSQL impossible : %w", err)
	}

	return &Store{pool: pool, connection: pool}, nil
}

// Close ferme le pool de connexions.
func (s *Store) Close() {
	if s.connection != nil {
		s.connection.Close()
	}
}

// Ping vérifie la disponibilité de la base (utilisé par /readyz).
func (s *Store) Ping(ctx context.Context) error {
	var one int
	return s.pool.QueryRow(ctx, "SELECT 1").Scan(&one)
}
