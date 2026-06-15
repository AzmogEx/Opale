package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// ErrNotFound est renvoyé quand une entité demandée n'existe pas (ou n'appartient
// pas au profil courant).
var ErrNotFound = errors.New("introuvable")

// CreateProfile crée un profil avec un PIN déjà haché.
func (s *Store) CreateProfile(ctx context.Context, name, pinHash, privacyDefault string) (Profile, error) {
	var p Profile
	err := s.pool.QueryRow(ctx, `
		INSERT INTO profiles (name, pin_hash, privacy_default)
		VALUES ($1, $2, $3)
		RETURNING id, name, privacy_default, created_at, updated_at`,
		name, pinHash, privacyDefault,
	).Scan(&p.ID, &p.Name, &p.PrivacyDefault, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return Profile{}, fmt.Errorf("CreateProfile: %w", err)
	}
	return p, nil
}

// ListProfiles renvoie les profils (sans secret) pour l'écran de sélection.
func (s *Store) ListProfiles(ctx context.Context) ([]Profile, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, name, privacy_default, created_at, updated_at
		FROM profiles ORDER BY created_at`)
	if err != nil {
		return nil, fmt.Errorf("ListProfiles: %w", err)
	}
	defer rows.Close()

	profiles := []Profile{}
	for rows.Next() {
		var p Profile
		if err := rows.Scan(&p.ID, &p.Name, &p.PrivacyDefault, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("ListProfiles scan: %w", err)
		}
		profiles = append(profiles, p)
	}
	return profiles, rows.Err()
}

// GetProfile renvoie un profil par son ID.
func (s *Store) GetProfile(ctx context.Context, id string) (Profile, error) {
	var p Profile
	err := s.pool.QueryRow(ctx, `
		SELECT id, name, privacy_default, created_at, updated_at
		FROM profiles WHERE id = $1`, id,
	).Scan(&p.ID, &p.Name, &p.PrivacyDefault, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Profile{}, ErrNotFound
	}
	if err != nil {
		return Profile{}, fmt.Errorf("GetProfile: %w", err)
	}
	return p, nil
}

// ProfilePINHash renvoie le hash du PIN d'un profil (pour la connexion).
func (s *Store) ProfilePINHash(ctx context.Context, id string) (string, error) {
	var hash string
	err := s.pool.QueryRow(ctx, `SELECT pin_hash FROM profiles WHERE id = $1`, id).Scan(&hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("ProfilePINHash: %w", err)
	}
	return hash, nil
}
