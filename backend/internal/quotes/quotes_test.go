package quotes

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/opale-app/opale/internal/money"
)

func TestParseDecimalMicro(t *testing.T) {
	cases := map[string]int64{
		"1.0832":    1_083_200,
		"58231.42":  58_231_420_000,
		"0.9421":    942_100,
		"161.53":    161_530_000,
		"3":         3_000_000,
		"0.0000012": 1, // tronqué à 6 décimales
	}
	for in, want := range cases {
		got, err := ParseDecimalMicro(in)
		if err != nil {
			t.Fatalf("%q : %v", in, err)
		}
		if got != want {
			t.Fatalf("%q → %d, attendu %d", in, got, want)
		}
	}
	if _, err := ParseDecimalMicro("abc"); err == nil {
		t.Fatal("chaîne invalide acceptée")
	}
}

func TestFetchCryptoPricesEUR(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v3/simple/price" {
			http.NotFound(w, r)
			return
		}
		w.Write([]byte(`{"bitcoin":{"eur":58231.42},"ethereum":{"eur":3120.5}}`))
	}))
	defer srv.Close()

	c := New(srv.Client())
	c.CoinGeckoURL = srv.URL
	prices, err := c.FetchCryptoPricesEUR(context.Background(), []string{"bitcoin", "ethereum"})
	if err != nil {
		t.Fatal(err)
	}
	if prices["bitcoin"] != money.Cents(5_823_142) {
		t.Fatalf("bitcoin %d centimes, attendu 5823142", prices["bitcoin"])
	}
	if prices["ethereum"] != money.Cents(312_050) {
		t.Fatalf("ethereum %d centimes, attendu 312050", prices["ethereum"])
	}
}

func TestFetchECBRates(t *testing.T) {
	const sample = `<?xml version="1.0" encoding="UTF-8"?>
<gesmes:Envelope xmlns:gesmes="http://www.gesmes.org/xml/2002-08-01" xmlns="http://www.ecb.int/vocabulary/2002-08-01/eurofxref">
  <Cube><Cube time="2026-07-06">
    <Cube currency="USD" rate="1.0832"/>
    <Cube currency="GBP" rate="0.8532"/>
  </Cube></Cube>
</gesmes:Envelope>`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(sample))
	}))
	defer srv.Close()

	c := New(srv.Client())
	c.ECBURL = srv.URL
	rates, err := c.FetchECBRates(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	// 1 USD = 1e12/1_083_200 ≈ 923 189 micro-euros (≈ 0,92 €).
	if rates["USD"] < 923_000 || rates["USD"] > 923_400 {
		t.Fatalf("USD %d micro-euros, attendu ≈ 923189", rates["USD"])
	}
	// 1 GBP ≈ 1,172 € — supérieur à l'euro.
	if rates["GBP"] < 1_170_000 || rates["GBP"] > 1_175_000 {
		t.Fatalf("GBP %d micro-euros, attendu ≈ 1172033", rates["GBP"])
	}
}

func TestSubCentCryptoAndOverflow(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`{"tiny":{"eur":0.000123}}`)) }))
	defer s.Close()
	c := New(s.Client())
	c.CoinGeckoURL = s.URL
	p, e := c.FetchCryptoPricesMicroEUR(context.Background(), []string{"tiny"})
	if e != nil || p["tiny"] != 123 {
		t.Fatal(p, e)
	}
	if _, e = ParseDecimalMicro("999999999999999999999999999999999"); e == nil {
		t.Fatal("overflow accepted")
	}
}
