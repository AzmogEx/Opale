package store

import (
	"context"
	"fmt"
	"time"

	"github.com/opale-app/opale/internal/money"
)

// ── Cours automatiques (opt-in) ───────────────────────────────────────────────

// QuotedAsset — un actif suivi automatiquement (crypto avec symbole CoinGecko).
type QuotedAsset struct {
	ID            string
	ProfileID     string
	Name          string
	Kind          string
	QuoteSymbol   string
	QuantityMicro int64 // millionièmes d'unité (0,5 BTC = 500000)
}

// ListQuotedAssets renvoie TOUS les actifs à cours automatique (tous profils :
// le job serveur tourne pour tout le foyer).
func (s *Store) ListQuotedAssets(ctx context.Context, owners ...string) ([]QuotedAsset, error) {
	owner := ""
	if len(owners) > 0 {
		owner = owners[0]
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, profile_id, name, kind, quote_symbol, quote_quantity_micro
		FROM assets
		WHERE quote_symbol <> '' AND quote_quantity_micro > 0 AND NOT archived AND ($1='' OR profile_id::text=$1)
		ORDER BY profile_id, created_at`, owner)
	if err != nil {
		return nil, fmt.Errorf("ListQuotedAssets: %w", err)
	}
	defer rows.Close()
	out := []QuotedAsset{}
	for rows.Next() {
		var q QuotedAsset
		if err := rows.Scan(&q.ID, &q.ProfileID, &q.Name, &q.Kind, &q.QuoteSymbol, &q.QuantityMicro); err != nil {
			return nil, fmt.Errorf("ListQuotedAssets scan: %w", err)
		}
		out = append(out, q)
	}
	return out, rows.Err()
}

// SetAssetQuote active (ou coupe, symbol vide) le suivi automatique d'un actif.
func (s *Store) SetAssetQuote(ctx context.Context, profileID, assetID, symbol string, quantityMicro int64) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE assets SET quote_symbol = $3, quote_quantity_micro = $4, updated_at = now()
		WHERE id = $2 AND profile_id = $1`, profileID, assetID, symbol, quantityMicro)
	if err != nil {
		return fmt.Errorf("SetAssetQuote: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// AutoValuationNote marque les valorisations écrites par le job de cours.
const AutoValuationNote = "Cours automatique"

// ReplaceAutoValuation écrit la valorisation du jour d'un actif suivi, en
// remplaçant celle déjà posée par le job (une seule ligne auto par jour).
func (s *Store) ReplaceAutoValuation(ctx context.Context, profileID, assetID string, value money.Cents, day time.Time) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("ReplaceAutoValuation: %w", err)
	}
	defer tx.Rollback(ctx)
	var locked string
	if e := tx.QueryRow(ctx, `SELECT id FROM assets WHERE profile_id=$1 AND id=$2 FOR UPDATE`, profileID, assetID).Scan(&locked); e != nil {
		return e
	}
	if _, err := tx.Exec(ctx, `
		DELETE FROM valuations WHERE asset_id = $1 AND as_of = $2 AND note = $3 AND profile_id=$4`,
		assetID, day, AutoValuationNote, profileID); err != nil {
		return fmt.Errorf("ReplaceAutoValuation delete: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO valuations (profile_id, asset_id, value_cents, as_of, note)
		SELECT $1, $2, native_from_eur($3,currency,$4,$1::uuid), $4, $5 FROM assets WHERE profile_id=$1 AND id=$2`,
		profileID, assetID, int64(value), day, AutoValuationNote); err != nil {
		return fmt.Errorf("ReplaceAutoValuation insert: %w", err)
	}
	return tx.Commit(ctx)
}

// ── Snapshots mensuels ────────────────────────────────────────────────────────

// MonthlySnapshot — un point mensuel du patrimoine, pris par le serveur.
type MonthlySnapshot struct {
	RecordedAt  time.Time   `json:"recorded_at"`
	Month       time.Time   `json:"month"`
	Assets      money.Cents `json:"assets_cents"`
	Liabilities money.Cents `json:"liabilities_cents"`
	NetWorth    money.Cents `json:"net_worth_cents"`
}

// UpsertMonthlySnapshot enregistre (ou remplace) le snapshot d'un mois.
func (s *Store) UpsertMonthlySnapshot(ctx context.Context, profileID string, month time.Time, assets, liabilities, netWorth money.Cents) error {
	first := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.UTC)
	_, err := s.pool.Exec(ctx, `
		INSERT INTO monthly_snapshots (profile_id, month, assets_cents, liabilities_cents, net_worth_cents)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (profile_id, month) DO UPDATE SET
			assets_cents = EXCLUDED.assets_cents,
			liabilities_cents = EXCLUDED.liabilities_cents,
			net_worth_cents = EXCLUDED.net_worth_cents,
			created_at = now()`,
		profileID, first, int64(assets), int64(liabilities), int64(netWorth))
	if err != nil {
		return fmt.Errorf("UpsertMonthlySnapshot: %w", err)
	}
	return nil
}

// HasMonthlySnapshot dit si le snapshot du mois existe déjà.
func (s *Store) HasMonthlySnapshot(ctx context.Context, profileID string, month time.Time) (bool, error) {
	first := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.UTC)
	var exists bool
	err := s.pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM monthly_snapshots WHERE profile_id = $1 AND month = $2)`,
		profileID, first).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("HasMonthlySnapshot: %w", err)
	}
	return exists, nil
}

// ListMonthlySnapshots renvoie les snapshots d'un profil (plus récent d'abord).
func (s *Store) ListMonthlySnapshots(ctx context.Context, profileID string, limit int) ([]MonthlySnapshot, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT month, assets_cents, liabilities_cents, net_worth_cents, created_at
		FROM monthly_snapshots WHERE profile_id = $1
		ORDER BY month DESC LIMIT $2`, profileID, limit)
	if err != nil {
		return nil, fmt.Errorf("ListMonthlySnapshots: %w", err)
	}
	defer rows.Close()
	out := []MonthlySnapshot{}
	for rows.Next() {
		var m MonthlySnapshot
		var a, l, n int64
		if err := rows.Scan(&m.Month, &a, &l, &n, &m.RecordedAt); err != nil {
			return nil, fmt.Errorf("ListMonthlySnapshots scan: %w", err)
		}
		m.Assets, m.Liabilities, m.NetWorth = money.Cents(a), money.Cents(l), money.Cents(n)
		out = append(out, m)
	}
	return out, rows.Err()
}

// ── Allocation cible ──────────────────────────────────────────────────────────

// AllocationTarget — cible d'une classe d'actifs, en points de base.
type AllocationTarget struct {
	Class     string `json:"class"` // stocks, real_estate, crypto, cash, other
	TargetBps int    `json:"target_bps"`
}

// SetAllocationTargets remplace les cibles du profil (tout ou rien).
func (s *Store) SetAllocationTargets(ctx context.Context, profileID string, targets []AllocationTarget) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("SetAllocationTargets: %w", err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `DELETE FROM allocation_targets WHERE profile_id = $1`, profileID); err != nil {
		return fmt.Errorf("SetAllocationTargets delete: %w", err)
	}
	for _, t := range targets {
		if _, err := tx.Exec(ctx, `
			INSERT INTO allocation_targets (profile_id, class, target_bps) VALUES ($1, $2, $3)`,
			profileID, t.Class, t.TargetBps); err != nil {
			return fmt.Errorf("SetAllocationTargets insert %s: %w", t.Class, err)
		}
	}
	return tx.Commit(ctx)
}

// ListAllocationTargets renvoie les cibles du profil.
func (s *Store) ListAllocationTargets(ctx context.Context, profileID string) ([]AllocationTarget, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT class, target_bps FROM allocation_targets WHERE profile_id = $1 ORDER BY class`, profileID)
	if err != nil {
		return nil, fmt.Errorf("ListAllocationTargets: %w", err)
	}
	defer rows.Close()
	out := []AllocationTarget{}
	for rows.Next() {
		var t AllocationTarget
		if err := rows.Scan(&t.Class, &t.TargetBps); err != nil {
			return nil, fmt.Errorf("ListAllocationTargets scan: %w", err)
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// ── Alertes personnalisables ──────────────────────────────────────────────────

// CustomAlert — un seuil surveillé par le moteur d'alertes.
type CustomAlert struct {
	ID        string      `json:"id"`
	Kind      string      `json:"kind"` // cash_below, net_worth_below, expenses_month_above
	Threshold money.Cents `json:"threshold_cents"`
	Enabled   bool        `json:"enabled"`
	CreatedAt time.Time   `json:"created_at"`
}

// CreateCustomAlert ajoute un seuil personnalisé.
func (s *Store) CreateCustomAlert(ctx context.Context, profileID, kind string, threshold money.Cents) (CustomAlert, error) {
	var a CustomAlert
	var cents int64
	err := s.pool.QueryRow(ctx, `
		INSERT INTO custom_alerts (profile_id, kind, threshold_cents) VALUES ($1, $2, $3)
		RETURNING id, kind, threshold_cents, enabled, created_at`,
		profileID, kind, int64(threshold)).
		Scan(&a.ID, &a.Kind, &cents, &a.Enabled, &a.CreatedAt)
	if err != nil {
		return CustomAlert{}, fmt.Errorf("CreateCustomAlert: %w", err)
	}
	a.Threshold = money.Cents(cents)
	return a, nil
}

// ListCustomAlerts renvoie les seuils du profil.
func (s *Store) ListCustomAlerts(ctx context.Context, profileID string, enabledOnly bool) ([]CustomAlert, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, kind, threshold_cents, enabled, created_at FROM custom_alerts
		WHERE profile_id = $1 AND (NOT $2 OR enabled)
		ORDER BY created_at`, profileID, enabledOnly)
	if err != nil {
		return nil, fmt.Errorf("ListCustomAlerts: %w", err)
	}
	defer rows.Close()
	out := []CustomAlert{}
	for rows.Next() {
		var a CustomAlert
		var cents int64
		if err := rows.Scan(&a.ID, &a.Kind, &cents, &a.Enabled, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("ListCustomAlerts scan: %w", err)
		}
		a.Threshold = money.Cents(cents)
		out = append(out, a)
	}
	return out, rows.Err()
}

// UpdateCustomAlert active/coupe ou re-seuil une alerte.
func (s *Store) UpdateCustomAlert(ctx context.Context, profileID, id string, threshold money.Cents, enabled bool) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE custom_alerts SET threshold_cents = $3, enabled = $4
		WHERE id = $2 AND profile_id = $1`, profileID, id, int64(threshold), enabled)
	if err != nil {
		return fmt.Errorf("UpdateCustomAlert: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteCustomAlert supprime un seuil.
func (s *Store) DeleteCustomAlert(ctx context.Context, profileID, id string) error {
	tag, err := s.pool.Exec(ctx, `
		DELETE FROM custom_alerts WHERE id = $2 AND profile_id = $1`, profileID, id)
	if err != nil {
		return fmt.Errorf("DeleteCustomAlert: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ── Jetons push (APNs) ────────────────────────────────────────────────────────

// UpsertPushToken enregistre le jeton APNs d'un appareil pour un profil.
func (s *Store) UpsertPushToken(ctx context.Context, profileID, token, platform string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO push_tokens (token, profile_id, platform) VALUES ($1, $2, $3)
		ON CONFLICT (token) DO UPDATE SET profile_id = EXCLUDED.profile_id, updated_at = now()`,
		token, profileID, platform)
	if err != nil {
		return fmt.Errorf("UpsertPushToken: %w", err)
	}
	return nil
}

// ListPushTokens renvoie les jetons d'un profil.
func (s *Store) ListPushTokens(ctx context.Context, profileID string) ([]string, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT token FROM push_tokens WHERE profile_id = $1`, profileID)
	if err != nil {
		return nil, fmt.Errorf("ListPushTokens: %w", err)
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			return nil, fmt.Errorf("ListPushTokens scan: %w", err)
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// DeletePushToken retire un jeton (désinscription ou jeton invalide côté APNs).
func (s *Store) DeletePushToken(ctx context.Context, token string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM push_tokens WHERE token = $1`, token)
	if err != nil {
		return fmt.Errorf("DeletePushToken: %w", err)
	}
	return nil
}
