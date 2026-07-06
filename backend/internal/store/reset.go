package store

// Gestion du profil (Réglages) : réinitialisation des données et suppression.

import (
	"context"
	"fmt"
)

// ResetProfileData efface TOUT le contenu du profil — actifs, passifs,
// valorisations, mouvements, règles, enveloppes, objectifs, documents,
// contacts, banques, espaces créés — mais conserve le profil, sa session
// courante et son journal d'accès. Transactionnel : tout ou rien.
func (s *Store) ResetProfileData(ctx context.Context, profileID string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("ResetProfileData: %w", err)
	}
	defer tx.Rollback(ctx)

	// L'ordre suit les dépendances ; les détails (property/object/company)
	// et les valorisations tombent en cascade avec les actifs/passifs.
	statements := []string{
		`DELETE FROM transactions WHERE profile_id = $1`,
		`DELETE FROM merchant_rules WHERE profile_id = $1`,
		`DELETE FROM envelopes WHERE profile_id = $1`,
		`DELETE FROM goals WHERE profile_id = $1`,
		`DELETE FROM documents WHERE profile_id = $1`,
		`DELETE FROM contacts WHERE profile_id = $1`,
		`DELETE FROM bank_links WHERE profile_id = $1`,
		`DELETE FROM spaces WHERE created_by = $1`,
		`DELETE FROM space_members WHERE profile_id = $1`,
		`DELETE FROM valuations WHERE profile_id = $1`,
		`DELETE FROM assets WHERE profile_id = $1`,
		`DELETE FROM liabilities WHERE profile_id = $1`,
		`DELETE FROM categories WHERE profile_id = $1`,
	}
	for _, stmt := range statements {
		if _, err := tx.Exec(ctx, stmt, profileID); err != nil {
			return fmt.Errorf("ResetProfileData: %w", err)
		}
	}
	return tx.Commit(ctx)
}

// DeleteProfile supprime le profil et tout ce qui lui appartient (les
// clés étrangères cascadent), sessions comprises. Irréversible.
func (s *Store) DeleteProfile(ctx context.Context, profileID string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM profiles WHERE id = $1`, profileID)
	if err != nil {
		return fmt.Errorf("DeleteProfile: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
