package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/opale-app/opale/internal/engine"
	"github.com/opale-app/opale/internal/money"
)

// ── Enveloppes (EF-028) ───────────────────────────────────────────────────────

// Envelope — budget mensuel alloué à une catégorie.
type Envelope struct {
	ID           string      `json:"id"`
	CategoryID   string      `json:"category_id"`
	CategoryName string      `json:"category_name"`
	CategoryIcon string      `json:"category_icon"`
	Budget       money.Cents `json:"monthly_budget_cents"`
}

// EnvelopeStatus — l'enveloppe et son consommé du mois (recalculé, jamais stocké).
type EnvelopeStatus struct {
	Envelope
	Spent     money.Cents `json:"spent_cents"`     // dépensé ce mois (positif)
	Remaining money.Cents `json:"remaining_cents"` // budget − dépensé (peut être négatif)
}

// UpsertEnvelope crée ou met à jour l'enveloppe d'une catégorie.
func (s *Store) UpsertEnvelope(ctx context.Context, profileID, categoryID string, budget money.Cents) (Envelope, error) {
	var id string
	err := s.pool.QueryRow(ctx, `
		INSERT INTO envelopes (profile_id, category_id, monthly_budget_cents)
		VALUES ($1, $2, $3)
		ON CONFLICT (profile_id, category_id)
		DO UPDATE SET monthly_budget_cents = EXCLUDED.monthly_budget_cents
		RETURNING id`,
		profileID, categoryID, budget,
	).Scan(&id)
	if err != nil {
		return Envelope{}, fmt.Errorf("UpsertEnvelope: %w", err)
	}
	var e Envelope
	err = s.pool.QueryRow(ctx, `
		SELECT e.id, e.category_id, c.name, c.icon, e.monthly_budget_cents
		FROM envelopes e JOIN categories c ON c.id = e.category_id
		WHERE e.id = $1`, id,
	).Scan(&e.ID, &e.CategoryID, &e.CategoryName, &e.CategoryIcon, &e.Budget)
	if err != nil {
		return Envelope{}, fmt.Errorf("UpsertEnvelope: relecture : %w", err)
	}
	return e, nil
}

