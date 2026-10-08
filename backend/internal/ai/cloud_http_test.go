package ai

import (
	"bytes"
	"context"
	"github.com/anthropics/anthropic-sdk-go/option"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOutboundHTTPIsAllowlistedAndOptIn(t *testing.T) {
	calls := 0
	var payload string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		b, _ := io.ReadAll(r.Body)
		payload = string(b)
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"id":"msg_test","type":"message","role":"assistant","model":"test","content":[{"type":"text","text":"{\"facts\":[\"summary\"]}"}],"stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":1}}`)
	}))
	defer server.Close()
	var logs bytes.Buffer
	router := NewRouter(nil, NewAnthropic("synthetic", option.WithBaseURL(server.URL), option.WithMaxRetries(0)), slog.New(slog.NewTextHandler(&logs, nil)))
	req := Request{System: FactSelectionSystem, Task: "assistant_ask", Prompt: "Alice Martin BNP IBAN FR761234 Testament", CloudFacts: &CloudFacts{Intent: "overview", CashThousands: 12}, AllowCloud: true}
	if _, e := router.Explain(context.Background(), req); e != nil {
		t.Fatal(e)
	}
	if calls != 1 {
		t.Fatal(calls)
	}
	for _, secret := range []string{"Alice", "BNP", "FR76", "Testament"} {
		if strings.Contains(payload, secret) || strings.Contains(logs.String(), secret) {
			t.Fatal("PII crossed HTTP/log boundary", secret)
		}
	}
	if !strings.Contains(payload, "cash_thousands_eur") {
		t.Fatal("no structured context emitted")
	}
	req.AllowCloud = false
	router.Explain(context.Background(), req)
	req.AllowCloud = true
	req.CloudFacts = nil
	router.Explain(context.Background(), req)
	if calls != 1 {
		t.Fatal("disabled/unsafe request caused HTTP call")
	}
}
func TestProviderCannotInventDisplayedNumbers(t *testing.T) {
	for _, raw := range []string{`{"facts":["cash"],"amount":999}`, `{"facts":["unknown"]}`, `{"facts":["cash"]} {"facts":["cash"]}`, `You own 9000000 euros`} {
		if _, e := SelectFacts(raw, map[string]string{"cash": "Cash : 1.23 €"}); e == nil {
			t.Fatal("unsafe response accepted", raw)
		}
	}
	text, e := SelectFacts(`{"facts":["cash","cash"]}`, map[string]string{"cash": "Cash : 1.23 €"})
	if e != nil || text != "Cash : 1.23 €" {
		t.Fatal(text, e)
	}
}
