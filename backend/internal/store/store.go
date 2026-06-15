// Package store gère l'accès à PostgreSQL : pool de connexions, migrations et
// requêtes typées. C'est l'unique couche de persistance du backend.
package store

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Store enveloppe un pool de connexions PostgreSQL.
type Store struct {
	pool *pgxpool.Pool
}

// New ouvre un pool de connexions et vérifie la connectivité.
func New(ctx context.Context, databaseURL string) (*Store, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("store: URL de base invalide : %w", err)
	}
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

	return &Store{pool: pool}, nil
}

// Close ferme le pool de connexions.
func (s *Store) Close() { s.pool.Close() }

// Ping vérifie la disponibilité de la base (utilisé par /readyz).
func (s *Store) Ping(ctx context.Context) error { return s.pool.Ping(ctx) }
