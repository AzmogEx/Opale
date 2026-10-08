package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/opale-app/opale/internal/bank"
	"github.com/opale-app/opale/internal/store"
)

func TestDeliveryEmergencyLifecycleAndBeneficiaryIsolation(t *testing.T) {
	s, db := testAPI(t)
	ctx := context.Background()
	owner, token := profile(t, s, "Domain owner")
	recipient, receiverToken := profile(t, s, "Domain trusted")
	outsider, outsideToken := profile(t, s, "Domain stranger")
	asset := account(t, s, token)
	otherAsset := account(t, s, token)
	foreignAsset := account(t, s, outsideToken)
	encrypted, e := s.vault.Encrypt([]byte("Selected encrypted contract"))
	if e != nil {
		t.Fatal(e)
	}
	doc, e := s.store.CreateDocument(ctx, owner, store.Document{Name: "Selected contract", Kind: "contract", Mime: "text/plain", SizeBytes: 27, SHA256: strings.Repeat("a", 64)}, encrypted)
	if e != nil {
		t.Fatal(e)
	}
	unrelated, e := s.store.CreateDocument(ctx, owner, store.Document{Name: "Private unrelated", Kind: "other", Mime: "text/plain", SizeBytes: 7, SHA256: strings.Repeat("b", 64)}, encrypted)
	if e != nil {
		t.Fatal(e)
	}
	create := func(assetIDs []string, docIDs []string) *httptest.ResponseRecorder {
		return request(t, s, token, "POST", "/v1/emergency-grants", map[string]any{"recipient_profile_id": recipient, "asset_ids": assetIDs, "document_ids": docIDs, "active": true, "expires_at": time.Now().Add(time.Hour)})
	}
	if w := create([]string{foreignAsset}, nil); w.Code != 404 {
		t.Fatalf("foreign reference %d %s", w.Code, w.Body)
	}
	w := create([]string{asset}, []string{doc.ID})
	if w.Code != 201 {
		t.Fatalf("grant %d %s", w.Code, w.Body)
	}
	grant := decode(t, w)
	id := grant["id"].(string)
	if grant["active"] != false {
		t.Fatal("creation silently activated access")
	}
	if w = request(t, s, receiverToken, "GET", "/v1/emergency/"+id, nil); w.Code != 404 {
		t.Fatalf("inactive %d", w.Code)
	}
	if w = request(t, s, receiverToken, "PATCH", "/v1/emergency-grants/"+id, map[string]any{"active": true}); w.Code != 404 {
		t.Fatalf("recipient activated %d", w.Code)
	}
	if w = request(t, s, token, "PATCH", "/v1/emergency-grants/"+id, map[string]any{"active": true}); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	if w = request(t, s, outsideToken, "GET", "/v1/emergency/"+id, nil); w.Code != 404 {
		t.Fatalf("stranger reads %d", w.Code)
	}
	w = request(t, s, receiverToken, "GET", "/v1/emergency/"+id, nil)
	if w.Code != 200 {
		t.Fatalf("read %d %s", w.Code, w.Body)
	}
	d := decode(t, w)
	assets := d["assets"].([]any)
	docs := d["documents"].([]any)
	if len(assets) != 1 || len(docs) != 1 || assets[0].(map[string]any)["id"] != asset || assets[0].(map[string]any)["theoretical_cents"] != float64(100000) || strings.Contains(w.Body.String(), otherAsset) || strings.Contains(w.Body.String(), "Private unrelated") {
		t.Fatalf("grant scope/data %s", w.Body)
	}
	if w = request(t, s, receiverToken, "GET", "/v1/emergency/"+id+"/documents/"+unrelated.ID, nil); w.Code != 404 {
		t.Fatalf("unselected doc %d", w.Code)
	}
	w = request(t, s, receiverToken, "GET", "/v1/emergency/"+id+"/documents/"+doc.ID, nil)
	if w.Code != 200 || w.Body.String() != "Selected encrypted contract" || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("download %d %s", w.Code, w.Body)
	}
	if w = request(t, s, receiverToken, "PATCH", "/v1/assets/"+asset, map[string]any{"name": "intrusion", "archived": false}); w.Code != 404 {
		t.Fatalf("recipient writes owner %d", w.Code)
	}
	if w = request(t, s, token, "DELETE", "/v1/emergency-grants/"+id, nil); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	if w = request(t, s, receiverToken, "GET", "/v1/emergency/"+id+"/documents/"+doc.ID, nil); w.Code != 404 {
		t.Fatalf("revoked download %d", w.Code)
	}
	if _, e = db.Exec(ctx, `UPDATE emergency_grants SET active=true,expires_at=now()-interval '1 second' WHERE id=$1`, id); e != nil {
		t.Fatal(e)
	}
	if w = request(t, s, receiverToken, "GET", "/v1/emergency/"+id, nil); w.Code != 404 {
		t.Fatalf("expired %d", w.Code)
	}

	c1, e := s.store.CreateContact(ctx, owner, store.Contact{Name: "Beneficiary1", Role: "trusted"})
	if e != nil {
		t.Fatal(e)
	}
	c2, e := s.store.CreateContact(ctx, owner, store.Contact{Name: "Beneficiary2", Role: "trusted"})
	if e != nil {
		t.Fatal(e)
	}
	c3, e := s.store.CreateContact(ctx, outsider, store.Contact{Name: "Foreign", Role: "trusted"})
	if e != nil {
		t.Fatal(e)
	}
	save := func(contact string, share int) *httptest.ResponseRecorder {
		return request(t, s, token, "PUT", "/v1/beneficiaries", map[string]any{"contact_id": contact, "document_id": doc.ID, "share_bps": share})
	}
	if w = save(c1.ID, 7000); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	if w = save(c2.ID, 4000); w.Code != 422 {
		t.Fatalf("shares >100%% %d %s", w.Code, w.Body)
	}
	if w = save(c3.ID, 1000); w.Code != 404 {
		t.Fatalf("foreign beneficiary %d %s", w.Code, w.Body)
	}
	if w = save(c2.ID, 3000); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	b, e := s.store.ListBeneficiaries(ctx, owner)
	if e != nil || len(b) != 2 {
		t.Fatalf("beneficiaries %v %v", b, e)
	}
	if w = request(t, s, outsideToken, "PATCH", "/v1/contacts/"+c1.ID, map[string]any{"name": "Changed", "role": "trusted"}); w.Code != 404 {
		t.Fatalf("contact edit %d", w.Code)
	}
	if w = request(t, s, token, "PATCH", "/v1/documents/"+doc.ID, map[string]any{"name": "Changed", "kind": "contract", "asset_id": foreignAsset}); w.Code != 404 {
		t.Fatalf("document cross link %d", w.Code)
	}
}

