package api

import (
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
)

// Operational metrics are bounded by registered route patterns, never by an ID,
// query, profile or imported label. Scrape only over the private API network.
type requestMetricKey struct {
	method, route string
	status        int
}
type requestMetric struct {
	count   uint64
	seconds float64
	buckets [6]uint64
}
type requestMetrics struct {
	mu     sync.Mutex
	values map[requestMetricKey]requestMetric
}

var latencyBounds = [6]float64{.01, .05, .1, .3, 1, 5}

func normalizedRoute(r *http.Request) string {
	if c := chi.RouteContext(r.Context()); c != nil {
		if pattern := c.RoutePattern(); pattern != "" {
			return pattern
		}
	}
	return "unmatched"
}
func metricMethod(method string) string {
	switch method {
	case "GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "HEAD":
		return method
	}
	return "OTHER"
}
func (m *requestMetrics) record(method, route string, status int, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.values == nil {
		m.values = make(map[requestMetricKey]requestMetric)
	}
	key := requestMetricKey{metricMethod(method), route, status}
	v := m.values[key]
	v.count++
	v.seconds += duration.Seconds()
	for i, b := range latencyBounds {
		if duration.Seconds() <= b {
			v.buckets[i]++
		}
	}
	m.values[key] = v
}
func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	s.metrics.mu.Lock()
	snapshot := make(map[requestMetricKey]requestMetric, len(s.metrics.values))
	for k, v := range s.metrics.values {
		snapshot[k] = v
	}
	s.metrics.mu.Unlock()
	keys := make([]requestMetricKey, 0, len(snapshot))
	for k := range snapshot {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		a, b := keys[i], keys[j]
		if a.route != b.route {
			return a.route < b.route
		}
		if a.method != b.method {
			return a.method < b.method
		}
		return a.status < b.status
	})
	fmt.Fprintln(w, "# HELP opale_http_request_duration_seconds HTTP latency by normalized route.\n# TYPE opale_http_request_duration_seconds histogram")
	for _, k := range keys {
		v := snapshot[k]
		labels := fmt.Sprintf("method=%q,route=%q,status=%q", k.method, k.route, fmt.Sprint(k.status))
		for i, b := range latencyBounds {
			fmt.Fprintf(w, "opale_http_request_duration_seconds_bucket{%s,le=%q} %d\n", labels, fmt.Sprint(b), v.buckets[i])
		}
		fmt.Fprintf(w, "opale_http_request_duration_seconds_bucket{%s,le=\"+Inf\"} %d\n", labels, v.count)
		fmt.Fprintf(w, "opale_http_request_duration_seconds_count{%s} %d\n", labels, v.count)
		fmt.Fprintf(w, "opale_http_request_duration_seconds_sum{%s} %g\n", labels, v.seconds)
	}
}
