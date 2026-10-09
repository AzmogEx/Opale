package api

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/opale-app/opale/internal/store"
)

func TestOnboardingOverdraftCanBeRevaluedOnlyForChecking(t *testing.T) {
	s, _ := testAPI(t)
	_, token := profile(t, s, "Overdraft lifecycle")
	_, otherToken := profile(t, s, "Overdraft other")
	for _, kind := range []string{"checking", "savings", "other", "consumer_loan"} {
		collection := "assets"
		if kind == "consumer_loan" {
			collection = "liabilities"
		}
		w := request(t, s, token, "POST", "/v1/"+collection, map[string]any{"name": kind, "kind": kind, "currency": "EUR"})
		if w.Code != 201 {
			t.Fatalf("create %s %d %s", kind, w.Code, w.Body)
		}
		id := decode(t, w)["id"].(string)
		path := fmt.Sprintf("/v1/%s/%s/valuations", collection, id)
		w = request(t, s, token, "POST", path, map[string]any{"value_cents": 10000})
		if w.Code != 201 {
			t.Fatalf("positive valuation %s %d %s", kind, w.Code, w.Body)
		}
		valuation := decode(t, w)["id"].(string)
		w = request(t, s, token, "POST", path, map[string]any{"value_cents": -1205})
		want := 422
		if kind == "checking" {
			want = 201
		}
		if w.Code != want {
			t.Fatalf("negative POST %s %d %s", kind, w.Code, w.Body)
		}
		w = request(t, s, token, "PATCH", "/v1/valuations/"+valuation, map[string]any{"value_cents": -1525})
		if kind == "checking" {
			want = 200
		}
		if w.Code != want {
			t.Fatalf("negative PATCH %s %d %s", kind, w.Code, w.Body)
		}
		w = request(t, s, otherToken, "POST", path, map[string]any{"value_cents": -1205})
		if w.Code != 404 {
			t.Fatalf("other-profile POST %s %d %s", kind, w.Code, w.Body)
		}
		w = request(t, s, otherToken, "PATCH", "/v1/valuations/"+valuation, map[string]any{"value_cents": -1525})
		if w.Code != 404 {
			t.Fatalf("other-profile PATCH %s %d %s", kind, w.Code, w.Body)
		}
	}
}

