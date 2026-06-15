package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// CreateSession enregistre une session (jeton déjà haché) avec son expiration.
func (s *Store) CreateSession(ctx context.Context, profileID, tokenHash string, expiresAt time.Time) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO sessions (profile_id, token_hash, expires_at)
		VALUES ($1, $2, $3)`,
		profileID, tokenHash, expiresAt)
	if err != nil {
		return fmt.Errorf("CreateSession: %w", err)
	}
	return nil
}

// ProfileForSession renvoie le profil associé à un jeton valide et non expiré.
func (s *Store) ProfileForSession(ctx context.Context, tokenHash string) (Profile, error) {
	var p Profile
	err := s.pool.QueryRow(ctx, `
		SELECT p.id, p.name, p.privacy_default, p.created_at, p.updated_at
		FROM sessions s
		JOIN profiles p ON p.id = s.profile_id
		WHERE s.token_hash = $1 AND s.expires_at > now()`, tokenHash,
	).Scan(&p.ID, &p.Name, &p.PrivacyDefault, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Profile{}, ErrNotFound
	}
	if err != nil {
		return Profile{}, fmt.Errorf("ProfileForSession: %w", err)
	}
	return p, nil
}

// DeleteSession révoque une session (déconnexion).
func (s *Store) DeleteSession(ctx context.Context, tokenHash string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM sessions WHERE token_hash = $1`, tokenHash)
	if err != nil {
		return fmt.Errorf("DeleteSession: %w", err)
	}
	return nil
}
