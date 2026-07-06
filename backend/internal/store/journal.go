package store

// Journal d'accès (ENF-004) : « qui a consulté quoi » — argument de
// confidentialité du cahier des charges (§10).

import (
	"context"
	"fmt"
	"time"
)

// AccessEvent — une entrée du journal d'accès.
type AccessEvent struct {
	ID        string    `json:"id"`
	ProfileID *string   `json:"profile_id,omitempty"`
	Event     string    `json:"event"`
	Detail    string    `json:"detail"`
	At        time.Time `json:"at"`
}

// LogAccess trace un événement sensible. Ne bloque jamais l'appelant :
// une erreur de journalisation est renvoyée pour log, pas pour refuser l'action.
func (s *Store) LogAccess(ctx context.Context, profileID *string, event, detail string) error {
	_, err := s.pool.Exec(ctx,
		`INSERT INTO access_log (profile_id, event, detail) VALUES ($1, $2, $3)`,
		profileID, event, detail)
	if err != nil {
		return fmt.Errorf("LogAccess: %w", err)
	}
	return nil
}

// AccessLog — les derniers événements du profil (les échecs de connexion le
// visant inclus).
func (s *Store) AccessLog(ctx context.Context, profileID string, limit int) ([]AccessEvent, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, profile_id, event, detail, at
		FROM access_log
		WHERE profile_id = $1
		ORDER BY at DESC
		LIMIT $2`, profileID, limit)
	if err != nil {
		return nil, fmt.Errorf("AccessLog: %w", err)
	}
	defer rows.Close()

	var out []AccessEvent
	for rows.Next() {
		var e AccessEvent
		if err := rows.Scan(&e.ID, &e.ProfileID, &e.Event, &e.Detail, &e.At); err != nil {
			return nil, fmt.Errorf("AccessLog: %w", err)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// DeleteExpiredSessions purge les sessions expirées (appelé périodiquement).
func (s *Store) DeleteExpiredSessions(ctx context.Context) (int64, error) {
	tag, err := s.pool.Exec(ctx, `DELETE FROM sessions WHERE expires_at < now()`)
	if err != nil {
		return 0, fmt.Errorf("DeleteExpiredSessions: %w", err)
	}
	return tag.RowsAffected(), nil
}
