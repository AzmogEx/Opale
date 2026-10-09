package store

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/opale-app/opale/internal/money"
)

func TestDeliveryGoalAllocationsSerialize(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	p := newTestProfile(t, s, "Domain goal allocations")
	a, e := s.CreateAsset(ctx, p.ID, "cash", "checking", "EUR", "")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.CreateTransaction(ctx, p.ID, NewTransaction{AssetID: a.ID, Amount: 300000, OccurredOn: time.Now().UTC(), Label: "Savings capacity", RawLabel: "Savings capacity"}); e != nil {
		t.Fatal(e)
	}
	// Observed three-month capacity is 100000. Two simultaneous allocations of
	// 80000 must never both commit against the same available capacity.
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for _, name := range []string{"A", "B"} {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			_, e := s.SaveGoal(ctx, p.ID, "", name, "target", 1000000, nil, &a.ID, 80000)
			errs <- e
		}(name)
	}
	wg.Wait()
	close(errs)
	ok, rejected := 0, 0
	for e := range errs {
		if e == nil {
			ok++
		} else if errors.Is(e, ErrInvalid) {
			rejected++
		} else {
			t.Fatal(e)
		}
	}
	if ok != 1 || rejected != 1 {
		t.Fatalf("committed %d, rejected %d", ok, rejected)
	}
	goals, e := s.ListGoals(ctx, p.ID)
	if e != nil || len(goals) != 1 {
		t.Fatalf("goals %v %v", goals, e)
	}
	g, e := s.SaveGoal(ctx, p.ID, goals[0].ID, "Renamed", "target", 2000000, nil, &a.ID, 90000)
	if e != nil || g.MonthlySavings != 90000 {
		t.Fatalf("own allocation edit %v %v", g, e)
	}
	other := newTestProfile(t, s, "Domain goal outsider")
	if _, e = s.SaveGoal(ctx, other.ID, "", "cross", "target", 100000, nil, &a.ID, 0); !errors.Is(e, ErrNotFound) {
		t.Fatalf("cross reference: %v", e)
	}
}

func TestDeliveryCalendarPersistenceAndRealizations(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	p := newTestProfile(t, s, "Domain calendar")
	a, e := s.CreateAsset(ctx, p.ID, "cash", "checking", "EUR", "")
	if e != nil {
		t.Fatal(e)
	}
	rule, e := s.SaveCalendarRule(ctx, p.ID, CalendarRule{AssetID: a.ID, Label: "Rent", Amount: -10000, Date: "2026-01-31", Frequency: "monthly", Active: true, MerchantKey: "rent"})
	if e != nil {
		t.Fatal(e)
	}
	tx, e := s.CreateTransaction(ctx, p.ID, NewTransaction{AssetID: a.ID, Amount: -10000, OccurredOn: day(t, "2026-02-28"), Label: "Rent", RawLabel: "Rent"})
	if e != nil {
		t.Fatal(e)
	}
	if e = s.SetCalendarOccurrence(ctx, p.ID, rule.ID, "2026-02-28", "realized", nil, &tx.ID); e != nil {
		t.Fatal(e)
	}
	if e = s.SetCalendarOccurrence(ctx, p.ID, rule.ID, "2026-03-31", "excluded", nil, nil); e != nil {
		t.Fatal(e)
	}
	if e = s.SetCalendarOccurrence(ctx, p.ID, rule.ID, "2026-04-01", "excluded", nil, nil); !errors.Is(e, ErrNotFound) {
		t.Fatalf("non-occurrence: %v", e)
	}
	if e = s.SetCalendarOccurrence(ctx, p.ID, rule.ID, "2026-04-30", "realized", nil, &tx.ID); e == nil {
		t.Fatal("one booked operation must not realize two occurrences")
	}
	occ, e := s.CalendarOccurrences(ctx, p.ID, day(t, "2026-02-01"), day(t, "2026-04-30"))
	if e != nil {
		t.Fatal(e)
	}
	if len(occ) != 3 || occ[0].Status != "realized" || occ[1].Status != "excluded" || occ[2].Status != "planned" || occ[2].EUR != -10000 {
		t.Fatalf("persisted calendar %+v", occ)
	}
	excluded, e := s.CalendarMerchantKeys(ctx, p.ID)
	if e != nil || len(excluded) != 1 || excluded[0] != "rent" {
		t.Fatalf("manual recurring dedup %v %v", excluded, e)
	}
	if e = s.DeleteCalendarRule(ctx, p.ID, rule.ID); e != nil {
		t.Fatal(e)
	}
	var count int
	if e = s.pool.QueryRow(ctx, `SELECT count(*) FROM calendar_occurrences WHERE profile_id=$1`, p.ID).Scan(&count); e != nil || count != 0 {
		t.Fatalf("orphan occurrences %d %v", count, e)
	}
}

