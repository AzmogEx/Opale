package store

import (
	"context"
	"fmt"
	"github.com/opale-app/opale/internal/money"
	"sync"
	"testing"
	"time"
)

func TestImportsConcurrencySplitAndProviderUpdates(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	p := newTestProfile(t, s, "import concurrency")
	a, e := s.CreateAsset(ctx, p.ID, "cash", "checking", "EUR", "")
	if e != nil {
		t.Fatal(e)
	}
	n := NewTransaction{AssetID: a.ID, Amount: -10000, OccurredOn: day(t, "2026-03-01"), Label: "M", RawLabel: "M"}
	var wg sync.WaitGroup
	errs := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); _, e := s.ImportTransactions(ctx, p.ID, []NewTransaction{n, n}); errs <- e }()
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		if e != nil {
			t.Fatal(e)
		}
	}
	rows, e := s.ListTransactions(ctx, p.ID, TransactionFilter{})
	if e != nil || len(rows) != 2 {
		t.Fatalf("expected2,got%d %v", len(rows), e)
	}
	if _, e = s.SplitTransaction(ctx, p.ID, rows[0].ID, []SplitPart{{Amount: -6000}, {Amount: -4000}}); e != nil {
		t.Fatal(e)
	}
	r, e := s.ImportTransactions(ctx, p.ID, []NewTransaction{n, n})
	if e != nil || r.Imported != 0 {
		t.Fatal(r, e)
	}
	rows, _ = s.ListTransactions(ctx, p.ID, TransactionFilter{})
	var total money.Cents
	for _, r := range rows {
		total += r.Amount
	}
	if len(rows) != 3 || total != -20000 {
		t.Fatal(rows)
	}
	n.SourceID = "bank-1"
	n.BankStatus = "pending"
	n.RawLabel = "provider"
	n.Amount = -100
	if _, e = s.ImportTransactions(ctx, p.ID, []NewTransaction{n}); e != nil {
		t.Fatal(e)
	}
	n.BankStatus = "booked"
	n.Amount = -200
	if _, e = s.ImportTransactions(ctx, p.ID, []NewTransaction{n}); e != nil {
		t.Fatal(e)
	}
	n.Amount = -300
	if _, e = s.ImportTransactions(ctx, p.ID, []NewTransaction{n}); e != nil {
		t.Fatal(e)
	}
	rows, _ = s.ListTransactions(ctx, p.ID, TransactionFilter{})
	found := 0
	for _, r := range rows {
		if r.SourceID == n.SourceID {
			found++
			if r.Amount != -300 || r.BankStatus != "booked" {
				t.Fatal(r)
			}
		}
	}
	if found != 1 {
		t.Fatal("provider duplicated")
	}
}
func TestBalancesTransfersAndValuationBoundaries(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	p := newTestProfile(t, s, "boundaries")
	a, _ := s.CreateAsset(ctx, p.ID, "A", "checking", "EUR", "")
	b, _ := s.CreateAsset(ctx, p.ID, "B", "savings", "EUR", "")
	s.AddAssetValuation(ctx, p.ID, a.ID, 100000, day(t, "2026-03-02"), "")
	for i, n := range []struct {
		date   string
		amount money.Cents
	}{{"2026-03-01", -500}, {"2026-03-02", -500}, {"2026-03-03", -1000}, {"2099-01-01", 999999}} {
		if _, e := s.CreateTransaction(ctx, p.ID, NewTransaction{AssetID: a.ID, Amount: n.amount, OccurredOn: day(t, n.date), Label: fmt.Sprint(i)}); e != nil {
			t.Fatal(e)
		}
	}
	cash, e := s.CashBalance(ctx, p.ID)
	if e != nil || cash != 99000 {
		t.Fatal(cash, e)
	}
	tr := TransferInput{RequestID: "88888888-1111-4444-8888-888888888888", FromID: a.ID, ToID: b.ID, FromAmount: 20000, ToAmount: 20000, Date: "2026-03-04"}
	if _, e = s.Transfer(ctx, p.ID, tr); e != nil {
		t.Fatal(e)
	}
	if _, e = s.Transfer(ctx, p.ID, tr); e != nil {
		t.Fatal(e)
	}
	cash, e = s.CashBalance(ctx, p.ID)
	if e != nil || cash != 99000 {
		t.Fatal(cash, e)
	}
	tr.RequestID = "88888888-1111-4444-8888-888888888889"
	tr.Fee = 200
	if _, e = s.Transfer(ctx, p.ID, tr); e != nil {
		t.Fatal(e)
	}
	cash, e = s.CashBalance(ctx, p.ID)
	if e != nil || cash != 98800 {
		t.Fatal(cash, e)
	}
	summary, e := s.ComputeMonthSummary(ctx, p.ID, 2026, time.March)
	if e != nil || summary.Income != 0 || summary.Expenses != 2200 {
		t.Fatal(summary, e)
	}
	// Principal is a financing movement; interest and fees are economic expenses.
	for _, n := range []NewTransaction{{AssetID: a.ID, Amount: -3000, OccurredOn: day(t, "2026-03-05"), Label: "principal", FlowKind: "loan_principal"}, {AssetID: a.ID, Amount: -100, OccurredOn: day(t, "2026-03-05"), Label: "interest", FlowKind: "interest"}} {
		if _, e = s.CreateTransaction(ctx, p.ID, n); e != nil {
			t.Fatal(e)
		}
	}
	summary, e = s.ComputeMonthSummary(ctx, p.ID, 2026, time.March)
	if e != nil || summary.Expenses != 2300 {
		t.Fatal(summary, e)
	}
	nw, e := s.ComputeNetWorth(ctx, p.ID)
	cash, _ = s.CashBalance(ctx, p.ID)
	if e != nil || nw.Net != cash {
		t.Fatal(nw, cash, e)
	}
	h, e := s.ComputeNetWorthHistory(ctx, p.ID, 12)
	if e != nil {
		t.Fatal(e)
	}
	if h.Points[len(h.Points)-1].Net != nw.Net {
		t.Fatal("history differs from current")
	}
}
func TestCurrencyMinorUnitsHistoryAndIsolation(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	a := newTestProfile(t, s, "FX A")
	b := newTestProfile(t, s, "FX B")
	for _, p := range []Profile{a, b} {
		asset, e := s.CreateAsset(ctx, p.ID, "yen", "checking", "JPY", "")
		if e != nil {
			t.Fatal(e)
		}
		if _, e = s.AddAssetValuation(ctx, p.ID, asset.ID, 100, day(t, "2026-01-01"), ""); e != nil {
			t.Fatal(e)
		}
	}
	if _, e := s.UpsertProfileFX(ctx, a.ID, "JPY", "2026-01-01", 10000); e != nil {
		t.Fatal(e)
	}
	nw, e := s.ComputeNetWorth(ctx, a.ID)
	if e != nil || nw.Net != 100 {
		t.Fatal(nw, e)
	}
	if _, e = s.ComputeNetWorth(ctx, b.ID); e == nil {
		t.Fatal("missing rate silently valued")
	}
	if _, e = s.UpsertProfileFX(ctx, a.ID, "JPY", "2026-07-01", 20000); e != nil {
		t.Fatal(e)
	}
	var historical int64
	if e = s.pool.QueryRow(ctx, `SELECT amount_eur(100,'JPY','2026-03-01',$1::uuid)`, a.ID).Scan(&historical); e != nil || historical != 100 {
		t.Fatal(historical, e)
	}
	if e = s.pool.QueryRow(ctx, `SELECT amount_eur(100,'JPY','2025-03-01',$1::uuid)`, a.ID).Scan(&historical); e == nil {
		t.Fatal("historical missing rate accepted")
	}
	s.UpsertProfileFX(ctx, a.ID, "KWD", "2026-01-01", 3000000)
	if e = s.pool.QueryRow(ctx, `SELECT amount_eur(1234,'KWD','2026-03-01',$1::uuid)`, a.ID).Scan(&historical); e != nil || historical != 370 {
		t.Fatal(historical, e)
	}
}

