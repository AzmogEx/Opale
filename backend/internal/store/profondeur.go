package store

// La profondeur (P6) : détails immobiliers (EF-033), objets (EF-035),
// coffre-fort (EF-064) et contacts de transmission (EF-063).

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/opale-app/opale/internal/money"
)

// ErrInvalid : la requête référence un objet du mauvais type.
var ErrInvalid = errors.New("type d'actif invalide pour cette opération")

// ── Immobilier (EF-033) ───────────────────────────────────────────────────────

// PropertyDetails — détails d'un bien immobilier (extension d'un actif).
type PropertyDetails struct {
	AssetID            string      `json:"asset_id"`
	PurchasePrice      money.Cents `json:"purchase_price_cents"`
	PurchaseDate       *time.Time  `json:"purchase_date,omitempty"`
	MonthlyRent        money.Cents `json:"monthly_rent_cents"`
	MonthlyCharges     money.Cents `json:"monthly_charges_cents"`
	PropertyTaxYearly  money.Cents `json:"property_tax_yearly_cents"`
	LiabilityID        *string     `json:"liability_id,omitempty"`
	MonthlyLoanPayment money.Cents `json:"monthly_loan_payment_cents"`
}

// Property — un bien complet : l'actif, ses détails, la dette adossée.
type Property struct {
	Asset   Asset           `json:"asset"`
	Details PropertyDetails `json:"details"`
	// LoanRemaining : dernière valorisation du passif lié (nil si aucun).
	LoanRemaining *money.Cents `json:"loan_remaining_cents,omitempty"`
	LoanName      string       `json:"loan_name,omitempty"`
}