func TestDeliveryInvestmentFlowsAndArchiveHistory(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	p := newTestProfile(t, s, "Domain performance")
	a, e := s.CreateAsset(ctx, p.ID, "Portfolio", "pea", "EUR", "")
	if e != nil {
		t.Fatal(e)
	}
	for _, v := range []struct {
		date   string
		amount money.Cents
	}{{"2026-01-01", 100000}, {"2026-03-01", 160000}, {"2099-01-01", 990000}} {
		if _, e = s.AddAssetValuation(ctx, p.ID, a.ID, v.amount, day(t, v.date), ""); e != nil {
			t.Fatal(e)
		}
	}
	f, e := s.SaveInvestmentFlow(ctx, p.ID, InvestmentFlow{AssetID: a.ID, Kind: "contribution", Amount: 50000, Date: "2026-02-01"})
	if e != nil {
		t.Fatal(e)
	}
	d, e := s.InvestmentDetail(ctx, p.ID, a.ID)
	if e != nil || d.Performance.Known {
		t.Fatalf("unknown until confirmed: %+v %v", d, e)
	}
	if e = s.SetInvestmentCoverage(ctx, p.ID, a.ID, true); e != nil {
		t.Fatal(e)
	}
	d, e = s.InvestmentDetail(ctx, p.ID, a.ID)
	if e != nil || !d.Performance.Known || d.Performance.Gain != 10000 || *d.LastDate != "2026-03-01" {
		t.Fatalf("contribution-adjusted %+v %v", d, e)
	}
	f.Amount = 60000
	if _, e = s.SaveInvestmentFlow(ctx, p.ID, f); e != nil {
		t.Fatal(e)
	}
	if _, e = s.UpdateAsset(ctx, p.ID, a.ID, a.Name, "", true); e != nil {
		t.Fatal(e)
	}
	d, e = s.InvestmentDetail(ctx, p.ID, a.ID)
	if e != nil || !d.Asset.Archived || d.Performance.Gain != 0 || len(d.Flows) != 1 {
		t.Fatalf("archive lost history %+v %v", d, e)
	}
}

func TestDeliveryCompanyCCAIsCountedOnceAndKeepsEdits(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	p := newTestProfile(t, s, "Domain company")
	a, e := s.CreateAsset(ctx, p.ID, "Company", "company_share", "EUR", "")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.AddAssetValuation(ctx, p.ID, a.ID, 100000, day(t, "2026-01-01"), "Value of my 50 percent share"); e != nil {
		t.Fatal(e)
	}
	d := CompanyDetails{AssetID: a.ID, OwnershipBps: 5000, CCA: 10000}
	for _, value := range []money.Cents{10000, 20000, 10000} {
		d.CCA = value
		if e = s.UpsertCompanyDetails(ctx, p.ID, d); e != nil {
			t.Fatal(e)
		}
	}
	companies, e := s.ListCompanies(ctx, p.ID)
	if e != nil || len(companies) != 1 || companies[0].Details.CCA != 10000 || companies[0].Details.CCAAssetID == nil {
		t.Fatalf("company %+v %v", companies, e)
	}
	nw, e := s.ComputeNetWorth(ctx, p.ID)
	if e != nil || nw.Net != 110000 {
		t.Fatalf("share + one receivable %+v %v", nw, e)
	}
	var count int
	if e = s.pool.QueryRow(ctx, `SELECT count(*) FROM assets WHERE profile_id=$1`, p.ID).Scan(&count); e != nil || count != 2 {
		t.Fatalf("repeated CCA created %d assets: %v", count, e)
	}
	replacement, e := s.CreateAsset(ctx, p.ID, "Existing receivable", "other", "EUR", "")
	if e != nil {
		t.Fatal(e)
	}
	d.CCAAssetID = &replacement.ID
	if e = s.UpsertCompanyDetails(ctx, p.ID, d); !errors.Is(e, ErrInvalid) {
		t.Fatalf("live CCA relink must reject %v", e)
	}
	d.CCAAssetID = nil
	d.CCA = 0
	if e = s.UpsertCompanyDetails(ctx, p.ID, d); e != nil {
		t.Fatal(e)
	}
	d.CCAAssetID = &replacement.ID
	d.CCA = 10000
	if e = s.UpsertCompanyDetails(ctx, p.ID, d); e != nil {
		t.Fatal(e)
	}
	nw, e = s.ComputeNetWorth(ctx, p.ID)
	if e != nil || nw.Net != 110000 {
		t.Fatalf("relink double counted %+v %v", nw, e)
	}
}