func TestOnboardingAPIContractPrivacyAndFinalization(t *testing.T) {
	s, db := testAPI(t)
	owner, token := profile(t, s, "Setup API owner")
	_, otherToken := profile(t, s, "Setup API other")
	if w := request(t, s, "", "GET", "/v1/onboarding", nil); w.Code != 401 {
		t.Fatalf("unauthenticated %d", w.Code)
	}
	w := request(t, s, token, "GET", "/v1/onboarding", nil)
	var initial store.OnboardingState
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &initial) != nil || initial.Status != "not_started" || !initial.ShouldPrompt || !initial.Draft.Income.Enabled || !initial.Draft.Account.Enabled {
		t.Fatalf("GET %d %s", w.Code, w.Body)
	}
	draft := initial.Draft
	draft.Income.Amount = "2 345,67 €"
	draft.Account.Balance = "-12,05"
	draft.Expenses = []store.OnboardingExpense{{ID: "ca555723-2eeb-4475-94e4-05338f54ddbe", Label: "Charges déclarées", Amount: "560,15", Date: parisToday().Format(dayLayout), Frequency: "monthly"}}
	draft.Subscriptions = []store.OnboardingExpense{{ID: "35d7b05b-8dbd-4302-aac0-8d66d571309f", Label: "Musique déclarée", Amount: "11,99", Date: parisToday().Format(dayLayout), Frequency: "monthly"}}
	put := map[string]any{"expected_revision": 0, "step": 5, "draft": draft}
	w = request(t, s, token, "PUT", "/v1/onboarding", put)
	var saved store.OnboardingState
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &saved) != nil || saved.Revision != 1 {
		t.Fatalf("PUT %d %s", w.Code, w.Body)
	}
	// Drafts are confined to their authenticated profile and never public profiles.
	w = request(t, s, otherToken, "GET", "/v1/onboarding", nil)
	var other store.OnboardingState
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &other) != nil || other.Draft.Income.Amount != "" || other.Revision != 0 {
		t.Fatalf("draft isolation %s", w.Body)
	}
	w = request(t, s, "", "GET", "/v1/profiles", nil)
	if _, leaked := decode(t, w)["draft"]; leaked {
		t.Fatal("public draft")
	}
	w = request(t, s, token, "POST", "/v1/onboarding/complete", map[string]any{"expected_revision": 0})
	if w.Code != 409 || decode(t, w)["error"].(map[string]any)["code"] != "onboarding_conflict" {
		t.Fatalf("conflict %d %s", w.Code, w.Body)
	}
	w = request(t, s, token, "POST", "/v1/onboarding/complete", map[string]any{"expected_revision": saved.Revision})
	var done store.OnboardingState
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &done) != nil || done.Status != "completed" || done.Result == nil || done.Result.AssetID == "" {
		t.Fatalf("complete %d %s", w.Code, w.Body)
	}
	w = request(t, s, token, "POST", "/v1/onboarding/complete", map[string]any{"expected_revision": saved.Revision})
	var retry store.OnboardingState
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &retry) != nil || retry.Result.AssetID != done.Result.AssetID {
		t.Fatalf("retry %d %s", w.Code, w.Body)
	}
	var count int
	if err := db.QueryRow(context.Background(), "SELECT count(*) FROM transactions WHERE profile_id=$1", owner).Scan(&count); err != nil || count != 0 {
		t.Fatalf("artificial actuals %d %v", count, err)
	}
	w = request(t, s, token, "GET", "/v1/cashflow?days=30", nil)
	if w.Code != 200 {
		t.Fatalf("cashflow %d %s", w.Code, w.Body)
	}
	projection := decode(t, w)
	if projection["start_cash_cents"] != float64(-1205) || len(projection["upcoming"].([]any)) < 3 {
		t.Fatalf("planned cash %+v", projection)
	}
	// The intended variable budget was never silently turned into realized spending.
	w = request(t, s, token, "GET", "/v1/transactions/summary", nil)
	if w.Code != 200 {
		t.Fatalf("summary %d %s", w.Code, w.Body)
	}
}

func TestOnboardingAPIRejectsMalformedAndForeignReferences(t *testing.T) {
	s, _ := testAPI(t)
	_, token := profile(t, s, "Setup validation")
	_, otherToken := profile(t, s, "Setup other account")
	foreign := account(t, s, otherToken)
	for _, tc := range []struct {
		method, path string
		body         any
		want         int
	}{
		{"PUT", "/v1/onboarding", map[string]any{"step": 1, "draft": map[string]any{}}, 400},
		{"PUT", "/v1/onboarding", map[string]any{"expected_revision": 0, "step": 6, "draft": map[string]any{}}, 422},
		{"PUT", "/v1/onboarding", map[string]any{"expected_revision": 0, "step": 0, "draft": map[string]any{"unknown": true}}, 400},
		{"POST", "/v1/onboarding/complete", map[string]any{}, 400},
		{"POST", "/v1/onboarding/skip", map[string]any{"expected_revision": -1}, 422},
	} {
		w := request(t, s, token, tc.method, tc.path, tc.body)
		if w.Code != tc.want {
			t.Fatalf("%s %s = %d %s", tc.method, tc.path, w.Code, w.Body)
		}
	}
	draft := store.OnboardingDraft{Account: store.OnboardingAccount{Enabled: true, ExistingAssetID: foreign}}
	w := request(t, s, token, "PUT", "/v1/onboarding", map[string]any{"expected_revision": 0, "step": 5, "draft": draft})
	if w.Code != 200 {
		t.Fatalf("save %d %s", w.Code, w.Body)
	}
	w = request(t, s, token, "POST", "/v1/onboarding/complete", map[string]any{"expected_revision": 1})
	if w.Code != 404 {
		t.Fatalf("foreign account %d %s", w.Code, w.Body)
	}
	w = request(t, s, token, "POST", "/v1/onboarding/skip", map[string]any{"expected_revision": 1})
	var state store.OnboardingState
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &state) != nil || state.Status != "skipped" || state.ShouldPrompt || state.Draft.Account.ExistingAssetID != foreign {
		t.Fatalf("skip %d %s", w.Code, w.Body)
	}
}