// DeleteEnvelope supprime une enveloppe du profil.
func (s *Store) DeleteEnvelope(ctx context.Context, profileID, id string) error {
	tag, err := s.pool.Exec(ctx,
		`DELETE FROM envelopes WHERE id = $1 AND profile_id = $2`, id, profileID)
	if err != nil {
		return fmt.Errorf("DeleteEnvelope: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// EnvelopeStatuses renvoie chaque enveloppe avec son consommé du mois donné.
func (s *Store) EnvelopeStatuses(ctx context.Context, profileID string, year int, month time.Month) ([]EnvelopeStatus, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT e.id, e.category_id, c.name, c.icon, e.monthly_budget_cents,
			COALESCE((
				SELECT -SUM(t.amount_cents)
				FROM transactions t
				WHERE t.profile_id = e.profile_id
				  AND t.category_id = e.category_id
				  AND t.amount_cents < 0
				  AND t.occurred_on >= make_date($2, $3, 1)
				  AND t.occurred_on < make_date($2, $3, 1) + interval '1 month'
			), 0) AS spent
		FROM envelopes e
		JOIN categories c ON c.id = e.category_id
		WHERE e.profile_id = $1
		ORDER BY c.name`,
		profileID, year, int(month))
	if err != nil {
		return nil, fmt.Errorf("EnvelopeStatuses: %w", err)
	}
	defer rows.Close()

	out := []EnvelopeStatus{}
	for rows.Next() {
		var st EnvelopeStatus
		if err := rows.Scan(&st.ID, &st.CategoryID, &st.CategoryName, &st.CategoryIcon,
			&st.Budget, &st.Spent); err != nil {
			return nil, fmt.Errorf("EnvelopeStatuses: scan: %w", err)
		}
		rem, err := money.Sub(st.Budget, st.Spent)
		if err != nil {
			return nil, err
		}
		st.Remaining = rem
		out = append(out, st)
	}
	return out, rows.Err()
}

// ── Données pour les moteurs P4 ───────────────────────────────────────────────

// RecurringObservations renvoie les mouvements des 18 derniers mois utilisables
// pour la détection de récurrence (EF-026).
func (s *Store) RecurringObservations(ctx context.Context, profileID string) ([]engine.TxObs, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT merchant_key, label, occurred_on, amount_cents
		FROM transactions
		WHERE profile_id = $1 AND merchant_key <> ''
		  AND occurred_on >= CURRENT_DATE - interval '18 months'
		ORDER BY occurred_on`, profileID)
	if err != nil {
		return nil, fmt.Errorf("RecurringObservations: %w", err)
	}
	defer rows.Close()

	var obs []engine.TxObs
	for rows.Next() {
		var o engine.TxObs
		if err := rows.Scan(&o.MerchantKey, &o.Label, &o.OccurredOn, &o.Amount); err != nil {
			return nil, fmt.Errorf("RecurringObservations: scan: %w", err)
		}
		obs = append(obs, o)
	}
	return obs, rows.Err()
}

// CashBalance — cash disponible : somme des dernières valorisations des
// comptes courants et livrets non archivés (EF-014/EF-027).
func (s *Store) CashBalance(ctx context.Context, profileID string) (money.Cents, error) {
	var cash int64
	err := s.pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(lv.value_cents), 0) FROM (
			SELECT DISTINCT ON (v.asset_id) v.value_cents
			FROM valuations v
			JOIN assets a ON a.id = v.asset_id
			WHERE v.profile_id = $1 AND a.archived = false
			  AND a.kind IN ('checking', 'savings')
			ORDER BY v.asset_id, v.as_of DESC, v.created_at DESC
		) lv`, profileID,
	).Scan(&cash)
	if err != nil {
		return 0, fmt.Errorf("CashBalance: %w", err)
	}
	return money.Cents(cash), nil
}

// AvgDailyVariableSpend — dépense variable moyenne par jour sur 90 jours,
// hors marchands récurrents (EF-027).
func (s *Store) AvgDailyVariableSpend(ctx context.Context, profileID string, excludeKeys []string) (money.Cents, error) {
	var total int64
	err := s.pool.QueryRow(ctx, `
		SELECT COALESCE(-SUM(amount_cents), 0)
		FROM transactions
		WHERE profile_id = $1 AND amount_cents < 0
		  AND occurred_on >= CURRENT_DATE - interval '90 days'
		  AND NOT (merchant_key = ANY($2))`,
		profileID, excludeKeys,
	).Scan(&total)
	if err != nil {
		return 0, fmt.Errorf("AvgDailyVariableSpend: %w", err)
	}
	return money.Cents(total / 90), nil
}

// FlowTotals3M — revenus et dépenses des 3 derniers mois (score de santé).
func (s *Store) FlowTotals3M(ctx context.Context, profileID string) (income, expenses money.Cents, err error) {
	var inc, exp int64
	err = s.pool.QueryRow(ctx, `
		SELECT
			COALESCE(SUM(amount_cents) FILTER (WHERE amount_cents > 0), 0),
			COALESCE(-SUM(amount_cents) FILTER (WHERE amount_cents < 0), 0)
		FROM transactions
		WHERE profile_id = $1 AND occurred_on >= CURRENT_DATE - interval '3 months'`,
		profileID,
	).Scan(&inc, &exp)
	if err != nil {
		return 0, 0, fmt.Errorf("FlowTotals3M: %w", err)
	}
	return money.Cents(inc), money.Cents(exp), nil
}

// AssetKindValues — valeur totale par type d'actif (diversification).
func (s *Store) AssetKindValues(ctx context.Context, profileID string) (map[string]money.Cents, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT kind, SUM(value_cents) FROM (
			SELECT DISTINCT ON (v.asset_id) a.kind, v.value_cents
			FROM valuations v
			JOIN assets a ON a.id = v.asset_id
			WHERE v.profile_id = $1 AND a.archived = false
			ORDER BY v.asset_id, v.as_of DESC, v.created_at DESC
		) lv GROUP BY kind`, profileID)
	if err != nil {
		return nil, fmt.Errorf("AssetKindValues: %w", err)
	}
	defer rows.Close()

	out := map[string]money.Cents{}
	for rows.Next() {
		var kind string
		var v int64
		if err := rows.Scan(&kind, &v); err != nil {
			return nil, fmt.Errorf("AssetKindValues: scan: %w", err)
		}
		out[kind] = money.Cents(v)
	}
	return out, rows.Err()
}

