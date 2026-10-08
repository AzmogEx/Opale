package store

import (
	"context"
	"testing"
	"time"

	"github.com/opale-app/opale/internal/migrations"
)

func TestCalendarDatesMonthEnds(t *testing.T) {
	d := func(s string) time.Time { v, _ := time.Parse("2006-01-02", s); return v }
	r := CalendarRule{Date: "2024-01-31", Active: true, Frequency: "monthly"}
	out := CalendarDates(r, d("2024-01-01"), d("2024-04-30"))
	want := []string{"2024-01-31", "2024-02-29", "2024-03-31", "2024-04-30"}
	if len(out) != len(want) {
		t.Fatal(out)
	}
	for i, v := range out {
		if v.Format("2006-01-02") != want[i] {
			t.Fatal(out)
		}
	}
	r.Frequency = "yearly"
	r.Date = "2024-02-29"
	out = CalendarDates(r, d("2025-01-01"), d("2025-12-31"))
	if len(out) != 1 || out[0].Format("2006-01-02") != "2025-02-28" {
		t.Fatal(out)
	}
	r.Active = false
	if len(CalendarDates(r, d("2025-01-01"), d("2025-12-31"))) != 0 {
		t.Fatal("stopped rule emitted")
	}
}

func TestCalendarDatesQuarterlyPreservesOriginalDay(t *testing.T) {
	r := CalendarRule{Date: "2024-01-31", Active: true, Frequency: "quarterly"}
	out := CalendarDates(r, day(t, "2024-01-01"), day(t, "2024-10-31"))
	want := []string{"2024-01-31", "2024-04-30", "2024-07-31", "2024-10-31"}
	if len(out) != len(want) {
		t.Fatal(out)
	}
	for i, d := range out {
		if d.Format("2006-01-02") != want[i] {
			t.Fatal(out)
		}
	}
}

func TestDeliveryQuarterlyCalendarPersistsAndDownRefusesDataLoss(t *testing.T) {
	s := newTestStore(t)
	ctx := context.Background()
	p := newTestProfile(t, s, "Quarterly calendar")
	a, err := s.CreateAsset(ctx, p.ID, "cash", "checking", "EUR", "")
	if err != nil {
		t.Fatal(err)
	}
	rule, err := s.SaveCalendarRule(ctx, p.ID, CalendarRule{AssetID: a.ID, Label: "Quarterly insurance", Amount: -30000, Date: "2024-01-31", Frequency: "quarterly", Active: true})
	if err != nil {
		t.Fatal(err)
	}
	if err = s.SetCalendarOccurrence(ctx, p.ID, rule.ID, "2024-04-30", "excluded", nil, nil); err != nil {
		t.Fatal(err)
	}
	occ, err := s.CalendarOccurrences(ctx, p.ID, day(t, "2024-01-01"), day(t, "2024-07-31"))
	if err != nil || len(occ) != 3 {
		t.Fatal(occ, err)
	}
	for i, date := range []string{"2024-01-31", "2024-04-30", "2024-07-31"} {
		if occ[i].Date != date || occ[i].Amount != -30000 || occ[i].EUR != -30000 {
			t.Fatal(occ)
		}
	}
	if occ[1].Status != "excluded" {
		t.Fatal(occ)
	}
	// Execute the actual down migration in a transaction that is always rolled
	// back. It must reject incompatible data before altering the constraint.
	down, err := migrations.FS.ReadFile("0021_quarterly_calendar.down.sql")
	if err != nil {
		t.Fatal(err)
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	if _, err = tx.Exec(ctx, string(down)); err == nil {
		t.Fatal("down migration accepted quarterly data")
	}
	if err = tx.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	rules, err := s.ListCalendarRules(ctx, p.ID)
	if err != nil || len(rules) != 1 || rules[0].Frequency != "quarterly" {
		t.Fatal(rules, err)
	}
}