func TestLinkedPrincipalPreservesNetWorthAndRefusesOverpayment(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	p := newTestProfile(t, s, "loan principal")
	a, _ := s.CreateAsset(ctx, p.ID, "A", "checking", "EUR", "")
	l, _ := s.CreateLiability(ctx, p.ID, "L", "consumer_loan", "EUR", "")
	s.AddAssetValuation(ctx, p.ID, a.ID, 100000, day(t, "2026-01-01"), "")
	s.AddLiabilityValuation(ctx, p.ID, l.ID, 50000, day(t, "2026-01-01"), "")
	n := NewTransaction{AssetID: a.ID, LinkedLiabilityID: &l.ID, FlowKind: "loan_principal", Amount: -10000, OccurredOn: day(t, "2026-03-01"), Label: "Repayment"}
	if _, e := s.CreateTransaction(ctx, p.ID, n); e != nil {
		t.Fatal(e)
	}
	nw, e := s.ComputeNetWorth(ctx, p.ID)
	if e != nil || nw.Net != 50000 {
		t.Fatal(nw, e)
	}
	n.Amount = -50000
	if _, e = s.CreateTransaction(ctx, p.ID, n); e == nil {
		t.Fatal("overpayment accepted")
	}
	cash, e := s.CashBalance(ctx, p.ID)
	if e != nil || cash != 90000 {
		t.Fatal("failed operation mutated cash", cash, e)
	}
}

