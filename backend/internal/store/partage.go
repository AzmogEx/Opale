package store

// Espace partagé (EF-007) : dépenses communes du foyer.
// Cloisonnement : seules les transactions explicitement marquées « communes »
// (space_id renseigné) deviennent visibles entre les membres de l'espace.

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/opale-app/opale/internal/money"
)

// SpaceMember — un membre d'un espace partagé.
type SpaceMember struct {
	ProfileID string    `json:"profile_id"`
	Name      string    `json:"name"`
	JoinedAt  time.Time `json:"joined_at"`
}

// Space — un espace partagé et ses membres.
type Space struct {
	ID        string        `json:"id"`
	Name      string        `json:"name"`
	CreatedBy string        `json:"created_by"`
	CreatedAt time.Time     `json:"created_at"`
	Members   []SpaceMember `json:"members"`
}

// CreateSpace crée un espace et y ajoute le créateur.
func (s *Store) CreateSpace(ctx context.Context, profileID, name string) (Space, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Space{}, fmt.Errorf("CreateSpace: %w", err)
	}
	defer tx.Rollback(ctx)

	var sp Space
	sp.Name, sp.CreatedBy = name, profileID
	err = tx.QueryRow(ctx,
		`INSERT INTO spaces (name, created_by) VALUES ($1, $2) RETURNING id, created_at`,
		name, profileID,
	).Scan(&sp.ID, &sp.CreatedAt)
	if err != nil {
		return Space{}, fmt.Errorf("CreateSpace: %w", err)
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO space_members (space_id, profile_id) VALUES ($1, $2)`,
		sp.ID, profileID); err != nil {
		return Space{}, fmt.Errorf("CreateSpace: membre fondateur : %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Space{}, fmt.Errorf("CreateSpace: %w", err)
	}
	return sp, nil
}

// ListSpaces — les espaces dont le profil est membre, avec leurs membres.
func (s *Store) ListSpaces(ctx context.Context, profileID string) ([]Space, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT sp.id, sp.name, sp.created_by, sp.created_at,
		       m.profile_id, p.name, m.joined_at
		FROM spaces sp
		JOIN space_members me ON me.space_id = sp.id AND me.profile_id = $1
		JOIN space_members m ON m.space_id = sp.id
		JOIN profiles p ON p.id = m.profile_id
		ORDER BY sp.created_at, m.joined_at`,
		profileID,
	)
	if err != nil {
		return nil, fmt.Errorf("ListSpaces: %w", err)
	}
	defer rows.Close()

	var out []Space
	index := map[string]int{}
	for rows.Next() {
		var sp Space
		var m SpaceMember
		if err := rows.Scan(&sp.ID, &sp.Name, &sp.CreatedBy, &sp.CreatedAt,
			&m.ProfileID, &m.Name, &m.JoinedAt); err != nil {
			return nil, fmt.Errorf("ListSpaces: %w", err)
		}
		i, ok := index[sp.ID]
		if !ok {
			out = append(out, sp)
			i = len(out) - 1
			index[sp.ID] = i
		}
		out[i].Members = append(out[i].Members, m)
	}
	return out, rows.Err()
}

