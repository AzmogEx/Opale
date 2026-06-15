package store

import (
	"context"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"github.com/opale-app/opale/internal/migrations"
)

// Migrate applique, dans l'ordre, toutes les migrations *.up.sql embarquées qui
// n'ont pas encore été exécutées. Idempotent : sûr à lancer à chaque démarrage.
func (s *Store) Migrate(ctx context.Context) error {
	if _, err := s.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version    TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`); err != nil {
		return fmt.Errorf("store: création de schema_migrations : %w", err)
	}

	entries, err := fs.ReadDir(migrations.FS, ".")
	if err != nil {
		return fmt.Errorf("store: lecture des migrations : %w", err)
	}

	var versions []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".up.sql") {
			versions = append(versions, strings.TrimSuffix(e.Name(), ".up.sql"))
		}
	}
	sort.Strings(versions)

	for _, v := range versions {
		var exists bool
		if err := s.pool.QueryRow(ctx,
			`SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)`, v,
		).Scan(&exists); err != nil {
			return fmt.Errorf("store: vérification migration %s : %w", v, err)
		}
		if exists {
			continue
		}

		sqlBytes, err := migrations.FS.ReadFile(v + ".up.sql")
		if err != nil {
			return fmt.Errorf("store: lecture migration %s : %w", v, err)
		}

		tx, err := s.pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("store: début transaction migration %s : %w", v, err)
		}
		if _, err := tx.Exec(ctx, string(sqlBytes)); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("store: exécution migration %s : %w", v, err)
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO schema_migrations (version) VALUES ($1)`, v,
		); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("store: enregistrement migration %s : %w", v, err)
		}
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("store: commit migration %s : %w", v, err)
		}
	}
	return nil
}