func TestDeliveryBankMockSyncReconciliationIdempotencyAndBackoff(t *testing.T) {
	s, db := testAPI(t)
	ctx := context.Background()
	id, token := profile(t, s, "Bank synthetic")
	asset := account(t, s, token)
	var phase, calls atomic.Int32
	today := parisToday().Format("2006-01-02")
	remote := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v2/token/new/":
			fmt.Fprint(w, `{"access":"test-token","access_expires":86400}`)
		case "/api/v2/requisitions/test-requisition/":
			fmt.Fprint(w, `{"id":"test-requisition","status":"LN","accounts":["remote-account"]}`)
		case "/api/v2/accounts/remote-account/details/":
			fmt.Fprint(w, `{"account":{"currency":"EUR","name":"Synthetic account","iban":"FR7612345678901234567890123","status":"enabled"}}`)
		case "/api/v2/accounts/remote-account/transactions/":
			calls.Add(1)
			if phase.Load() == 2 {
				w.Header().Set("Retry-After", "30000")
				w.WriteHeader(429)
				fmt.Fprint(w, `{"private":"must not persist"}`)
				return
			}
			first := map[string]any{"internalTransactionId": "booked-1", "bookingDate": today, "transactionAmount": map[string]string{"amount": "-10.00", "currency": "EUR"}, "remittanceInformationUnstructured": "Booked purchase"}
			second := map[string]any{"internalTransactionId": "booked-2", "transactionAmount": map[string]string{"amount": "-3.00", "currency": "EUR"}, "remittanceInformationUnstructured": "Pending purchase"}
			booked := []any{first}
			pending := []any{second}
			if phase.Load() == 1 {
				second["bookingDate"] = today
				booked = append(booked, second)
				pending = []any{}
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"transactions": map[string]any{"booked": booked, "pending": pending}})
		case "/api/v2/accounts/remote-account/balances/":
			amount := "100.00"
			if phase.Load() == 1 {
				amount = "97.00"
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"balances": []any{map[string]any{"balanceAmount": map[string]string{"amount": amount, "currency": "EUR"}, "balanceType": "interimBooked", "referenceDate": today, "creditLimitIncluded": false}}})
		default:
			t.Errorf("unexpected provider call %s", r.URL.Path)
			w.WriteHeader(404)
		}
	}))
	defer remote.Close()
	s.bank = bank.New(remote.URL, "fake-id", "fake-secret")
	link, e := s.store.CreateBankLink(ctx, id, store.BankLink{AssetID: asset, RequisitionID: "test-requisition", InstitutionID: "sandbox", InstitutionName: "Synthetic bank"})
	if e != nil {
		t.Fatal(e)
	}
	res := s.syncBankLink(ctx, id, link)
	if res.Status != "needs_mapping" || calls.Load() != 0 {
		t.Fatalf("must discover before importing %+v calls%d", res, calls.Load())
	}
	accounts, e := s.store.ListBankAccounts(ctx, id)
	if e != nil || len(accounts) != 1 || accounts[0].AssetID != nil {
		t.Fatalf("account map %v %v", accounts, e)
	}
	_, foreignToken := profile(t, s, "Bank stranger")
	if w := request(t, s, foreignToken, "PUT", "/v1/bank/accounts/"+accounts[0].ID, map[string]string{"asset_id": asset}); w.Code != 404 {
		t.Fatalf("foreign map %d %s", w.Code, w.Body)
	}
	if w := request(t, s, token, "PUT", "/v1/bank/accounts/"+accounts[0].ID, map[string]string{"asset_id": asset}); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	res = s.syncBankLink(ctx, id, link)
	if res.Status != "synced" || res.Imported != 1 {
		t.Fatalf("sync %+v", res)
	}
	cash, e := s.store.CashBalance(ctx, id)
	if e != nil || cash != 10000 {
		t.Fatalf("booked balance %d %v", cash, e)
	}
	w := request(t, s, token, "GET", "/v1/bank/accounts", nil)
	d := decode(t, w)
	pending := d["pending"].([]any)
	if len(pending) != 1 || pending[0].(map[string]any)["occurred_on"] != nil {
		t.Fatalf("pending undated observation %s", w.Body)
	}
	if res = s.syncBankLink(ctx, id, link); res.Status != "cooldown" || calls.Load() != 1 {
		t.Fatalf("persisted schedule ignored %+v calls%d", res, calls.Load())
	}
	phase.Store(1)
	if _, e = db.Exec(ctx, `UPDATE bank_links SET next_sync_at=now() WHERE id=$1`, link.ID); e != nil {
		t.Fatal(e)
	}
	res = s.syncBankLink(ctx, id, link)
	if res.Status != "synced" || res.Imported != 1 || res.Duplicates != 1 {
		t.Fatalf("pending becomes booked once %+v", res)
	}
	cash, e = s.store.CashBalance(ctx, id)
	if e != nil || cash != 9700 {
		t.Fatalf("reconciled double cash %d %v", cash, e)
	}
	if p, e := s.store.ListBankPending(ctx, id); e != nil || len(p) != 0 {
		t.Fatalf("pending retained %v %v", p, e)
	}
	phase.Store(2)
	if _, e = db.Exec(ctx, `UPDATE bank_links SET next_sync_at=now() WHERE id=$1`, link.ID); e != nil {
		t.Fatal(e)
	}
	res = s.syncBankLink(ctx, id, link)
	if res.Status != "error" || strings.Contains(res.Error, "private") {
		t.Fatalf("rate limit status %+v", res)
	}
	saved, e := s.store.BankLink(ctx, id, link.ID)
	if e != nil || saved.NextSyncAt.Before(time.Now().Add(8*time.Hour)) || saved.LastSyncedAt == nil || saved.Attempts != 1 {
		t.Fatalf("backoff state %+v %v", saved, e)
	}
	cash, e = s.store.CashBalance(ctx, id)
	if e != nil || cash != 9700 {
		t.Fatalf("failure changed cash %d %v", cash, e)
	}
	// The binding survives disconnect: re-consent cannot silently move the
	// same provider history into a second local account and duplicate wealth.
	replacement, e := s.store.CreateAsset(ctx, id, "Other local account", "checking", "EUR", "")
	if e != nil {
		t.Fatal(e)
	}
	if w = request(t, s, token, "PUT", "/v1/bank/accounts/"+accounts[0].ID, map[string]string{"asset_id": replacement.ID}); w.Code != 422 {
		t.Fatalf("live remapping %d %s", w.Code, w.Body)
	}
	if w = request(t, s, token, "DELETE", "/v1/bank/links/"+link.ID, nil); w.Code != 204 {
		t.Fatalf("disconnect %d %s", w.Code, w.Body)
	}
	link, e = s.store.CreateBankLink(ctx, id, store.BankLink{AssetID: asset, RequisitionID: "test-requisition", InstitutionID: "sandbox", InstitutionName: "Synthetic bank"})
	if e != nil {
		t.Fatal(e)
	}
	phase.Store(1)
	if res = s.syncBankLink(ctx, id, link); res.Status != "needs_mapping" {
		t.Fatalf("reconnect discovery %+v", res)
	}
	accounts, e = s.store.ListBankAccounts(ctx, id)
	if e != nil || len(accounts) != 1 {
		t.Fatalf("reconnected accounts %v %v", accounts, e)
	}
	if w = request(t, s, token, "PUT", "/v1/bank/accounts/"+accounts[0].ID, map[string]string{"asset_id": replacement.ID}); w.Code != 422 {
		t.Fatalf("reconnected remapping %d %s", w.Code, w.Body)
	}
	if w = request(t, s, token, "PUT", "/v1/bank/accounts/"+accounts[0].ID, map[string]string{"asset_id": asset}); w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	if res = s.syncBankLink(ctx, id, link); res.Status != "synced" || res.Imported != 0 || res.Duplicates != 2 {
		t.Fatalf("reconnect double import %+v", res)
	}
	cash, e = s.store.CashBalance(ctx, id)
	if e != nil || cash != 9700 {
		t.Fatalf("reconnect cash %d %v", cash, e)
	}
}

func TestDeliveryDeleteAccountReturnsActionableConflict(t *testing.T) {
	s, _ := testAPI(t)
	ctx := context.Background()
	id, token := profile(t, s, "API history guard")
	asset := account(t, s, token)
	if _, e := s.store.CreateTransaction(ctx, id, store.NewTransaction{AssetID: asset, Amount: -100, OccurredOn: parisToday(), Label: "History"}); e != nil {
		t.Fatal(e)
	}
	w := request(t, s, token, "DELETE", "/v1/assets/"+asset, nil)
	if w.Code != 409 || decode(t, w)["error"].(map[string]any)["code"] != "history_required" {
		t.Fatalf("guard status %d %s", w.Code, w.Body)
	}
	rows, e := s.store.ListTransactions(ctx, id, store.TransactionFilter{AssetID: asset})
	if e != nil || len(rows) != 1 {
		t.Fatalf("guard changed history %v %v", rows, e)
	}
}
