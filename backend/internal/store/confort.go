package store

// Le confort (P7) : module entrepreneur (EF-036) et liens bancaires (EF-071).

import (
	"context"
	"fmt"
	"time"

	"github.com/opale-app/opale/internal/money"
)

// ── Entreprise (EF-036) ───────────────────────────────────────────────────────

// CompanyDetails — extension d'un actif company_share.
type CompanyDetails struct {
	AssetID         string      `json:"asset_id"`
	SIREN           string      `json:"siren"`
	OwnershipBps    int         `json:"ownership_bps"`
	CCA             money.Cents `json:"cca_cents"`
	CCAAssetID      *string     `json:"cca_asset_id,omitempty"`
	AnnualDividends money.Cents `json:"annual_dividends_cents"`
	MonthlySalary   money.Cents `json:"monthly_salary_cents"`
}

// Company — une société : l'actif (valorisation = valeur de ma part) + détails.
type Company struct {
	Asset   Asset          `json:"asset"`
	Details CompanyDetails `json:"details"`
}

// UpsertCompanyDetails crée ou met à jour les détails d'une société.
func (s *Store) UpsertCompanyDetails(ctx context.Context, profileID string, d CompanyDetails) error {
	return s.saveCompanyWithCCA(ctx, profileID, d)
}

// ListCompanies — toutes les sociétés du profil.
func (s *Store) ListCompanies(ctx context.Context, profileID string) ([]Company, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT a.id, a.name, a.kind, a.currency, a.note, a.archived, a.created_at, a.updated_at,
		       lv.value_cents,
		       COALESCE(cd.siren, ''), COALESCE(cd.ownership_bps, 10000),
		       CASE WHEN cd.cca_asset_id IS NOT NULL THEN current_asset_value(a.profile_id,cd.cca_asset_id,CURRENT_DATE) ELSE COALESCE(cd.cca_cents,0) END, COALESCE(cd.annual_dividends_cents, 0),
		       COALESCE(cd.monthly_salary_cents, 0), cd.cca_asset_id
		FROM assets a
		LEFT JOIN company_details cd ON cd.asset_id = a.id
		LEFT JOIN LATERAL (
			SELECT value_cents FROM valuations
			WHERE asset_id = a.id AND profile_id=a.profile_id AND as_of<=CURRENT_DATE ORDER BY as_of DESC, created_at DESC LIMIT 1
		) lv ON true
		WHERE a.profile_id = $1 AND a.kind = 'company_share' AND NOT a.archived
		ORDER BY a.created_at`,
		profileID,
	)
	if err != nil {
		return nil, fmt.Errorf("ListCompanies: %w", err)
	}
	defer rows.Close()

	out := []Company{}
	for rows.Next() {
		var c Company
		var latest *int64
		c.Asset.ProfileID = profileID
		if err := rows.Scan(
			&c.Asset.ID, &c.Asset.Name, &c.Asset.Kind, &c.Asset.Currency,
			&c.Asset.Note, &c.Asset.Archived, &c.Asset.CreatedAt, &c.Asset.UpdatedAt,
			&latest,
			&c.Details.SIREN, &c.Details.OwnershipBps,
			&c.Details.CCA, &c.Details.AnnualDividends, &c.Details.MonthlySalary, &c.Details.CCAAssetID,
		); err != nil {
			return nil, fmt.Errorf("ListCompanies: %w", err)
		}
		c.Details.AssetID = c.Asset.ID
		if latest != nil {
			v := money.Cents(*latest)
			c.Asset.LatestValue = &v
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// ── Liens bancaires (EF-071) ──────────────────────────────────────────────────

// BankLink — une banque connectée via GoCardless, rattachée à un actif.
type BankLink struct {
	ID              string     `json:"id"`
	SyncStatus      string     `json:"sync_status"`
	LastError       string     `json:"last_error"`
	NextSyncAt      time.Time  `json:"next_sync_at"`
	Attempts        int        `json:"attempts"`
	AssetID         string     `json:"asset_id"`
	AssetName       string     `json:"asset_name,omitempty"`
	RequisitionID   string     `json:"requisition_id"`
	InstitutionID   string     `json:"institution_id"`
	InstitutionName string     `json:"institution_name"`
	Status          string     `json:"status"` // created | linked
	LastSyncedAt    *time.Time `json:"last_synced_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

// CreateBankLink enregistre une réquisition envoyée à la banque.
func (s *Store) CreateBankLink(ctx context.Context, profileID string, l BankLink) (BankLink, error) {
	err := s.pool.QueryRow(ctx, `
		INSERT INTO bank_links (profile_id, asset_id, requisition_id, institution_id, institution_name)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, status, created_at`,
		profileID, l.AssetID, l.RequisitionID, l.InstitutionID, l.InstitutionName,
	).Scan(&l.ID, &l.Status, &l.CreatedAt)
	if err != nil {
		return BankLink{}, fmt.Errorf("CreateBankLink: %w", err)
	}
	return l, nil
}

// ListBankLinks — les banques connectées du profil.
func (s *Store) ListBankLinks(ctx context.Context, profileID string) ([]BankLink, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT bl.id, bl.asset_id, a.name, bl.requisition_id, bl.institution_id,
		       bl.institution_name, bl.status, bl.last_synced_at, bl.created_at, bl.sync_status,bl.last_error,bl.next_sync_at,bl.attempts
		FROM bank_links bl
		JOIN assets a ON a.id = bl.asset_id
		WHERE bl.profile_id = $1
		ORDER BY bl.created_at`,
		profileID,
	)
	if err != nil {
		return nil, fmt.Errorf("ListBankLinks: %w", err)
	}
	defer rows.Close()

	out := []BankLink{}
	for rows.Next() {
		var l BankLink
		if err := rows.Scan(&l.ID, &l.AssetID, &l.AssetName, &l.RequisitionID,
			&l.InstitutionID, &l.InstitutionName, &l.Status, &l.LastSyncedAt,
			&l.CreatedAt, &l.SyncStatus, &l.LastError, &l.NextSyncAt, &l.Attempts); err != nil {
			return nil, fmt.Errorf("ListBankLinks: %w", err)
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// MarkBankLinkSynced passe le lien en « linked » et date la synchro.
func (s *Store) MarkBankLinkSynced(ctx context.Context, profileID, id string) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE bank_links SET status = 'linked', last_synced_at = now()
		WHERE id = $1 AND profile_id = $2`, id, profileID)
	if err != nil {
		return fmt.Errorf("MarkBankLinkSynced: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteBankLink déconnecte une banque (les mouvements importés restent).
func (s *Store) DeleteBankLink(ctx context.Context, profileID, id string) error {
	tag, err := s.pool.Exec(ctx,
		`DELETE FROM bank_links WHERE id = $1 AND profile_id = $2`, id, profileID)
	if err != nil {
		return fmt.Errorf("DeleteBankLink: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