// IsSpaceMember vérifie l'appartenance (cloisonnement des endpoints).
func (s *Store) IsSpaceMember(ctx context.Context, spaceID, profileID string) (bool, error) {
	var ok bool
	err := s.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM space_members WHERE space_id = $1 AND profile_id = $2)`,
		spaceID, profileID).Scan(&ok)
	if err != nil {
		return false, fmt.Errorf("IsSpaceMember: %w", err)
	}
	return ok, nil
}

// AddSpaceMember ajoute un profil du foyer à l'espace.
func (s *Store) AddSpaceMember(ctx context.Context, spaceID, profileID string) error {
	tag, err := s.pool.Exec(ctx, `
		INSERT INTO space_members (space_id, profile_id) VALUES ($1, $2)
		ON CONFLICT DO NOTHING`, spaceID, profileID)
	if err != nil {
		return fmt.Errorf("AddSpaceMember: %w", err)
	}
	_ = tag
	return nil
}

// RemoveSpaceMember retire un profil (ses dépenses communes sont détachées).
func (s *Store) RemoveSpaceMember(ctx context.Context, spaceID, profileID string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("RemoveSpaceMember: %w", err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `
		UPDATE transactions SET space_id = NULL
		WHERE space_id = $1 AND profile_id = $2`, spaceID, profileID); err != nil {
		return fmt.Errorf("RemoveSpaceMember: détachement : %w", err)
	}
	tag, err := tx.Exec(ctx,
		`DELETE FROM space_members WHERE space_id = $1 AND profile_id = $2`,
		spaceID, profileID)
	if err != nil {
		return fmt.Errorf("RemoveSpaceMember: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return tx.Commit(ctx)
}

// SetTransactionSpace marque/démarque une transaction comme dépense commune.
// La transaction doit appartenir au profil, et le profil à l'espace.
func (s *Store) SetTransactionSpace(ctx context.Context, profileID, txID string, spaceID *string) error {
	if spaceID != nil {
		ok, err := s.IsSpaceMember(ctx, *spaceID, profileID)
		if err != nil {
			return err
		}
		if !ok {
			return ErrNotFound
		}
	}
	tag, err := s.pool.Exec(ctx,
		`UPDATE transactions SET space_id = $1 WHERE id = $2 AND profile_id = $3`,
		spaceID, txID, profileID)
	if err != nil {
		return fmt.Errorf("SetTransactionSpace: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// SharedTransaction — une dépense commune, avec son payeur.
type SharedTransaction struct {
	ID         string      `json:"id"`
	PayerID    string      `json:"payer_id"`
	PayerName  string      `json:"payer_name"`
	Label      string      `json:"label"`
	Amount     money.Cents `json:"amount_cents"`
	OccurredOn time.Time   `json:"occurred_on"`
}

// SpaceTransactions — les dépenses communes de l'espace (accès réservé aux
// membres, vérifié par l'appelant via IsSpaceMember).
func (s *Store) SpaceTransactions(ctx context.Context, spaceID string, limit int) ([]SharedTransaction, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT t.id, t.profile_id, p.name, t.label, t.amount_cents, t.occurred_on
		FROM transactions t
		JOIN profiles p ON p.id = t.profile_id
		WHERE t.space_id = $1
		ORDER BY t.occurred_on DESC, t.created_at DESC
		LIMIT $2`, spaceID, limit)
	if err != nil {
		return nil, fmt.Errorf("SpaceTransactions: %w", err)
	}
	defer rows.Close()

	var out []SharedTransaction
	for rows.Next() {
		var t SharedTransaction
		var amount int64
		if err := rows.Scan(&t.ID, &t.PayerID, &t.PayerName, &t.Label, &amount, &t.OccurredOn); err != nil {
			return nil, fmt.Errorf("SpaceTransactions: %w", err)
		}
		t.Amount = money.Cents(amount)
		out = append(out, t)
	}
	return out, rows.Err()
}

// MemberBalance — la position d'un membre dans l'espace.
type MemberBalance struct {
	ProfileID string      `json:"profile_id"`
	Name      string      `json:"name"`
	Paid      money.Cents `json:"paid_cents"`    // dépenses communes payées
	Share     money.Cents `json:"share_cents"`   // quote-part (total / membres)
	Balance   money.Cents `json:"balance_cents"` // payé − quote-part (positif = on lui doit)
}

// SpaceBalance calcule la balance déterministe de l'espace : chaque membre
// doit une quote-part égale du total des dépenses communes. Division entière :
// la somme des balances peut différer du zéro de ± quelques centimes
// (reste de la division), assumé et documenté.
func (s *Store) SpaceBalance(ctx context.Context, spaceID string) ([]MemberBalance, money.Cents, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT m.profile_id, p.name,
		       COALESCE(-SUM(t.amount_cents) FILTER (WHERE t.amount_cents < 0), 0)
		FROM space_members m
		JOIN profiles p ON p.id = m.profile_id
		LEFT JOIN transactions t ON t.space_id = m.space_id AND t.profile_id = m.profile_id
		WHERE m.space_id = $1
		GROUP BY m.profile_id, p.name, m.joined_at
		ORDER BY m.joined_at`, spaceID)
	if err != nil {
		return nil, 0, fmt.Errorf("SpaceBalance: %w", err)
	}
	defer rows.Close()

	var members []MemberBalance
	var total int64
	for rows.Next() {
		var m MemberBalance
		var paid int64
		if err := rows.Scan(&m.ProfileID, &m.Name, &paid); err != nil {
			return nil, 0, fmt.Errorf("SpaceBalance: %w", err)
		}
		m.Paid = money.Cents(paid)
		total += paid
		members = append(members, m)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	if len(members) == 0 {
		return nil, 0, errors.Join(ErrNotFound, pgx.ErrNoRows)
	}

	share := total / int64(len(members))
	for i := range members {
		members[i].Share = money.Cents(share)
		members[i].Balance = members[i].Paid - money.Cents(share)
	}
	return members, money.Cents(total), nil
}
