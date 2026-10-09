package api

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/opale-app/opale/internal/money"
	"github.com/opale-app/opale/internal/store"
)

func TestJourneyResumeIsolationRevisionExportResetAndNoActuals(t *testing.T) {
	s, db := testAPI(t)
	owner, token := profile(t, s, "Journey owner")
	_, foreign := profile(t, s, "Journey foreign")
	if r := request(t, s, "", "GET", "/v1/journey", nil); r.Code != 401 {
		t.Fatal(r.Code)
	}
	r := request(t, s, token, "GET", "/v1/journey", nil)
	initial := decode(t, r)
	if r.Code != 200 || initial["revision"] != float64(0) || len(initial["reviewed"].([]any)) != 0 {
		t.Fatal(initial)
	}
	zero := money.Cents(0)
	j := store.FinancialJourney{Step: 5, Reviewed: []string{"accounts", "income"}, Skipped: []string{"wealth"}, DailyBudget: &zero, BudgetCurrency: "JPY"}
	r = request(t, s, token, "PUT", "/v1/journey", j)
	if r.Code != 200 {
		t.Fatal(r.Code, r.Body)
	}
	saved := decode(t, r)
	if saved["revision"] != float64(1) || saved["daily_budget_cents"] != float64(0) {
		t.Fatal(saved)
	}
	if r = request(t, s, token, "PUT", "/v1/journey", j); r.Code != 409 {
		t.Fatal("lost update", r.Code, r.Body)
	}
	resumed := decode(t, request(t, s, token, "GET", "/v1/journey", nil))
	if resumed["budget_currency"] != "JPY" || len(resumed["skipped"].([]any)) != 1 {
		t.Fatal(resumed)
	}
	other := decode(t, request(t, s, foreign, "GET", "/v1/journey", nil))
	if other["revision"] != float64(0) || other["daily_budget_cents"] != nil {
		t.Fatal("cross profile read", other)
	}
	var count int
	if e := db.QueryRow(context.Background(), `SELECT count(*) FROM transactions WHERE profile_id=$1`, owner).Scan(&count); e != nil || count != 0 {
		t.Fatal("journey created actual money", e, count)
	}
	export := request(t, s, token, "GET", "/v1/export", nil)
	if export.Code != 200 {
		t.Fatal(export.Code, export.Body)
	}
	zr, e := zip.NewReader(bytes.NewReader(export.Body.Bytes()), int64(export.Body.Len()))
	if e != nil {
		t.Fatal(e)
	}
	found := false
	for _, f := range zr.File {
		if f.Name != "export.json" {
			continue
		}
		rc, e := f.Open()
		if e != nil {
			t.Fatal(e)
		}
		b, _ := io.ReadAll(rc)
		rc.Close()
		var data map[string]json.RawMessage
		if e = json.Unmarshal(b, &data); e != nil {
			t.Fatal(e)
		}
		if !strings.Contains(string(data["profile_journey"]), "JPY") {
			t.Fatal("journey budget omitted", string(data["profile_journey"]))
		}
		found = true
	}
	if !found {
		t.Fatal("progress omitted from export")
	}
	if e := s.store.ResetProfileData(context.Background(), owner); e != nil {
		t.Fatal(e)
	}
	if d := decode(t, request(t, s, token, "GET", "/v1/journey", nil)); d["revision"] != float64(0) {
		t.Fatal("reset retained progress", d)
	}
}

func TestJourneyRejectsInvalidStepsAndOverlappingProgress(t *testing.T) {
	s, _ := testAPI(t)
	_, token := profile(t, s, "Journey validation")
	for _, body := range []string{
		`{"step":7,"budget_currency":"EUR"}`, `{"step":-1,"budget_currency":"EUR"}`,
		`{"budget_currency":"ZZZ"}`, `{"budget_currency":"EUR","daily_budget_cents":-1}`,
		`{"budget_currency":"EUR","daily_budget_cents":1000000000000}`,
		`{"budget_currency":"EUR","reviewed":["unknown"]}`,
		`{"budget_currency":"EUR","reviewed":["income","income"]}`,
		`{"budget_currency":"EUR","reviewed":["income"],"skipped":["income"]}`,
	} {
		var data any
		json.Unmarshal([]byte(body), &data)
		r := request(t, s, token, "PUT", "/v1/journey", data)
		if r.Code != 422 {
			t.Fatal(body, r.Code, r.Body)
		}
	}
}

func TestJourneyFirstSaveSerialisesConcurrentWriters(t *testing.T) {
	s, _ := testAPI(t)
	_, token := profile(t, s, "Journey concurrent")
	statuses := make(chan int, 2)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r := request(t, s, token, "PUT", "/v1/journey", store.FinancialJourney{BudgetCurrency: "EUR", Reviewed: []string{"accounts"}})
			statuses <- r.Code
		}()
	}
	wg.Wait()
	close(statuses)
	counts := map[int]int{}
	for code := range statuses {
		counts[code]++
	}
	if counts[200] != 1 || counts[409] != 1 {
		t.Fatal(counts)
	}
}

func TestJourneyCarriesCompletedLegacyBudgetForwardWithoutMarkingProgress(t *testing.T) {
	s, _ := testAPI(t)
	owner, token := profile(t, s, "Legacy journey budget")
	ctx := context.Background()
	old, e := s.store.GetOnboarding(ctx, owner)
	if e != nil {
		t.Fatal(e)
	}
	draft := old.Draft
	draft.Income.Enabled = false
	draft.Account.Enabled = false
	draft.VariableBudget = "735,42"
	old, e = s.store.SaveOnboarding(ctx, owner, 0, 5, draft)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.store.CompleteOnboarding(ctx, owner, old.Revision); e != nil {
		t.Fatal(e)
	}
	data := decode(t, request(t, s, token, "GET", "/v1/journey", nil))
	if data["daily_budget_cents"] != float64(73542) || data["revision"] != float64(0) || len(data["reviewed"].([]any)) != 0 {
		t.Fatal(data)
	}
	// Clearing the new budget is intentional and must not re-import an old value.
	r := request(t, s, token, "PUT", "/v1/journey", store.FinancialJourney{BudgetCurrency: "EUR"})
	if r.Code != 200 {
		t.Fatal(r.Code, r.Body)
	}
	if data = decode(t, request(t, s, token, "GET", "/v1/journey", nil)); data["daily_budget_cents"] != nil {
		t.Fatal(data)
	}
}