// Deletion must also work with restrictive owner relations introduced by delivery.
func TestProfileCleanupWithLinkedDomains(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	for _, reset := range []bool{false, true} {
		t.Run(fmt.Sprint(reset), func(t *testing.T) {
			p, e := s.CreateProfile(ctx, "cleanup links", "hash", "N1")
			if e != nil {
				t.Fatal(e)
			}
			t.Cleanup(func() { s.DeleteProfile(ctx, p.ID) })
			a, e := s.CreateAsset(ctx, p.ID, "Account", "checking", "EUR", "")
			if e != nil {
				t.Fatal(e)
			}
			c, e := s.CreateAsset(ctx, p.ID, "Company", "company_share", "EUR", "")
			if e != nil {
				t.Fatal(e)
			}
			if e = s.UpsertCompanyDetails(ctx, p.ID, CompanyDetails{AssetID: c.ID, OwnershipBps: 10000, CCA: 1000}); e != nil {
				t.Fatal(e)
			}
			link, e := s.CreateBankLink(ctx, p.ID, BankLink{AssetID: a.ID, RequisitionID: "synthetic-" + p.ID, InstitutionID: "TEST", InstitutionName: "Test"})
			if e != nil {
				t.Fatal(e)
			}
			if e = s.DiscoverBankAccounts(ctx, p.ID, link.ID, []string{"test-" + p.ID}); e != nil {
				t.Fatal(e)
			}
			if _, e = s.pool.Exec(ctx, `INSERT INTO bank_account_bindings(profile_id,provider_account_id,asset_id)VALUES($1,$2,$3)`, p.ID, "test-"+p.ID, a.ID); e != nil {
				t.Fatal(e)
			}
			if reset {
				e = s.ResetProfileData(ctx, p.ID)
			} else {
				e = s.DeleteProfile(ctx, p.ID)
			}
			if e != nil {
				t.Fatal(e)
			}
			for _, table := range []string{"assets", "liabilities", "bank_links", "bank_account_bindings", "company_details"} {
				var count int
				if e = s.pool.QueryRow(ctx, `SELECT count(*) FROM `+table+` WHERE profile_id=$1`, p.ID).Scan(&count); e != nil || count != 0 {
					t.Fatal(table, count, e)
				}
			}
		})
	}
}

func TestLegacySplitImportRefusesAmbiguousParentAtomically(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	p := newTestProfile(t, s, "legacy split")
	a, e := s.CreateAsset(ctx, p.ID, "Account", "checking", "EUR", "")
	if e != nil {
		t.Fatal(e)
	}
	for _, amount := range []money.Cents{-6000, -4000} {
		if _, e = s.CreateTransaction(ctx, p.ID, NewTransaction{AssetID: a.ID, Amount: amount, OccurredOn: day(t, "2026-03-01"), Label: "Legacy", RawLabel: "Legacy"}); e != nil {
			t.Fatal(e)
		}
	}
	_, e = s.ImportTransactions(ctx, p.ID, []NewTransaction{{AssetID: a.ID, Amount: 123, OccurredOn: day(t, "2026-03-02"), Label: "new", RawLabel: "new"}, {AssetID: a.ID, Amount: -10000, OccurredOn: day(t, "2026-03-01"), Label: "Legacy", RawLabel: "Legacy"}})
	if e != ErrImportAmbiguous {
		t.Fatalf("wanted ambiguity, got %v", e)
	}
	rows, e := s.ListTransactions(ctx, p.ID, TransactionFilter{})
	if e != nil || len(rows) != 2 {
		t.Fatal("partial import", len(rows), e)
	}
}
