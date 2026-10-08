package store

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
)

func (s *Store) saveCompanyWithCCA(ctx context.Context, profile string, d CompanyDetails) error {
	tx, e := s.pool.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	var name, currency, kind string
	e = tx.QueryRow(ctx, `SELECT name,currency,kind FROM assets WHERE id=$1 AND profile_id=$2 FOR UPDATE`, d.AssetID, profile).Scan(&name, &currency, &kind)
	if e != nil {
		return domainNotFound(e)
	}
	if kind != "company_share" {
		return ErrInvalid
	}
	var previous *string
	e = tx.QueryRow(ctx, `SELECT cca_asset_id FROM company_details WHERE asset_id=$1 AND profile_id=$2`, d.AssetID, profile).Scan(&previous)
	if e != nil && !errors.Is(e, pgx.ErrNoRows) {
		return e
	}
	linked := d.CCAAssetID
	if linked == nil || *linked == "" {
		linked = previous
	}
	if previous != nil && linked != nil && *previous != *linked {
		var oldValue int64
		e = tx.QueryRow(ctx, `SELECT COALESCE(current_asset_value($1,$2,CURRENT_DATE),0)`, profile, *previous).Scan(&oldValue)
		if e != nil {
			return e
		}
		// Repointing a live receivable would leave its value counted twice.
		// Clear its balance first; historical valuations remain available.
		if oldValue != 0 {
			return ErrInvalid
		}
	}
	if linked != nil {
		if *linked == d.AssetID {
			return ErrInvalid
		}
		var found string
		e = tx.QueryRow(ctx, `SELECT id FROM assets WHERE id=$1 AND profile_id=$2 AND currency=$3 AND NOT archived AND kind<>'company_share' FOR UPDATE`, *linked, profile, currency).Scan(&found)
		if e != nil {
			return domainNotFound(e)
		}
	} else if d.CCA > 0 {
		var id string
		e = tx.QueryRow(ctx, `INSERT INTO assets(profile_id,name,kind,currency,note) VALUES($1,$2,'other',$3,'Créance de compte courant associé') RETURNING id`, profile, "CCA — "+name, currency).Scan(&id)
		if e != nil {
			return e
		}
		linked = &id
	}
	if linked != nil {
		_, e = tx.Exec(ctx, `INSERT INTO valuations(profile_id,asset_id,value_cents,as_of,note) SELECT $1,$2,$3,CURRENT_DATE,'Compte courant associé' WHERE (SELECT value_cents FROM valuations WHERE asset_id=$2 AND as_of<=CURRENT_DATE ORDER BY as_of DESC,created_at DESC,id DESC LIMIT 1) IS DISTINCT FROM $3::bigint`, profile, *linked, d.CCA)
		if e != nil {
			return e
		}
	}
	_, e = tx.Exec(ctx, `INSERT INTO company_details(asset_id,profile_id,siren,ownership_bps,cca_cents,annual_dividends_cents,monthly_salary_cents,cca_asset_id) VALUES($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT(asset_id) DO UPDATE SET siren=$3,ownership_bps=$4,cca_cents=$5,annual_dividends_cents=$6,monthly_salary_cents=$7,cca_asset_id=$8`, d.AssetID, profile, d.SIREN, d.OwnershipBps, d.CCA, d.AnnualDividends, d.MonthlySalary, linked)
	if e != nil {
		return e
	}
	return tx.Commit(ctx)
}
