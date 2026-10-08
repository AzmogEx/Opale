package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

func TestReadinessStatus(t *testing.T) {
	for _, status := range []int{200, 204, 302, 503} {
		t.Run(strconv.Itoa(status), func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet || r.URL.Path != "/readyz" {
					t.Errorf("unexpected probe: %s %s", r.Method, r.URL.Path)
				}
				if status == http.StatusFound {
					w.Header().Set("Location", "/login")
				}
				w.WriteHeader(status)
			}))
			defer srv.Close()
			err := check(context.Background(), srv.URL+"/readyz")
			if (err == nil) != (status == http.StatusOK) {
				t.Fatalf("status %d: %v", status, err)
			}
		})
	}
}

func TestUnavailableAPI(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	srv.Close()
	if err := check(context.Background(), srv.URL+"/readyz"); err == nil {
		t.Fatal("an unreachable API must fail readiness")
	}
}

func TestCancelledProbe(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := check(ctx, "http://127.0.0.1:8080/readyz"); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation, got %v", err)
	}
}
