package bank

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestProviderExactAmountsAndStates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v2/token/new/":
			fmt.Fprint(w, `{"access":"test-token","access_expires":3600}`)
		case "/api/v2/accounts/account/transactions/":
			if r.Header.Get("Authorization") != "Bearer test-token" {
				t.Error("missing token")
			}
			fmt.Fprint(w, `{"transactions":{"booked":[{"internalTransactionId":"stable","transactionAmount":{"amount":"1.234","currency":"KWD"},"bookingDate":"2026-10-08","creditorName":"Synthetic"}],"pending":[{"transactionAmount":{"amount":"2.000","currency":"KWD"},"creditorName":"Pending without date"}]}}`)
		default:
			w.WriteHeader(404)
		}
	}))
	defer server.Close()
	g := New(server.URL, "id", "key")
	rows, e := g.Transactions(context.Background(), "account")
	if e != nil {
		t.Fatal(e)
	}
	if len(rows) != 2 || rows[0].SourceID != "gocardless:account:stable" || rows[0].Status != "booked" || rows[1].Status != "pending" {
		t.Fatal(rows)
	}
	for _, tt := range []struct {
		raw  string
		exp  int
		want int64
	}{{"1.234", 3, 1234}, {"100", 0, 100}, {"-10.25", 2, -1025}} {
		v, e := ParseMinor(tt.raw, tt.exp)
		if e != nil || int64(v) != tt.want {
			t.Fatal(v, e)
		}
	}
	if _, e := ParseMinor("1.01", 0); e == nil {
		t.Fatal("precision loss")
	}
	if _, e := ParseMinor("1/2", 2); e == nil {
		t.Fatal("non-decimal")
	}
}
func TestProviderRefreshesRejectedTokenOnce(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/token/new/" {
			calls++
			fmt.Fprintf(w, `{"access":"token-%d","access_expires":3600}`, calls)
			return
		}
		if r.Header.Get("Authorization") == "Bearer token-1" {
			w.WriteHeader(401)
			return
		}
		fmt.Fprint(w, `{"id":"r","status":"LN","accounts":[]}`)
	}))
	defer server.Close()
	g := New(server.URL, "id", "key")
	if _, e := g.GetRequisition(context.Background(), "r"); e != nil || calls != 2 {
		t.Fatal(e, calls)
	}
}
func TestProviderRateLimitHasSafeErrorAndDelay(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/token/new/" {
			fmt.Fprint(w, `{"access":"token","access_expires":3600}`)
			return
		}
		w.Header().Set("Retry-After", "3600")
		w.WriteHeader(429)
		fmt.Fprint(w, `{"detail":"IBAN synthetic-private"}`)
	}))
	defer server.Close()
	g := New(server.URL, "id", "key")
	_, e := g.Balances(context.Background(), "a")
	var pe *APIError
	if !errors.As(e, &pe) || pe.RetryAfter != time.Hour || strings.Contains(e.Error(), "private") {
		t.Fatal(e)
	}
}
