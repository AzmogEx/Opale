package api

import (
	"io"
	"log/slog"
	"testing"

	"github.com/opale-app/opale/internal/ai"
)

func TestAssistantModeConsentPrivateIsolationAndIntegratedGuide(t *testing.T) {
	s, _ := testAPI(t)
	_, token := profile(t, s, "Assistant modes")
	private, cloud := &captureProvider{}, &captureProvider{}
	s.ai = ai.NewRouter(private, cloud, slog.New(slog.NewTextHandler(io.Discard, nil)))
	ask := func(question, mode string, consent bool) map[string]any {
		t.Helper()
		r := request(t, s, token, "POST", "/v1/assistant/ask", map[string]any{"question": question, "provider": mode, "allow_cloud": consent})
		if r.Code != 200 {
			t.Fatal(r.Code, r.Body)
		}
		return decode(t, r)
	}
	guide := ask("Par où commencer ?", "cloud", true)
	if guide["tier"] != "guide" || len(guide["actions"].([]any)) == 0 || private.calls != 0 || cloud.calls != 0 {
		t.Fatal(guide, private.calls, cloud.calls)
	}
	ask("Question arbitraire avec une note privée", "cloud", true)
	if private.calls != 0 || cloud.calls != 0 {
		t.Fatal("cloud mode sent free text to a model")
	}
	ask("Résume ma situation", "cloud", false)
	if private.calls != 0 || cloud.calls != 0 {
		t.Fatal("mode granted consent")
	}
	answer := ask("Résume ma situation", "cloud", true)
	if cloud.calls != 1 || private.calls != 0 || answer["tier"] != ai.TierCloud {
		t.Fatal(answer, private.calls, cloud.calls)
	}
	if r := request(t, s, token, "PATCH", "/v1/me", map[string]any{"privacy_default": "N1"}); r.Code != 200 {
		t.Fatal(r.Code, r.Body)
	}
	ask("Résume ma situation", "cloud", true)
	if cloud.calls != 1 || private.calls != 0 {
		t.Fatal("provider choice bypassed profile policy")
	}
	if r := request(t, s, token, "POST", "/v1/assistant/ask", map[string]any{"question": "Résume ma situation", "provider": "unknown"}); r.Code != 400 {
		t.Fatal(r.Code, r.Body)
	}
}
