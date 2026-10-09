package ai

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/anthropics/anthropic-sdk-go/option"
)

func TestChosenProviderDoesNotOverrideConsentOrFallThrough(t *testing.T) {
	h := &fakeProvider{name: "private", tier: TierHomelab, available: true}
	c := &fakeProvider{name: "cloud", tier: TierCloud, available: true}
	r := NewRouter(h, c, testLogger())
	req := Request{Provider: "cloud", Prompt: "private question", CloudFacts: &CloudFacts{Intent: "overview"}}
	if _, e := r.Explain(context.Background(), req); e == nil || h.gotPrompt != "" || c.gotPrompt != "" {
		t.Fatal("provider granted consent or sent private text")
	}
	req.AllowCloud = true
	if out, e := r.Explain(context.Background(), req); e != nil || out.Tier != TierCloud || h.gotPrompt != "" || strings.Contains(c.gotPrompt, "private question") {
		t.Fatal(out, e)
	}
	c.gotPrompt = ""
	h.available = false
	req.Provider = "homelab"
	if _, e := r.Explain(context.Background(), req); e == nil || c.gotPrompt != "" {
		t.Fatal("forced private mode cascaded to cloud")
	}
	req.Provider = "unknown"
	if _, e := r.Explain(context.Background(), req); e == nil {
		t.Fatal("invalid route accepted")
	}
}

func TestCoachingRejectsInventedValuesDestinationsAndTrailingOutput(t *testing.T) {
	facts := map[string]string{"cash": "Solde calculé : 1,23 €."}
	for _, raw := range []string{`{"facts":["cash"],"answer":"Investis tout"}`, `{"facts":["cash"],"actions":["buy_bitcoin"]}`, `{"facts":["cash"],"explanations":["promised_return"]}`, `{"facts":["cash"]} {}`, `{"facts":[]}`, `null`} {
		if _, _, e := SelectCoaching(raw, facts); e == nil {
			t.Fatal("unsafe output accepted", raw)
		}
	}
	text, actions, e := SelectCoaching(`{"facts":["cash"],"explanations":["cash_vs_wealth"],"actions":["journey"]}`, facts)
	if e != nil || !strings.Contains(text, "1,23") || !strings.Contains(text, CoachingExplanations["cash_vs_wealth"]) || len(actions) != 1 || actions[0].ID != "journey" {
		t.Fatal(text, actions, e)
	}
	if _, _, ok := LearningAnswer("Quel est mon budget de 500 euros ?"); ok {
		t.Fatal("personal amount silently replaced by teaching")
	}
	if _, actions, ok := LearningAnswer("Par où commencer ?"); !ok || len(actions) != 1 || actions[0].ID != "journey" {
		t.Fatal(actions, ok)
	}
}

func TestOllamaChecksInstalledModelAuthenticationAndStructuredChat(t *testing.T) {
	var chat map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer synthetic-private-token" {
			t.Error("missing private proxy authentication")
		}
		if r.URL.Path == "/api/tags" {
			io.WriteString(w, `{"models":[{"name":"qwen3.5:9b"}]}`)
			return
		}
		json.NewDecoder(r.Body).Decode(&chat)
		io.WriteString(w, `{"message":{"content":"{\"facts\":[\"cash\"]}"}}`)
	}))
	defer server.Close()
	provider := NewOllamaAuthenticated(server.URL, "qwen3.5:9b", "synthetic-private-token")
	if !provider.Available(context.Background()) {
		t.Fatal("installed model unavailable")
	}
	missing := NewOllamaAuthenticated(server.URL, "missing:9b", "synthetic-private-token")
	if missing.Available(context.Background()) {
		t.Fatal("HTTP 200 reported nonexistent model available")
	}
	if _, e := provider.Generate(context.Background(), CoachingSystem, "synthetic facts", 500); e != nil {
		t.Fatal(e)
	}
	if chat["format"] != "json" || chat["think"] != false || chat["model"] != "qwen3.5:9b" {
		t.Fatal(chat)
	}
}

func TestCloudModelIsConfigurableWithoutModelSpecificBeta(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["model"] != "synthetic-configured-model" || body["fallbacks"] != nil || r.Header.Get("anthropic-beta") != "" {
			t.Error(body, r.Header.Get("anthropic-beta"))
		}
		io.WriteString(w, `{"id":"msg_test","type":"message","role":"assistant","model":"test","content":[{"type":"text","text":"{\"facts\":[\"summary\"]}"}],"stop_reason":"end_turn","usage":{"input_tokens":1,"output_tokens":1}}`)
	}))
	defer server.Close()
	p := NewAnthropicWithModel("synthetic-key", "synthetic-configured-model", option.WithBaseURL(server.URL), option.WithMaxRetries(0))
	if _, e := p.Generate(context.Background(), CoachingSystem, `{"intent":"overview"}`, 500); e != nil {
		t.Fatal(e)
	}
}
