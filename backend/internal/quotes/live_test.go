package quotes

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"
)

// Explicitly opt-in: public symbols only, never portfolio data or credentials.
func TestPublicProvidersLive(t *testing.T) {
	if os.Getenv("OPALE_TEST_PUBLIC_QUOTES") != "1" {
		t.Skip("public network test opt-in")
	}
	c := New(&http.Client{Timeout: 20 * time.Second})
	ctx := context.Background()
	t.Run("ECB", func(t *testing.T) {
		r, d, e := c.FetchECBRatesDated(ctx)
		if e != nil {
			t.Fatal(e)
		}
		if r["USD"] <= 0 || time.Since(d) > 7*24*time.Hour {
			t.Fatal("missing/stale publication")
		}
		t.Log("ECB publication", d.Format("2006-01-02"))
	})
	t.Run("CoinGecko", func(t *testing.T) {
		r, e := c.FetchCryptoPricesMicroEUR(ctx, []string{"bitcoin"})
		if e != nil {
			t.Fatal(e)
		}
		if r["bitcoin"] <= 0 {
			t.Fatal("missing quote")
		}
		t.Log("public BTC quote received")
	})
}
