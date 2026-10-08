package jobs

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"github.com/opale-app/opale/internal/push"
	"github.com/opale-app/opale/internal/store"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
)

func TestPushPrivacyAndPersistentDeduplication(t *testing.T) {
	url := os.Getenv("OPALE_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("isolated PostgreSQL required")
	}
	ctx := context.Background()
	st, e := store.New(ctx, url)
	if e != nil {
		t.Fatal(e)
	}
	defer st.Close()
	if e = st.Migrate(ctx); e != nil {
		t.Fatal(e)
	}
	p, e := st.CreateProfile(ctx, "Secret Name", "hash", "N1")
	if e != nil {
		t.Fatal(e)
	}
	defer st.DeleteProfile(ctx, p.ID)
	alert, e := st.CreateCustomAlert(ctx, p.ID, "cash_below", 123456789)
	if e != nil {
		t.Fatal(e)
	}
	_ = alert
	token := strings.Repeat("a", 64)
	if e = st.UpsertPushToken(ctx, p.ID, token, "ios"); e != nil {
		t.Fatal(e)
	}
	var mu sync.Mutex
	calls := 0
	fake := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		mu.Lock()
		defer mu.Unlock()
		calls++
		if strings.Contains(string(b), "123456789") || strings.Contains(string(b), p.Name) {
			t.Error("sensitive push payload")
		}
		var payload map[string]any
		if e := json.Unmarshal(b, &payload); e != nil || payload["destination"] != "alerts" || payload["profile_id"] != p.ID {
			t.Error("missing destination")
		}
		w.WriteHeader(200)
	}))
	defer fake.Close()
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	der, _ := x509.MarshalPKCS8PrivateKey(key)
	client, e := push.New(string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})), "TEST", "TEST", "app.opale.ios", "sandbox")
	if e != nil {
		t.Fatal(e)
	}
	client.HTTP = fake.Client()
	client.BaseURL = fake.URL
	runner := &Runner{Store: st, Push: client, Log: slog.New(slog.NewTextHandler(io.Discard, nil))}
	runner.PushTriggeredAlerts(ctx)
	// New runner simulates restart; the delivery ledger must suppress duplication.
	again := &Runner{Store: st, Push: client, Log: runner.Log}
	again.PushTriggeredAlerts(ctx)
	mu.Lock()
	defer mu.Unlock()
	if calls != 1 {
		t.Fatalf("expected1 delivery got%d", calls)
	}
}
