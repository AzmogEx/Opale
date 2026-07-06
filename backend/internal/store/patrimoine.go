package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/opale-app/opale/internal/money"
)

// ─── Actifs ──────────────────────────────────────────────────────────────────

// CreateAsset crée un actif pour un profil.
func (s *Store) CreateAsset(ctx context.Context, profileID, name, kind, currency, note string) (Asset, error) {
	var a Asset
	err := s.pool.QueryRow(ctx, `
		INSERT INTO assets (profile_id, name, kind, currency, note)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, profile_id, name, kind, currency, note, archived, created_at, updated_at`,
		profileID, name, kind, currency, note,
	).Scan(&a.ID, &a.ProfileID, &a.Name, &a.Kind, &a.Currency, &a.Note, &a.Archived, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return Asset{}, fmt.Errorf("CreateAsset: %w", err)
	}
	return a, nil
}

// ListAssets renvoie les actifs d'un profil avec leur dernière valorisation.
func (s *Store) ListAssets(ctx context.Context, profileID string) ([]Asset, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT a.id, a.profile_id, a.name, a.kind, a.currency, a.note, a.archived,
		       a.created_at, a.updated_at, lv.value_cents,
		       -- Solde théorique (audit) : dernière valorisation + mouvements
		       -- postérieurs. Seulement pour les comptes de flux.
		       CASE WHEN a.kind IN ('checking', 'savings') AND lv.value_cents IS NOT NULL
		            THEN lv.value_cents + COALESCE((
		                SELECT SUM(t.amount_cents) FROM transactions t
		                WHERE t.asset_id = a.id AND t.occurred_on > lv.as_of
		            ), 0)
		       END AS theoretical_cents
		FROM assets a
		LEFT JOIN LATERAL (
			SELECT value_cents, as_of FROM valuations v
			WHERE v.asset_id = a.id
			ORDER BY v.as_of DESC, v.created_at DESC LIMIT 1
		) lv ON true
		WHERE a.profile_id = $1
		ORDER BY a.created_at`, profileID)
	if err != nil {
		return nil, fmt.Errorf("ListAssets: %w", err)
	}
	defer rows.Close()

	assets := []Asset{}
	for rows.Next() {
		var a Asset
		var v, theo *int64
		if err := rows.Scan(&a.ID, &a.ProfileID, &a.Name, &a.Kind, &a.Currency, &a.Note,
			&a.Archived, &a.CreatedAt, &a.UpdatedAt, &v, &theo); err != nil {
			return nil, fmt.Errorf("ListAssets scan: %w", err)
		}
		if v != nil {
			c := money.Cents(*v)
			a.LatestValue = &c
		}
		if theo != nil {
			c := money.Cents(*theo)
			a.TheoreticalValue = &c
		}
		assets = append(assets, a)
	}
	return assets, rows.Err()
}

// GetAsset renvoie un actif d'un profil par son ID.
func (s *Store) GetAsset(ctx context.Context, profileID, id string) (Asset, error) {
	var a Asset
	err := s.pool.QueryRow(ctx, `
		SELECT id, profile_id, name, kind, currency, note, archived, created_at, updated_at
		FROM assets WHERE id = $1 AND profile_id = $2`, id, profileID,
	).Scan(&a.ID, &a.ProfileID, &a.Name, &a.Kind, &a.Currency, &a.Note, &a.Archived, &a.CreatedAt, &a.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Asset{}, ErrNotFound
	}
	if err != nil {
		return Asset{}, fmt.Errorf("GetAsset: %w", err)
	}
	return a, nil
}

// UpdateAsset met à jour les champs modifiables d'un actif.
func (s *Store) UpdateAsset(ctx context.Context, profileID, id, name, note string, archived bool) (Asset, error) {
	var a Asset
	err := s.pool.QueryRow(ctx, `
		UPDATE assets SET name = $3, note = $4, archived = $5
		WHERE id = $1 AND profile_id = $2
		RETURNING id, profile_id, name, kind, currency, note, archived, created_at, updated_at`,
		id, profileID, name, note, archived,
	).Scan(&a.ID, &a.ProfileID, &a.Name, &a.Kind, &a.Currency, &a.Note, &a.Archived, &a.CreatedAt, &a.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Asset{}, ErrNotFound
	}
	if err != nil {
		return Asset{}, fmt.Errorf("UpdateAsset: %w", err)
	}
	return a, nil
}

// DeleteAsset supprime un actif (et ses valorisations, par cascade).
func (s *Store) DeleteAsset(ctx context.Context, profileID, id string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM assets WHERE id = $1 AND profile_id = $2`, id, profileID)
	if err != nil {
		return fmt.Errorf("DeleteAsset: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ─── Passifs ─────────────────────────────────────────────────────────────────

// CreateLiability crée un passif pour un profil.
func (s *Store) CreateLiability(ctx context.Context, profileID, name, kind, currency, note string) (Liability, error) {
	var l Liability
	err := s.pool.QueryRow(ctx, `
		INSERT INTO liabilities (profile_id, name, kind, currency, note)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, profile_id, name, kind, currency, note, archived, created_at, updated_at`,
		profileID, name, kind, currency, note,
	).Scan(&l.ID, &l.ProfileID, &l.Name, &l.Kind, &l.Currency, &l.Note, &l.Archived, &l.CreatedAt, &l.UpdatedAt)
	if err != nil {
		return Liability{}, fmt.Errorf("CreateLiability: %w", err)
	}
	return l, nil
}

// ListLiabilities renvoie les passifs d'un profil avec leur dernière valorisation.
func (s *Store) ListLiabilities(ctx context.Context, profileID string) ([]Liability, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT l.id, l.profile_id, l.name, l.kind, l.currency, l.note, l.archived,
		       l.created_at, l.updated_at, lv.value_cents
		FROM liabilities l
		LEFT JOIN LATERAL (
			SELECT value_cents FROM valuations v
			WHERE v.liability_id = l.id
			ORDER BY v.as_of DESC, v.created_at DESC LIMIT 1
		) lv ON true
		WHERE l.profile_id = $1
		ORDER BY l.created_at`, profileID)
	if err != nil {
		return nil, fmt.Errorf("ListLiabilities: %w", err)
	}
	defer rows.Close()

	liabs := []Liability{}
	for rows.Next() {
		var l Liability
		var v *int64
		if err := rows.Scan(&l.ID, &l.ProfileID, &l.Name, &l.Kind, &l.Currency, &l.Note,
			&l.Archived, &l.CreatedAt, &l.UpdatedAt, &v); err != nil {
			return nil, fmt.Errorf("ListLiabilities scan: %w", err)
		}
		if v != nil {
			c := money.Cents(*v)
			l.LatestValue = &c
		}
		liabs = append(liabs, l)
	}
	return liabs, rows.Err()
}

// GetLiability renvoie un passif d'un profil par son ID.
func (s *Store) GetLiability(ctx context.Context, profileID, id string) (Liability, error) {
	var l Liability
	err := s.pool.QueryRow(ctx, `
		SELECT id, profile_id, name, kind, currency, note, archived, created_at, updated_at
		FROM liabilities WHERE id = $1 AND profile_id = $2`, id, profileID,
	).Scan(&l.ID, &l.ProfileID, &l.Name, &l.Kind, &l.Currency, &l.Note, &l.Archived, &l.CreatedAt, &l.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Liability{}, ErrNotFound
	}
	if err != nil {
		return Liability{}, fmt.Errorf("GetLiability: %w", err)
	}
	return l, nil
}

// UpdateLiability met à jour les champs modifiables d'un passif.
func (s *Store) UpdateLiability(ctx context.Context, profileID, id, name, note string, archived bool) (Liability, error) {
	var l Liability
	err := s.pool.QueryRow(ctx, `
		UPDATE liabilities SET name = $3, note = $4, archived = $5
		WHERE id = $1 AND profile_id = $2
		RETURNING id, profile_id, name, kind, currency, note, archived, created_at, updated_at`,
		id, profileID, name, note, archived,
	).Scan(&l.ID, &l.ProfileID, &l.Name, &l.Kind, &l.Currency, &l.Note, &l.Archived, &l.CreatedAt, &l.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Liability{}, ErrNotFound
	}
	if err != nil {
		return Liability{}, fmt.Errorf("UpdateLiability: %w", err)
	}
	return l, nil
}

// DeleteLiability supprime un passif (et ses valorisations, par cascade).
func (s *Store) DeleteLiability(ctx context.Context, profileID, id string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM liabilities WHERE id = $1 AND profile_id = $2`, id, profileID)
	if err != nil {
		return fmt.Errorf("DeleteLiability: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ─── Valorisations ───────────────────────────────────────────────────────────

// AddAssetValuation ajoute une valorisation à un actif (vérifie l'appartenance).
func (s *Store) AddAssetValuation(ctx context.Context, profileID, assetID string, value money.Cents, asOf time.Time, note string) (Valuation, error) {
	return s.addValuation(ctx, `
		INSERT INTO valuations (profile_id, asset_id, value_cents, as_of, note)
		SELECT $1, a.id, $3, $4, $5 FROM assets a WHERE a.id = $2 AND a.profile_id = $1
		RETURNING id, profile_id, asset_id, liability_id, value_cents, as_of, note, created_at`,
		profileID, assetID, value, asOf, note)
}

// AddLiabilityValuation ajoute une valorisation à un passif (vérifie l'appartenance).
func (s *Store) AddLiabilityValuation(ctx context.Context, profileID, liabilityID string, value money.Cents, asOf time.Time, note string) (Valuation, error) {
	return s.addValuation(ctx, `
		INSERT INTO valuations (profile_id, liability_id, value_cents, as_of, note)
		SELECT $1, l.id, $3, $4, $5 FROM liabilities l WHERE l.id = $2 AND l.profile_id = $1
		RETURNING id, profile_id, asset_id, liability_id, value_cents, as_of, note, created_at`,
		profileID, liabilityID, value, asOf, note)
}

func (s *Store) addValuation(ctx context.Context, query, profileID, subjectID string, value money.Cents, asOf time.Time, note string) (Valuation, error) {
	var v Valuation
	var cents int64
	err := s.pool.QueryRow(ctx, query, profileID, subjectID, int64(value), asOf, note).
		Scan(&v.ID, &v.ProfileID, &v.AssetID, &v.LiabilityID, &cents, &v.AsOf, &v.Note, &v.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Valuation{}, ErrNotFound // sujet inexistant ou hors profil
	}
	if err != nil {
		return Valuation{}, fmt.Errorf("addValuation: %w", err)
	}
	v.Value = money.Cents(cents)
	return v, nil
}

// ListAssetValuations renvoie l'historique de valorisations d'un actif.
func (s *Store) ListAssetValuations(ctx context.Context, profileID, assetID string) ([]Valuation, error) {
	return s.listValuations(ctx, "asset_id", profileID, assetID)
}

// ListLiabilityValuations renvoie l'historique de valorisations d'un passif.
func (s *Store) ListLiabilityValuations(ctx context.Context, profileID, liabilityID string) ([]Valuation, error) {
	return s.listValuations(ctx, "liability_id", profileID, liabilityID)
}

func (s *Store) listValuations(ctx context.Context, column, profileID, subjectID string) ([]Valuation, error) {
	// column est une constante interne (jamais une entrée utilisateur).
	rows, err := s.pool.Query(ctx, fmt.Sprintf(`
		SELECT id, profile_id, asset_id, liability_id, value_cents, as_of, note, created_at
		FROM valuations
		WHERE profile_id = $1 AND %s = $2
		ORDER BY as_of DESC, created_at DESC`, column), profileID, subjectID)
	if err != nil {
		return nil, fmt.Errorf("listValuations: %w", err)
	}
	defer rows.Close()

	vals := []Valuation{}
	for rows.Next() {
		var v Valuation
		var cents int64
		if err := rows.Scan(&v.ID, &v.ProfileID, &v.AssetID, &v.LiabilityID, &cents, &v.AsOf, &v.Note, &v.CreatedAt); err != nil {
			return nil, fmt.Errorf("listValuations scan: %w", err)
		}
		v.Value = money.Cents(cents)
		vals = append(vals, v)
	}
	return vals, rows.Err()
}
