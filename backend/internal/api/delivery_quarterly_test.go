package api

import "testing"

func TestDeliveryQuarterlyCalendarAPI(t *testing.T) {
	s, _ := testAPI(t)
	_, token := profile(t, s, "Quarterly API")
	a := account(t, s, token)
	w := request(t, s, token, "POST", "/v1/calendar", map[string]any{"asset_id": a, "label": "Quarterly premium", "amount_cents": -30000, "date": "2024-01-31", "frequency": "quarterly", "active": true})
	if w.Code != 200 {
		t.Fatalf("create %d %s", w.Code, w.Body)
	}
	w = request(t, s, token, "GET", "/v1/calendar?from=2024-01-01&until=2024-07-31", nil)
	if w.Code != 200 {
		t.Fatalf("read %d %s", w.Code, w.Body)
	}
	d := decode(t, w)
	occ := d["occurrences"].([]any)
	if len(occ) != 3 {
		t.Fatal(d)
	}
	for i, date := range []string{"2024-01-31", "2024-04-30", "2024-07-31"} {
		if occ[i].(map[string]any)["date"] != date {
			t.Fatal(occ)
		}
	}
}
