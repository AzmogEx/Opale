package store

import (
	"context"
	"errors"
	"testing"
)

func TestDeliveryAccountDeletionPreservesTransferAndPrincipal(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	p := newTestProfile(t, s, "Guard linked history")
	a, e := s.CreateAsset(ctx, p.ID, "Source", "checking", "EUR", "")
	if e != nil {
		t.Fatal(e)
	}
	b, e := s.CreateAsset(ctx, p.ID, "Destination", "savings", "EUR", "")
	if e != nil {
		t.Fatal(e)
	}
	debt, e := s.CreateLiability(ctx, p.ID, "Loan", "consumer_loan", "EUR", "")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.AddLiabilityValuation(ctx, p.ID, debt.ID, 100000, day(t, "2026-01-01"), ""); e != nil {
		t.Fatal(e)
	}
	if _, e = s.Transfer(ctx, p.ID, TransferInput{RequestID: "19000000-0000-4000-8000-000000000001", FromID: a.ID, ToID: b.ID, FromAmount: 2000, ToAmount: 2000, Fee: 100, Date: "2026-02-01"}); e != nil {
		t.Fatal(e)
	}
	if _, e = s.CreateTransaction(ctx, p.ID, NewTransaction{AssetID: a.ID, Amount: -5000, FlowKind: "loan_principal", LinkedLiabilityID: &debt.ID, OccurredOn: day(t, "2026-02-02"), Label: "Principal"}); e != nil {
		t.Fatal(e)
	}
	for _, id := range []string{a.ID, b.ID} {
		if e = s.DeleteAsset(ctx, p.ID, id); !errors.Is(e, ErrHistoryRequired) {
			t.Fatalf("delete account %v", e)
		}
	}
	rows, e := s.ListTransactions(ctx, p.ID, TransactionFilter{})
	if e != nil || len(rows) != 4 {
		t.Fatalf("transfer/principal changed %d %v", len(rows), e)
	}
	var value int64
	if e = s.pool.QueryRow(ctx, `SELECT current_liability_value($1,$2)`, p.ID, debt.ID).Scan(&value); e != nil || value != 95000 {
		t.Fatalf("debt changed %d %v", value, e)
	}
	if e = s.DeleteLiability(ctx, p.ID, debt.ID); e == nil {
		t.Fatal("linked debt must remain protected")
	}
	// The deliberate whole-profile reset still removes the complete graph.
	if e = s.ResetProfileData(ctx, p.ID); e != nil {
		t.Fatal(e)
	}
	rows, e = s.ListTransactions(ctx, p.ID, TransactionFilter{})
	if e != nil || len(rows) != 0 {
		t.Fatalf("reset %d %v", len(rows), e)
	}
}

func TestDeliveryLiabilityValuationCorrectionRollsBack(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	p := newTestProfile(t, s, "Guard debt valuations")
	a, e := s.CreateAsset(ctx, p.ID, "Source", "checking", "EUR", "")
	if e != nil {
		t.Fatal(e)
	}
	debt, e := s.CreateLiability(ctx, p.ID, "Loan", "consumer_loan", "EUR", "")
	if e != nil {
		t.Fatal(e)
	}
	valuation, e := s.AddLiabilityValuation(ctx, p.ID, debt.ID, 100000, day(t, "2026-01-01"), "Original")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.CreateTransaction(ctx, p.ID, NewTransaction{AssetID: a.ID, Amount: -30000, FlowKind: "loan_principal", LinkedLiabilityID: &debt.ID, OccurredOn: day(t, "2026-03-01"), Label: "Principal"}); e != nil {
		t.Fatal(e)
	}
	low := int64(20000)
	if _, e = s.UpdateValuation(ctx, p.ID, valuation.ID, &low, nil, nil); e == nil {
		t.Fatal("correction below principal accepted")
	}
	if e = s.DeleteValuation(ctx, p.ID, valuation.ID); e == nil {
		t.Fatal("deletion left principal without baseline")
	}
	if _, e = s.AddLiabilityValuation(ctx, p.ID, debt.ID, 20000, day(t, "2026-02-01"), "Invalid baseline"); e == nil {
		t.Fatal("new insufficient baseline accepted")
	}
	vals, e := s.ListLiabilityValuations(ctx, p.ID, debt.ID)
	if e != nil || len(vals) != 1 || vals[0].Value != 100000 || vals[0].Note != "Original" {
		t.Fatalf("failed mutations persisted %+v %v", vals, e)
	}
	var value int64
	if e = s.pool.QueryRow(ctx, `SELECT current_liability_value($1,$2)`, p.ID, debt.ID).Scan(&value); e != nil || value != 70000 {
		t.Fatalf("debt unreadable %d %v", value, e)
	}
	good := int64(120000)
	if _, e = s.UpdateValuation(ctx, p.ID, valuation.ID, &good, nil, nil); e != nil {
		t.Fatal(e)
	}
	if e = s.pool.QueryRow(ctx, `SELECT current_liability_value($1,$2)`, p.ID, debt.ID).Scan(&value); e != nil || value != 90000 {
		t.Fatalf("valid correction %d %v", value, e)
	}
	// Full deletion remains an explicit operation, not routed through the
	// single-account history guard.
	if e = s.DeleteProfile(ctx, p.ID); e != nil {
		t.Fatal(e)
	}
}

