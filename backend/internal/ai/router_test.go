package ai

import (
	"context"
	"errors"
	"log/slog"
	"testing"
)

// fakeProvider — un niveau de cascade simulé, qui enregistre le prompt reçu.
type fakeProvider struct {
	name      string
	tier      string
	available bool
	fail      bool
	gotPrompt string
}

func (f *fakeProvider) Name() string                    { return f.name }
func (f *fakeProvider) Tier() string                    { return f.tier }
func (f *fakeProvider) Available(context.Context) bool  { return f.available }
func (f *fakeProvider) Generate(_ context.Context, _, prompt string, _ int) (string, error) {
	f.gotPrompt = prompt
	if f.fail {
		return "", errors.New("panne simulée")
	}
	return "réponse-" + f.name, nil
}

func testLogger() *slog.Logger { return slog.New(slog.DiscardHandler) }

func req(allowCloud bool) Request {
	return Request{
		Task:             "test",
		Prompt:           "PROMPT-COMPLET montants exacts 42300",
		AnonymizedPrompt: "PROMPT-ANONYMISÉ cash 42k",
		AllowCloud:       allowCloud,
	}
}

func TestRouterPrefersHomelab(t *testing.T) {
	homelab := &fakeProvider{name: "ollama", tier: TierHomelab, available: true}
	cloud := &fakeProvider{name: "anthropic", tier: TierCloud, available: true}
	r := NewRouter(homelab, cloud, testLogger())

	resp, err := r.Explain(context.Background(), req(true))
	if err != nil {
		t.Fatal(err)
	}
	if resp.Tier != TierHomelab {
		t.Fatalf("attendu n2, obtenu %q", resp.Tier)
	}
	// Le homelab est privé : il reçoit le contexte COMPLET.
	if homelab.gotPrompt != "PROMPT-COMPLET montants exacts 42300" {
		t.Fatalf("le homelab doit recevoir le prompt complet, reçu %q", homelab.gotPrompt)
	}
	if cloud.gotPrompt != "" {
		t.Fatal("le cloud ne doit pas être appelé quand le homelab répond")
	}
}

func TestRouterCascadesToCloudAnonymizedOnly(t *testing.T) {
	homelab := &fakeProvider{name: "ollama", tier: TierHomelab, available: false}
	cloud := &fakeProvider{name: "anthropic", tier: TierCloud, available: true}
	r := NewRouter(homelab, cloud, testLogger())

	resp, err := r.Explain(context.Background(), req(true))
	if err != nil {
		t.Fatal(err)
	}
	if resp.Tier != TierCloud {
		t.Fatalf("attendu n3, obtenu %q", resp.Tier)
	}
	// GARDE-FOU EIA-031/033 : le cloud ne reçoit QUE la version anonymisée.
	if cloud.gotPrompt != "PROMPT-ANONYMISÉ cash 42k" {
		t.Fatalf("le cloud doit recevoir le prompt anonymisé, reçu %q", cloud.gotPrompt)
	}
}

func TestRouterCloudRequiresConsent(t *testing.T) {
	cloud := &fakeProvider{name: "anthropic", tier: TierCloud, available: true}
	r := NewRouter(nil, cloud, testLogger())

	// Sans consentement (EIA-022) : repli déterministe, cloud jamais appelé.
	if _, err := r.Explain(context.Background(), req(false)); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("attendu ErrUnavailable, obtenu %v", err)
	}
	if cloud.gotPrompt != "" {
		t.Fatal("le cloud ne doit jamais être appelé sans consentement")
	}
}

func TestRouterCloudRequiresAnonymizedPrompt(t *testing.T) {
	cloud := &fakeProvider{name: "anthropic", tier: TierCloud, available: true}
	r := NewRouter(nil, cloud, testLogger())

	// Pas de variante anonymisée = interdit de cloud (EIA-032).
	request := req(true)
	request.AnonymizedPrompt = ""
	if _, err := r.Explain(context.Background(), request); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("attendu ErrUnavailable, obtenu %v", err)
	}
	if cloud.gotPrompt != "" {
		t.Fatal("le cloud ne doit jamais recevoir de données non anonymisées")
	}
}

func TestRouterFallsBackWhenHomelabFails(t *testing.T) {
	homelab := &fakeProvider{name: "ollama", tier: TierHomelab, available: true, fail: true}
	cloud := &fakeProvider{name: "anthropic", tier: TierCloud, available: true}
	r := NewRouter(homelab, cloud, testLogger())

	resp, err := r.Explain(context.Background(), req(true))
	if err != nil {
		t.Fatal(err)
	}
	if resp.Tier != TierCloud {
		t.Fatalf("échec homelab : cascade vers n3 attendue, obtenu %q", resp.Tier)
	}
}

func TestRouterNoProviders(t *testing.T) {
	r := NewRouter(nil, nil, testLogger())
	if _, err := r.Explain(context.Background(), req(true)); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("attendu ErrUnavailable, obtenu %v", err)
	}
}
