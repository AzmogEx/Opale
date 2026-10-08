package api

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestMetricsAndLogsNeverRecordUserPaths(t *testing.T) {
	var logs bytes.Buffer
	s := &Server{log: slog.New(slog.NewJSONHandler(&logs, nil))}
	r := chi.NewRouter()
	r.Use(s.logRequests, s.recoverer)
	r.Get("/assets/{id}", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) })
	for _, path := range []string{"/assets/IBAN_SECRET?note=PRIVATE", "/Alice_BNP"} {
		r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", path, nil))
	}
	w := httptest.NewRecorder()
	s.handleMetrics(w, httptest.NewRequest("GET", "/metrics", nil))
	both := logs.String() + w.Body.String()
	for _, secret := range []string{"IBAN_SECRET", "PRIVATE", "Alice_BNP"} {
		if strings.Contains(both, secret) {
			t.Fatal("private path in telemetry", secret)
		}
	}
	if !strings.Contains(w.Body.String(), `route="/assets/{id}"`) || !strings.Contains(w.Body.String(), `route="unmatched"`) {
		t.Fatal(w.Body.String())
	}
}
func TestMetricsConcurrentCollection(t *testing.T) {
	s := &Server{log: slog.New(slog.NewTextHandler(io.Discard, nil))}
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.metrics.record("GET", "/healthz", 200, 1)
			s.handleMetrics(httptest.NewRecorder(), httptest.NewRequest("GET", "/metrics", nil))
		}()
	}
	wg.Wait()
	if s.metrics.values[requestMetricKey{"GET", "/healthz", 200}].count != 20 {
		t.Fatal("lost measurements")
	}
}
