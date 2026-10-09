package api

import (
	"context"
	"fmt"
	"github.com/opale-app/opale/internal/store"
	"testing"
)

func TestFinancialToolsHTTPIsolationConflictAndManagedCalendar(t *testing.T) {
	s, db := testAPI(t)
	owner, token := profile(t, s, "HTTP financial")
	_, foreignToken := profile(t, s, "HTTP foreign")
	asset := account(t, s, token)
	var id string
	if e := db.QueryRow(context.Background(), `SELECT gen_random_uuid()::text`).Scan(&id); e != nil {
		t.Fatal(e)
	}
	c := store.FinancialContract{ID: id, Name: "Synthetic contract", Category: "subscription", Amount: 1299, Currency: "EUR", Frequency: "monthly", NextDueDate: parisToday().Format(dayLayout), AssetID: asset, Active: true, ReminderDays: 7, TrialEnd: parisToday().Format(dayLayout)}
	path := "/v1/contracts/" + id
	if r := request(t, s, "", "PUT", path, c); r.Code != 401 {
		t.Fatalf("auth %d", r.Code)
	}
	if r := request(t, s, foreignToken, "PUT", path, c); r.Code != 404 {
		t.Fatalf("foreign account %d %s", r.Code, r.Body)
	}
	r := request(t, s, token, "PUT", path, c)
	if r.Code != 200 {
		t.Fatalf("create %d %s", r.Code, r.Body)
	}
	saved := decode(t, r)
	rule := saved["calendar_rule_id"].(string)
	if r = request(t, s, token, "PUT", path, c); r.Code != 409 {
		t.Fatalf("retry must not duplicate %d", r.Code)
	}
	if r = request(t, s, foreignToken, "GET", path+"/prices", nil); r.Code != 404 {
		t.Fatalf("foreign price history %d", r.Code)
	}
	if r = request(t, s, foreignToken, "GET", "/v1/contracts", nil); r.Code != 200 || len(decode(t, r)["contracts"].([]any)) != 0 {
		t.Fatal("foreign list")
	}
	if r = request(t, s, token, "GET", "/v1/alerts", nil); r.Code != 200 {
		t.Fatalf("alerts %s", r.Body)
	} else {
		found := false
		for _, a := range decode(t, r)["alerts"].([]any) {
			if a.(map[string]any)["kind"] == "contract_trial" {
				found = true
			}
		}
		if !found {
			t.Fatal("contract reminder missing in inbox")
		}
	}
	if r = request(t, s, token, "DELETE", "/v1/calendar/"+rule, nil); r.Code != 409 {
		t.Fatalf("managed calendar %d %s", r.Code, r.Body)
	}
	c.Revision = 1
	c.Amount = 1499
	if r = request(t, s, token, "PUT", path, c); r.Code != 200 {
		t.Fatalf("edit %s", r.Body)
	}
	if r = request(t, s, token, "DELETE", path+"?revision=1", nil); r.Code != 409 {
		t.Fatalf("stale delete %d", r.Code)
	}
	var count int
	if e := db.QueryRow(context.Background(), `SELECT count(*) FROM transactions WHERE profile_id=$1`, owner).Scan(&count); e != nil || count != 0 {
		t.Fatal("invented actual")
	}
	if r = request(t, s, token, "DELETE", path+"?revision=2", nil); r.Code != 204 {
		t.Fatalf("delete %d %s", r.Code, r.Body)
	}
	v := store.VariableIncome{ID: id, Name: "Prime", Kind: "bonus", Currency: "EUR", Low: 0, Usual: 10000, High: 20000, Frequency: "once", NextDate: parisToday().Format(dayLayout), Forecast: "prudent", AssetID: asset, Active: true}
	if r = request(t, s, token, "PUT", "/v1/incomes/variable/"+id, v); r.Code != 200 {
		t.Fatalf("income %s", r.Body)
	}
	if r = request(t, s, foreignToken, "DELETE", fmt.Sprintf("/v1/incomes/variable/%s?revision=1", id), nil); r.Code != 404 {
		t.Fatalf("foreign income delete %d", r.Code)
	}
}