func TestDeliveryLaterSnapshotCannotHideNegativeDebtHistory(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	p := newTestProfile(t, s, "Guard debt whole history")
	a, e := s.CreateAsset(ctx, p.ID, "Source", "checking", "EUR", "")
	if e != nil {
		t.Fatal(e)
	}
	debt, e := s.CreateLiability(ctx, p.ID, "Loan", "consumer_loan", "EUR", "")
	if e != nil {
		t.Fatal(e)
	}
	initial, e := s.AddLiabilityValuation(ctx, p.ID, debt.ID, 10000, day(t, "2026-01-01"), "Initial100EUR")
	if e != nil {
		t.Fatal(e)
	}
	principal := NewTransaction{AssetID: a.ID, Amount: -8000, FlowKind: "loan_principal", LinkedLiabilityID: &debt.ID, OccurredOn: day(t, "2026-03-01"), Label: "March80EUR"}
	if _, e = s.CreateTransaction(ctx, p.ID, principal); e != nil {
		t.Fatal(e)
	}
	if _, e = s.AddLiabilityValuation(ctx, p.ID, debt.ID, 10000, day(t, "2026-06-01"), "Later100EUR"); e != nil {
		t.Fatal(e)
	}
	low := int64(5000)
	if _, e = s.UpdateValuation(ctx, p.ID, initial.ID, &low, nil, nil); e == nil {
		t.Fatal("later snapshot hid invalid January correction")
	}
	if e = s.DeleteValuation(ctx, p.ID, initial.ID); e == nil {
		t.Fatal("later snapshot hid deleted baseline")
	}
	if _, e = s.AddLiabilityValuation(ctx, p.ID, debt.ID, 5000, day(t, "2026-02-01"), "Invalid historical50EUR"); e == nil {
		t.Fatal("later snapshot hid invalid intermediate valuation")
	}
	principal.Amount = -3000
	principal.OccurredOn = day(t, "2026-02-01")
	if _, e = s.CreateTransaction(ctx, p.ID, principal); e == nil {
		t.Fatal("later snapshot hid historical overpayment")
	}
	for _, tt := range []struct {
		date string
		want int64
	}{{"2026-03-31", 2000}, {"2026-06-30", 10000}} {
		var value int64
		if e = s.pool.QueryRow(ctx, `SELECT current_liability_value($1,$2,$3)`, p.ID, debt.ID, tt.date).Scan(&value); e != nil || value != tt.want {
			t.Fatalf("history %s got%d want%d err%v", tt.date, value, tt.want, e)
		}
	}
	// A future booked repayment must also remain valid when an earlier payment
	// is added now. Checking only today or the new payment date misses this.
	principal.Amount = -8000
	principal.OccurredOn = day(t, "2099-03-01")
	if _, e = s.CreateTransaction(ctx, p.ID, principal); e != nil {
		t.Fatal(e)
	}
	principal.Amount = -3000
	principal.OccurredOn = day(t, "2026-07-01")
	if _, e = s.CreateTransaction(ctx, p.ID, principal); e == nil {
		t.Fatal("accepted an earlier payment that invalidates the future book")
	}
	var value int64
	if e = s.pool.QueryRow(ctx, `SELECT current_liability_value($1,$2,'2099-03-31')`, p.ID, debt.ID).Scan(&value); e != nil || value != 2000 {
		t.Fatalf("future history changed%d %v", value, e)
	}
	rows, e := s.ListTransactions(ctx, p.ID, TransactionFilter{})
	if e != nil || len(rows) != 2 {
		t.Fatalf("refused repayments persisted %d %v", len(rows), e)
	}
}
