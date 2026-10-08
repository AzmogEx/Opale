// Package quotes — récupération automatique des cours (OPT-IN, EF-008/EF-016).
//
// Deux sources publiques, sans clé ni compte :
//   - CoinGecko (simple/price) pour les cryptos, en euros ;
//   - la BCE (eurofxref-daily.xml) pour les devises.
//
// Confidentialité : les requêtes ne contiennent AUCUNE donnée personnelle
// (juste des symboles publics) et la fonctionnalité est désactivée par
// défaut (OPALE_AUTO_QUOTES=on pour l'activer). Les montants sont convertis
// en entiers dès le parsing — aucun float ne circule (ENF-007).
package quotes

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/opale-app/opale/internal/money"
)

// Client interroge les sources de cours. Les URLs sont paramétrables pour
// les tests (serveur factice) et un éventuel miroir auto-hébergé.
type Client struct {
	HTTP         *http.Client
	CoinGeckoURL string // défaut : https://api.coingecko.com
	ECBURL       string // défaut : https://www.ecb.europa.eu/stats/eurofxref/eurofxref-daily.xml
}

// New construit un client avec les URLs publiques par défaut.
func New(httpClient *http.Client) *Client {
	return &Client{
		HTTP:         httpClient,
		CoinGeckoURL: "https://api.coingecko.com",
		ECBURL:       "https://www.ecb.europa.eu/stats/eurofxref/eurofxref-daily.xml",
	}
}

// ParseDecimalMicro convertit une chaîne décimale (« 1.0832 », « 58231.42 »)
// en millionièmes (1.0832 → 1_083_200) — arithmétique entière pure.
func ParseDecimalMicro(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("décimal vide")
	}
	neg := false
	if s[0] == '-' {
		neg, s = true, s[1:]
	} else if s[0] == '+' {
		s = s[1:]
	}
	intPart, fracPart := s, ""
	if i := strings.IndexAny(s, "."); i >= 0 {
		intPart, fracPart = s[:i], s[i+1:]
	}
	if intPart == "" {
		intPart = "0"
	}
	// 6 décimales max, tronquées au-delà.
	if len(fracPart) > 6 {
		fracPart = fracPart[:6]
	}
	for len(fracPart) < 6 {
		fracPart += "0"
	}
	digits := intPart + fracPart
	if neg {
		digits = "-" + digits
	}
	value, err := strconv.ParseInt(digits, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("décimal hors limites")
	}
	return value, nil
}

// FetchCryptoPricesEUR interroge CoinGecko pour une liste d'identifiants
// (« bitcoin », « ethereum »…) et renvoie le cours unitaire en CENTIMES.
func (c *Client) FetchCryptoPricesEUR(ctx context.Context, ids []string) (map[string]money.Cents, error) {
	raw, e := c.FetchCryptoPricesMicroEUR(ctx, ids)
	out := map[string]money.Cents{}
	for k, v := range raw {
		out[k] = money.Cents(v / 10000)
	}
	return out, e
}
func (c *Client) FetchCryptoPricesMicroEUR(ctx context.Context, ids []string) (map[string]int64, error) {
	if len(ids) == 0 {
		return map[string]int64{}, nil
	}
	u := c.CoinGeckoURL + "/api/v3/simple/price?vs_currencies=eur&precision=6&ids=" +
		url.QueryEscape(strings.Join(ids, ","))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("CoinGecko injoignable : %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("CoinGecko : statut %d", resp.StatusCode)
	}

	// {"bitcoin":{"eur":58231.42},...} — json.Number pour rester en texte.
	dec := json.NewDecoder(io.LimitReader(resp.Body, 1<<20))
	dec.UseNumber()
	var raw map[string]map[string]json.Number
	if err := dec.Decode(&raw); err != nil {
		return nil, fmt.Errorf("CoinGecko : réponse illisible : %w", err)
	}
	out := make(map[string]int64, len(raw))
	for id, prices := range raw {
		n, ok := prices["eur"]
		if !ok {
			continue
		}
		micro, err := ParseDecimalMicro(n.String())
		if err != nil {
			return nil, fmt.Errorf("CoinGecko %s : %w", id, err)
		}
		if micro <= 0 {
			continue
		}
		out[id] = micro // preserve sub-cent prices until quantity multiplication
	}
	return out, nil
}

// ecbEnvelope — la structure minimale du XML eurofxref-daily.
type ecbEnvelope struct {
	Day struct {
		Value string `xml:"time,attr"`
		Cubes []struct {
			Currency string `xml:"currency,attr"`
			Rate     string `xml:"rate,attr"`
		} `xml:"Cube"`
	} `xml:"Cube>Cube"`
}

// FetchECBRates lit les taux de référence BCE et renvoie, par devise,
// la valeur d'UNE unité en micro-euros (compatible fx_rates.rate_micro).
// La BCE publie « 1 EUR = X devise » : on inverse en entiers.
func (c *Client) FetchECBRates(ctx context.Context) (map[string]int64, error) {
	rates, _, e := c.FetchECBRatesDated(ctx)
	return rates, e
}
func (c *Client) FetchECBRatesDated(ctx context.Context) (map[string]int64, time.Time, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.ECBURL, nil)
	if err != nil {
		return nil, time.Time{}, err
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, time.Time{}, fmt.Errorf("BCE injoignable : %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, time.Time{}, fmt.Errorf("BCE : statut %d", resp.StatusCode)
	}

	var env ecbEnvelope
	if err := xml.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&env); err != nil {
		return nil, time.Time{}, fmt.Errorf("BCE : XML illisible : %w", err)
	}
	out := make(map[string]int64, len(env.Day.Cubes))
	for _, cube := range env.Day.Cubes {
		perEuroMicro, err := ParseDecimalMicro(cube.Rate) // 1 EUR = X devise
		if err != nil || perEuroMicro <= 0 {
			continue
		}
		// 1 devise = 1e12 / X micro-euros.
		out[cube.Currency] = 1_000_000_000_000 / perEuroMicro
	}
	if len(out) == 0 {
		return nil, time.Time{}, fmt.Errorf("BCE : aucun taux dans la réponse")
	}
	day, e := time.Parse("2006-01-02", env.Day.Value)
	if e != nil || day.After(time.Now()) {
		return nil, time.Time{}, fmt.Errorf("BCE: date publication invalide")
	}
	return out, day, nil
}