// ── Objectifs (EF-042) ────────────────────────────────────────────────────────

// Goal — un objectif de vie.
type Goal struct {
	ID         string      `json:"id"`
	Name       string      `json:"name"`
	Icon       string      `json:"icon"`
	Target     money.Cents `json:"target_cents"`
	TargetDate *time.Time  `json:"target_date,omitempty"`
	AssetID    *string     `json:"asset_id,omitempty"`
	AssetName  *string     `json:"asset_name,omitempty"`
	CreatedAt  time.Time   `json:"created_at"`
}

const goalSelect = `
	SELECT g.id, g.name, g.icon, g.target_cents, g.target_date, g.asset_id,
	       a.name, g.created_at
	FROM goals g
	LEFT JOIN assets a ON a.id = g.asset_id`

func scanGoal(row pgx.Row) (Goal, error) {
	var g Goal
	err := row.Scan(&g.ID, &g.Name, &g.Icon, &g.Target, &g.TargetDate,
		&g.AssetID, &g.AssetName, &g.CreatedAt)
	return g, err
}

// ListGoals renvoie les objectifs du profil.
func (s *Store) ListGoals(ctx context.Context, profileID string) ([]Goal, error) {
	rows, err := s.pool.Query(ctx, goalSelect+
		" WHERE g.profile_id = $1 ORDER BY g.created_at", profileID)
	if err != nil {
		return nil, fmt.Errorf("ListGoals: %w", err)
	}
	defer rows.Close()

	goals := []Goal{}
	for rows.Next() {
		g, err := scanGoal(rows)
		if err != nil {
			return nil, fmt.Errorf("ListGoals: scan: %w", err)
		}
		goals = append(goals, g)
	}
	return goals, rows.Err()
}

// CreateGoal crée un objectif.
func (s *Store) CreateGoal(ctx context.Context, profileID, name, icon string,
	target money.Cents, targetDate *time.Time, assetID *string) (Goal, error) {
	var id string
	err := s.pool.QueryRow(ctx, `
		INSERT INTO goals (profile_id, name, icon, target_cents, target_date, asset_id)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
		profileID, name, icon, target, targetDate, assetID,
	).Scan(&id)
	if err != nil {
		return Goal{}, fmt.Errorf("CreateGoal: %w", err)
	}
	g, err := scanGoal(s.pool.QueryRow(ctx, goalSelect+" WHERE g.id = $1", id))
	if err != nil {
		return Goal{}, fmt.Errorf("CreateGoal: relecture : %w", err)
	}
	return g, nil
}

// DeleteGoal supprime un objectif du profil.
func (s *Store) DeleteGoal(ctx context.Context, profileID, id string) error {
	tag, err := s.pool.Exec(ctx,
		`DELETE FROM goals WHERE id = $1 AND profile_id = $2`, id, profileID)
	if err != nil {
		return fmt.Errorf("DeleteGoal: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// AssetLatestValue — dernière valorisation d'un actif (progression d'objectif).
func (s *Store) AssetLatestValue(ctx context.Context, profileID, assetID string) (money.Cents, error) {
	var v int64
	err := s.pool.QueryRow(ctx, `
		SELECT value_cents FROM valuations
		WHERE profile_id = $1 AND asset_id = $2
		ORDER BY as_of DESC, created_at DESC LIMIT 1`,
		profileID, assetID,
	).Scan(&v)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("AssetLatestValue: %w", err)
	}
	return money.Cents(v), nil
}