// UpsertPropertyDetails crée ou met à jour les détails d'un bien.
// L'actif doit appartenir au profil et être de type real_estate.
func (s *Store) UpsertPropertyDetails(ctx context.Context, profileID string, d PropertyDetails) error {
	var kind string
	err := s.pool.QueryRow(ctx,
		`SELECT kind FROM assets WHERE id = $1 AND profile_id = $2`,
		d.AssetID, profileID).Scan(&kind)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("UpsertPropertyDetails: %w", err)
	}
	if kind != "real_estate" {
		return fmt.Errorf("%w: l'actif n'est pas un bien immobilier", ErrInvalid)
	}
	if d.LiabilityID != nil && *d.LiabilityID == "" {
		d.LiabilityID = nil
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO property_details (asset_id, profile_id, purchase_price_cents,
			purchase_date, monthly_rent_cents, monthly_charges_cents,
			property_tax_yearly_cents, liability_id, monthly_loan_payment_cents)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (asset_id) DO UPDATE SET
			purchase_price_cents = EXCLUDED.purchase_price_cents,
			purchase_date = EXCLUDED.purchase_date,
			monthly_rent_cents = EXCLUDED.monthly_rent_cents,
			monthly_charges_cents = EXCLUDED.monthly_charges_cents,
			property_tax_yearly_cents = EXCLUDED.property_tax_yearly_cents,
			liability_id = EXCLUDED.liability_id,
			monthly_loan_payment_cents = EXCLUDED.monthly_loan_payment_cents`,
		d.AssetID, profileID, int64(d.PurchasePrice), d.PurchaseDate,
		int64(d.MonthlyRent), int64(d.MonthlyCharges), int64(d.PropertyTaxYearly),
		d.LiabilityID, int64(d.MonthlyLoanPayment),
	)
	if err != nil {
		return fmt.Errorf("UpsertPropertyDetails: %w", err)
	}
	return nil
}

// ListProperties — tous les biens immobiliers du profil, avec leurs détails
// (zéro si non renseignés) et le restant dû du crédit lié.
func (s *Store) ListProperties(ctx context.Context, profileID string) ([]Property, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT a.id, a.name, a.kind, a.currency, a.note, a.archived, a.created_at, a.updated_at,
		       lv.value_cents,
		       COALESCE(pd.purchase_price_cents, 0), pd.purchase_date,
		       COALESCE(pd.monthly_rent_cents, 0), COALESCE(pd.monthly_charges_cents, 0),
		       COALESCE(pd.property_tax_yearly_cents, 0),
		       pd.liability_id, COALESCE(pd.monthly_loan_payment_cents, 0),
		       l.name,
		       llv.value_cents
		FROM assets a
		LEFT JOIN property_details pd ON pd.asset_id = a.id
		LEFT JOIN liabilities l ON l.id = pd.liability_id
		LEFT JOIN LATERAL (
			SELECT value_cents FROM valuations
			WHERE asset_id = a.id ORDER BY as_of DESC, created_at DESC LIMIT 1
		) lv ON true
		LEFT JOIN LATERAL (
			SELECT value_cents FROM valuations
			WHERE liability_id = pd.liability_id ORDER BY as_of DESC, created_at DESC LIMIT 1
		) llv ON true
		WHERE a.profile_id = $1 AND a.kind = 'real_estate' AND NOT a.archived
		ORDER BY a.created_at`,
		profileID,
	)
	if err != nil {
		return nil, fmt.Errorf("ListProperties: %w", err)
	}
	defer rows.Close()

	var out []Property
	for rows.Next() {
		var p Property
		var latest, loanRemaining *int64
		var loanName *string
		p.Asset.ProfileID = profileID
		if err := rows.Scan(
			&p.Asset.ID, &p.Asset.Name, &p.Asset.Kind, &p.Asset.Currency,
			&p.Asset.Note, &p.Asset.Archived, &p.Asset.CreatedAt, &p.Asset.UpdatedAt,
			&latest,
			&p.Details.PurchasePrice, &p.Details.PurchaseDate,
			&p.Details.MonthlyRent, &p.Details.MonthlyCharges,
			&p.Details.PropertyTaxYearly,
			&p.Details.LiabilityID, &p.Details.MonthlyLoanPayment,
			&loanName, &loanRemaining,
		); err != nil {
			return nil, fmt.Errorf("ListProperties: %w", err)
		}
		p.Details.AssetID = p.Asset.ID
		if latest != nil {
			v := money.Cents(*latest)
			p.Asset.LatestValue = &v
		}
		if loanRemaining != nil {
			v := money.Cents(*loanRemaining)
			p.LoanRemaining = &v
		}
		if loanName != nil {
			p.LoanName = *loanName
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// ── Objets de valeur (EF-035) ─────────────────────────────────────────────────

// ObjectDetails — détails d'un objet de valeur (extension d'un actif).
type ObjectDetails struct {
	AssetID       string      `json:"asset_id"`
	Category      string      `json:"category"`
	Brand         string      `json:"brand"`
	PurchasePrice money.Cents `json:"purchase_price_cents"`
	PurchaseDate  *time.Time  `json:"purchase_date,omitempty"`
	Insured       bool        `json:"insured"`
}

// ValuableObject — un objet complet : l'actif et ses détails.
type ValuableObject struct {
	Asset   Asset         `json:"asset"`
	Details ObjectDetails `json:"details"`
}

// objectKinds : types d'actifs acceptés comme « objets de valeur ».
var objectKinds = map[string]bool{"precious_metal": true, "vehicle": true, "valuable": true}

// UpsertObjectDetails crée ou met à jour les détails d'un objet.
func (s *Store) UpsertObjectDetails(ctx context.Context, profileID string, d ObjectDetails) error {
	var kind string
	err := s.pool.QueryRow(ctx,
		`SELECT kind FROM assets WHERE id = $1 AND profile_id = $2`,
		d.AssetID, profileID).Scan(&kind)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("UpsertObjectDetails: %w", err)
	}
	if !objectKinds[kind] {
		return fmt.Errorf("%w: l'actif n'est pas un objet de valeur", ErrInvalid)
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO object_details (asset_id, profile_id, category, brand,
			purchase_price_cents, purchase_date, insured)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (asset_id) DO UPDATE SET
			category = EXCLUDED.category,
			brand = EXCLUDED.brand,
			purchase_price_cents = EXCLUDED.purchase_price_cents,
			purchase_date = EXCLUDED.purchase_date,
			insured = EXCLUDED.insured`,
		d.AssetID, profileID, d.Category, d.Brand,
		int64(d.PurchasePrice), d.PurchaseDate, d.Insured,
	)
	if err != nil {
		return fmt.Errorf("UpsertObjectDetails: %w", err)
	}
	return nil
}

// ListObjects — tous les objets de valeur du profil.
func (s *Store) ListObjects(ctx context.Context, profileID string) ([]ValuableObject, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT a.id, a.name, a.kind, a.currency, a.note, a.archived, a.created_at, a.updated_at,
		       lv.value_cents,
		       COALESCE(od.category, ''), COALESCE(od.brand, ''),
		       COALESCE(od.purchase_price_cents, 0), od.purchase_date,
		       COALESCE(od.insured, false)
		FROM assets a
		LEFT JOIN object_details od ON od.asset_id = a.id
		LEFT JOIN LATERAL (
			SELECT value_cents FROM valuations
			WHERE asset_id = a.id ORDER BY as_of DESC, created_at DESC LIMIT 1
		) lv ON true
		WHERE a.profile_id = $1
		  AND a.kind IN ('precious_metal', 'vehicle', 'valuable')
		  AND NOT a.archived
		ORDER BY a.created_at`,
		profileID,
	)
	if err != nil {
		return nil, fmt.Errorf("ListObjects: %w", err)
	}
	defer rows.Close()

	var out []ValuableObject
	for rows.Next() {
		var o ValuableObject
		var latest *int64
		o.Asset.ProfileID = profileID
		if err := rows.Scan(
			&o.Asset.ID, &o.Asset.Name, &o.Asset.Kind, &o.Asset.Currency,
			&o.Asset.Note, &o.Asset.Archived, &o.Asset.CreatedAt, &o.Asset.UpdatedAt,
			&latest,
			&o.Details.Category, &o.Details.Brand,
			&o.Details.PurchasePrice, &o.Details.PurchaseDate, &o.Details.Insured,
		); err != nil {
			return nil, fmt.Errorf("ListObjects: %w", err)
		}
		o.Details.AssetID = o.Asset.ID
		if latest != nil {
			v := money.Cents(*latest)
			o.Asset.LatestValue = &v
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

// ── Coffre-fort (EF-064) ──────────────────────────────────────────────────────

// Document — métadonnées d'un document du coffre (jamais le contenu).
type Document struct {
	ID        string    `json:"id"`
	AssetID   *string   `json:"asset_id,omitempty"`
	AssetName string    `json:"asset_name,omitempty"`
	Name      string    `json:"name"`
	Kind      string    `json:"kind"`
	Mime      string    `json:"mime"`
	SizeBytes int64     `json:"size_bytes"`
	SHA256    string    `json:"sha256"`
	CreatedAt time.Time `json:"created_at"`
}

// DocumentKinds : valeurs autorisées (alignées avec le CHECK SQL).
var DocumentKinds = map[string]bool{
	"deed": true, "contract": true, "invoice": true, "identity": true,
	"insurance": true, "tax": true, "other": true,
}

// CreateDocument stocke un document déjà chiffré (le chiffrement est fait
// par la couche API — le store ne voit jamais le clair).
func (s *Store) CreateDocument(ctx context.Context, profileID string, d Document, encrypted []byte) (Document, error) {
	if d.AssetID != nil && *d.AssetID == "" {
		d.AssetID = nil
	}
	err := s.pool.QueryRow(ctx, `
		INSERT INTO documents (profile_id, asset_id, name, kind, mime, size_bytes, sha256, content)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at`,
		profileID, d.AssetID, d.Name, d.Kind, d.Mime, d.SizeBytes, d.SHA256, encrypted,
	).Scan(&d.ID, &d.CreatedAt)
	if err != nil {
		return Document{}, fmt.Errorf("CreateDocument: %w", err)
	}
	return d, nil
}

// ListDocuments — métadonnées de tous les documents du profil.
func (s *Store) ListDocuments(ctx context.Context, profileID string) ([]Document, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT d.id, d.asset_id, COALESCE(a.name, ''), d.name, d.kind, d.mime,
		       d.size_bytes, d.sha256, d.created_at
		FROM documents d
		LEFT JOIN assets a ON a.id = d.asset_id
		WHERE d.profile_id = $1
		ORDER BY d.created_at DESC`,
		profileID,
	)
	if err != nil {
		return nil, fmt.Errorf("ListDocuments: %w", err)
	}
	defer rows.Close()

	var out []Document
	for rows.Next() {
		var d Document
		if err := rows.Scan(&d.ID, &d.AssetID, &d.AssetName, &d.Name, &d.Kind,
			&d.Mime, &d.SizeBytes, &d.SHA256, &d.CreatedAt); err != nil {
			return nil, fmt.Errorf("ListDocuments: %w", err)
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// DocumentContent — métadonnées + contenu chiffré d'un document.
func (s *Store) DocumentContent(ctx context.Context, profileID, id string) (Document, []byte, error) {
	var d Document
	var encrypted []byte
	err := s.pool.QueryRow(ctx, `
		SELECT id, asset_id, name, kind, mime, size_bytes, sha256, created_at, content
		FROM documents WHERE id = $1 AND profile_id = $2`,
		id, profileID,
	).Scan(&d.ID, &d.AssetID, &d.Name, &d.Kind, &d.Mime, &d.SizeBytes,
		&d.SHA256, &d.CreatedAt, &encrypted)
	if errors.Is(err, pgx.ErrNoRows) {
		return Document{}, nil, ErrNotFound
	}
	if err != nil {
		return Document{}, nil, fmt.Errorf("DocumentContent: %w", err)
	}
	return d, encrypted, nil
}

// DeleteDocument supprime un document du coffre.
func (s *Store) DeleteDocument(ctx context.Context, profileID, id string) error {
	tag, err := s.pool.Exec(ctx,
		`DELETE FROM documents WHERE id = $1 AND profile_id = $2`, id, profileID)
	if err != nil {
		return fmt.Errorf("DeleteDocument: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ── Contacts de transmission (EF-063) ─────────────────────────────────────────

// Contact — une personne à contacter (notaire, banquier, proche…).
type Contact struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Role      string    `json:"role"`
	Phone     string    `json:"phone"`
	Email     string    `json:"email"`
	Note      string    `json:"note"`
	CreatedAt time.Time `json:"created_at"`
}

// ContactRoles : valeurs autorisées (alignées avec le CHECK SQL).
var ContactRoles = map[string]bool{
	"notary": true, "banker": true, "insurer": true,
	"accountant": true, "trusted": true, "other": true,
}

// CreateContact ajoute un contact de transmission.
func (s *Store) CreateContact(ctx context.Context, profileID string, c Contact) (Contact, error) {
	err := s.pool.QueryRow(ctx, `
		INSERT INTO contacts (profile_id, name, role, phone, email, note)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at`,
		profileID, c.Name, c.Role, c.Phone, c.Email, c.Note,
	).Scan(&c.ID, &c.CreatedAt)
	if err != nil {
		return Contact{}, fmt.Errorf("CreateContact: %w", err)
	}
	return c, nil
}

// ListContacts — tous les contacts du profil, notaire d'abord.
func (s *Store) ListContacts(ctx context.Context, profileID string) ([]Contact, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, name, role, phone, email, note, created_at
		FROM contacts WHERE profile_id = $1
		ORDER BY CASE role
			WHEN 'notary' THEN 0 WHEN 'trusted' THEN 1 WHEN 'banker' THEN 2
			WHEN 'insurer' THEN 3 WHEN 'accountant' THEN 4 ELSE 5
		END, name`,
		profileID,
	)
	if err != nil {
		return nil, fmt.Errorf("ListContacts: %w", err)
	}
	defer rows.Close()

	var out []Contact
	for rows.Next() {
		var c Contact
		if err := rows.Scan(&c.ID, &c.Name, &c.Role, &c.Phone, &c.Email,
			&c.Note, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("ListContacts: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// DeleteContact supprime un contact.
func (s *Store) DeleteContact(ctx context.Context, profileID, id string) error {
	tag, err := s.pool.Exec(ctx,
		`DELETE FROM contacts WHERE id = $1 AND profile_id = $2`, id, profileID)
	if err != nil {
		return fmt.Errorf("DeleteContact: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ── Timeline (EF-045) : premières valorisations ───────────────────────────────

// FirstValuation — première valorisation connue d'un actif (acquisition).
type FirstValuation struct {
	AssetID   string
	AssetName string
	AssetKind string
	Value     money.Cents
	AsOf      time.Time
}

// FirstValuations — la première valorisation de chaque actif du profil
// (sert d'événement « acquisition » sur la timeline patrimoniale).
func (s *Store) FirstValuations(ctx context.Context, profileID string) ([]FirstValuation, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT DISTINCT ON (v.asset_id)
		       v.asset_id, a.name, a.kind, v.value_cents, v.as_of
		FROM valuations v
		JOIN assets a ON a.id = v.asset_id
		WHERE v.profile_id = $1 AND v.asset_id IS NOT NULL AND NOT a.archived
		ORDER BY v.asset_id, v.as_of ASC, v.created_at ASC`,
		profileID,
	)
	if err != nil {
		return nil, fmt.Errorf("FirstValuations: %w", err)
	}
	defer rows.Close()

	var out []FirstValuation
	for rows.Next() {
		var f FirstValuation
		var v int64
		if err := rows.Scan(&f.AssetID, &f.AssetName, &f.AssetKind, &v, &f.AsOf); err != nil {
			return nil, fmt.Errorf("FirstValuations: %w", err)
		}
		f.Value = money.Cents(v)
		out = append(out, f)
	}
	return out, rows.Err()
}
