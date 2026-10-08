package store

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"time"
)

func domainNotFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

type Beneficiary struct {
	ID         string `json:"id"`
	ContactID  string `json:"contact_id"`
	DocumentID string `json:"document_id"`
	ShareBps   int    `json:"share_bps"`
	Note       string `json:"note"`
}

func (s *Store) ListBeneficiaries(ctx context.Context, profile string) ([]Beneficiary, error) {
	rows, e := s.pool.Query(ctx, `SELECT id,contact_id,document_id,share_bps,note FROM beneficiaries WHERE profile_id=$1 ORDER BY id`, profile)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []Beneficiary{}
	for rows.Next() {
		var b Beneficiary
		if e = rows.Scan(&b.ID, &b.ContactID, &b.DocumentID, &b.ShareBps, &b.Note); e != nil {
			return nil, e
		}
		out = append(out, b)
	}
	return out, rows.Err()
}
func (s *Store) SaveBeneficiary(ctx context.Context, profile string, b Beneficiary) (Beneficiary, error) {
	tx, e := s.pool.Begin(ctx)
	if e != nil {
		return b, e
	}
	defer tx.Rollback(ctx)
	var doc string
	e = tx.QueryRow(ctx, `SELECT id FROM documents WHERE id=$1 AND profile_id=$2 FOR UPDATE`, b.DocumentID, profile).Scan(&doc)
	if e != nil {
		return b, domainNotFound(e)
	}
	var contact string
	e = tx.QueryRow(ctx, `SELECT id FROM contacts WHERE id=$1 AND profile_id=$2`, b.ContactID, profile).Scan(&contact)
	if e != nil {
		return b, domainNotFound(e)
	}
	var sum int
	e = tx.QueryRow(ctx, `SELECT COALESCE(SUM(share_bps),0) FROM beneficiaries WHERE profile_id=$1 AND document_id=$2 AND contact_id<>$3`, profile, b.DocumentID, b.ContactID).Scan(&sum)
	if e != nil {
		return b, e
	}
	if b.ShareBps <= 0 || sum+b.ShareBps > 10000 {
		return b, ErrInvalid
	}
	e = tx.QueryRow(ctx, `INSERT INTO beneficiaries(profile_id,contact_id,document_id,share_bps,note) VALUES($1,$2,$3,$4,$5) ON CONFLICT(profile_id,contact_id,document_id) DO UPDATE SET share_bps=$4,note=$5 RETURNING id`, profile, b.ContactID, b.DocumentID, b.ShareBps, b.Note).Scan(&b.ID)
	if e != nil {
		return b, e
	}
	return b, tx.Commit(ctx)
}
func (s *Store) DeleteBeneficiary(ctx context.Context, profile, id string) error {
	tag, e := s.pool.Exec(ctx, `DELETE FROM beneficiaries WHERE id=$1 AND profile_id=$2`, id, profile)
	if e == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return e
}

type EmergencyGrant struct {
	ID                 string    `json:"id"`
	OwnerProfileID     string    `json:"owner_profile_id"`
	RecipientProfileID string    `json:"recipient_profile_id"`
	AssetIDs           []string  `json:"asset_ids"`
	DocumentIDs        []string  `json:"document_ids"`
	Active             bool      `json:"active"`
	ExpiresAt          time.Time `json:"expires_at"`
	CreatedAt          time.Time `json:"created_at"`
}

const grantSelect = `SELECT id,owner_profile_id,recipient_profile_id,asset_ids,document_ids,active,expires_at,created_at FROM emergency_grants`

func scanGrant(row pgx.Row) (EmergencyGrant, error) {
	var g EmergencyGrant
	e := row.Scan(&g.ID, &g.OwnerProfileID, &g.RecipientProfileID, &g.AssetIDs, &g.DocumentIDs, &g.Active, &g.ExpiresAt, &g.CreatedAt)
	return g, domainNotFound(e)
}
func (s *Store) ListEmergencyGrants(ctx context.Context, profile string) ([]EmergencyGrant, error) {
	rows, e := s.pool.Query(ctx, grantSelect+` WHERE owner_profile_id=$1 OR recipient_profile_id=$1 ORDER BY created_at DESC`, profile)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []EmergencyGrant{}
	for rows.Next() {
		g, e := scanGrant(rows)
		if e != nil {
			return nil, e
		}
		out = append(out, g)
	}
	return out, rows.Err()
}
func (s *Store) CreateEmergencyGrant(ctx context.Context, owner string, g EmergencyGrant) (EmergencyGrant, error) {
	if g.RecipientProfileID == owner || g.ExpiresAt.Before(time.Now()) || g.ExpiresAt.After(time.Now().AddDate(1, 0, 0)) || len(g.AssetIDs)+len(g.DocumentIDs) == 0 {
		return g, ErrInvalid
	}
	tx, e := s.pool.Begin(ctx)
	if e != nil {
		return g, e
	}
	defer tx.Rollback(ctx)
	// Lock reference rows until the grant is durable. Only exact owner-selected
	// resources are reachable; no profile-wide or write permission is granted.
	for _, id := range g.AssetIDs {
		var found string
		e = tx.QueryRow(ctx, `SELECT id FROM assets WHERE id=$1 AND profile_id=$2 FOR SHARE`, id, owner).Scan(&found)
		if e != nil {
			return g, domainNotFound(e)
		}
	}
	for _, id := range g.DocumentIDs {
		var found string
		e = tx.QueryRow(ctx, `SELECT id FROM documents WHERE id=$1 AND profile_id=$2 FOR SHARE`, id, owner).Scan(&found)
		if e != nil {
			return g, domainNotFound(e)
		}
	}
	var receiver string
	e = tx.QueryRow(ctx, `SELECT id FROM profiles WHERE id=$1`, g.RecipientProfileID).Scan(&receiver)
	if e != nil {
		return g, domainNotFound(e)
	}
	if g.AssetIDs == nil {
		g.AssetIDs = []string{}
	}
	if g.DocumentIDs == nil {
		g.DocumentIDs = []string{}
	}
	g, e = scanGrant(tx.QueryRow(ctx, `INSERT INTO emergency_grants(owner_profile_id,recipient_profile_id,asset_ids,document_ids,expires_at) VALUES($1,$2,$3,$4,$5) RETURNING id,owner_profile_id,recipient_profile_id,asset_ids,document_ids,active,expires_at,created_at`, owner, g.RecipientProfileID, g.AssetIDs, g.DocumentIDs, g.ExpiresAt))
	if e != nil {
		return g, e
	}
	return g, tx.Commit(ctx)
}
func (s *Store) ActivateEmergencyGrant(ctx context.Context, owner, id string, active bool) error {
	tag, e := s.pool.Exec(ctx, `UPDATE emergency_grants SET active=$3 WHERE id=$1 AND owner_profile_id=$2 AND (NOT $3 OR expires_at>now())`, id, owner, active)
	if e == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return e
}
func (s *Store) ReadEmergencyGrant(ctx context.Context, recipient, id string) (EmergencyGrant, error) {
	return scanGrant(s.pool.QueryRow(ctx, grantSelect+` WHERE id=$1 AND recipient_profile_id=$2 AND active AND expires_at>now()`, id, recipient))
}

// Revoked grants remain as access-control audit records. Re-activation requires
// the owner session and is logged separately by the API.
