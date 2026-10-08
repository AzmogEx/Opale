package store

import (
	"context"
	"github.com/opale-app/opale/internal/engine"
	"github.com/opale-app/opale/internal/money"
	"time"
	_ "time/tzdata"
)

type InvestmentFlow struct {
	ID      string      `json:"id"`
	AssetID string      `json:"asset_id"`
	Kind    string      `json:"kind"`
	Amount  money.Cents `json:"amount_cents"`
	Date    string      `json:"occurred_on"`
	Note    string      `json:"note"`
}
type InvestmentDetail struct {
	Asset            Asset                        `json:"asset"`
	Flows            []InvestmentFlow             `json:"flows"`
	Performance      engine.InvestmentPerformance `json:"performance"`
	CoverageComplete bool                         `json:"coverage_complete"`
	FirstDate        *string                      `json:"first_date,omitempty"`
	LastDate         *string                      `json:"last_date,omitempty"`
}

func (s *Store) InvestmentDetail(ctx context.Context, profile, asset string) (InvestmentDetail, error) {
	a, err := s.GetAsset(ctx, profile, asset)
	if err != nil {
		return InvestmentDetail{}, err
	}
	out := InvestmentDetail{Asset: a, Flows: []InvestmentFlow{}}
	rows, err := s.pool.Query(ctx, `SELECT id,asset_id,kind,amount_cents,occurred_on::text,note FROM investment_flows WHERE profile_id=$1 AND asset_id=$2 ORDER BY occurred_on,id`, profile, asset)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		var f InvestmentFlow
		if err = rows.Scan(&f.ID, &f.AssetID, &f.Kind, &f.Amount, &f.Date, &f.Note); err != nil {
			rows.Close()
			return out, err
		}
		out.Flows = append(out.Flows, f)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return out, err
	}
	err = s.pool.QueryRow(ctx, `SELECT COALESCE((SELECT complete FROM investment_coverage WHERE profile_id=$1 AND asset_id=$2),false)`, profile, asset).Scan(&out.CoverageComplete)
	if err != nil {
		return out, err
	}
	vals, err := s.ListAssetValuations(ctx, profile, asset)
	if err != nil {
		return out, err
	}
	if len(vals) == 0 {
		out.Performance.Reason = "Aucune valorisation disponible"
		return out, nil
	}
	// Ignore future snapshots. The first/last observations are closing-day values.
	past := []Valuation{}
	location, err := time.LoadLocation("Europe/Paris")
	if err != nil {
		return out, err
	}
	now := time.Now().In(location)
	// SQL DATE values use UTC midnight as a civil-date encoding, not an
	// instant. Compare against today's Paris calendar date in that encoding.
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	for _, v := range vals {
		if !v.AsOf.After(today) {
			past = append(past, v)
		}
	}
	if len(past) == 0 {
		out.Performance.Reason = "Aucune valorisation passée"
		return out, nil
	}
	first, last := past[len(past)-1], past[0]
	fd, ld := first.AsOf.Format("2006-01-02"), last.AsOf.Format("2006-01-02")
	out.FirstDate = &fd
	out.LastDate = &ld
	out.Asset.LatestValue = &last.Value
	flows := []engine.InvestmentCashFlow{}
	for _, f := range out.Flows {
		if f.Date > fd && f.Date <= ld {
			flows = append(flows, engine.InvestmentCashFlow{Kind: f.Kind, Amount: f.Amount})
		}
	}
	out.Performance, err = engine.ComputeInvestmentPerformance(first.Value, last.Value, flows, out.CoverageComplete)
	return out, err
}
func (s *Store) SaveInvestmentFlow(ctx context.Context, profile string, f InvestmentFlow) (InvestmentFlow, error) {
	var ok bool
	err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM assets WHERE profile_id=$1 AND id=$2 AND kind IN('pea','cto','life_insurance','crypto'))`, profile, f.AssetID).Scan(&ok)
	if err != nil {
		return f, err
	}
	if !ok {
		return f, ErrNotFound
	}
	if f.ID == "" {
		err = s.pool.QueryRow(ctx, `INSERT INTO investment_flows(profile_id,asset_id,kind,amount_cents,occurred_on,note) VALUES($1,$2,$3,$4,$5,$6) RETURNING id`, profile, f.AssetID, f.Kind, f.Amount, f.Date, f.Note).Scan(&f.ID)
	} else {
		tag, e := s.pool.Exec(ctx, `UPDATE investment_flows SET kind=$4,amount_cents=$5,occurred_on=$6,note=$7 WHERE id=$1 AND profile_id=$2 AND asset_id=$3`, f.ID, profile, f.AssetID, f.Kind, f.Amount, f.Date, f.Note)
		err = e
		if err == nil && tag.RowsAffected() == 0 {
			err = ErrNotFound
		}
	}
	return f, err
}
func (s *Store) DeleteInvestmentFlow(ctx context.Context, profile, asset, id string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM investment_flows WHERE id=$1 AND profile_id=$2 AND asset_id=$3`, id, profile, asset)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}
func (s *Store) SetInvestmentCoverage(ctx context.Context, profile, asset string, complete bool) error {
	tag, err := s.pool.Exec(ctx, `INSERT INTO investment_coverage(profile_id,asset_id,complete) SELECT $1,id,$3 FROM assets WHERE profile_id=$1 AND id=$2 ON CONFLICT(profile_id,asset_id) DO UPDATE SET complete=$3`, profile, asset, complete)
	if err == nil && tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return err
}
