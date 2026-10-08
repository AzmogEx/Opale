// Package push — notifications APNs réelles, activables par configuration
// (OPALE_APNS_*). Implémentation directe du protocole : JWT ES256 signé avec
// la clé .p8 + HTTP/2 (natif dans net/http) — zéro dépendance externe.
package push

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// Client envoie des notifications via APNs.
type Client struct {
	HTTP     *http.Client
	BaseURL  string // surchargé en test ; sinon dérivé de l'environnement
	key      *ecdsa.PrivateKey
	keyID    string
	teamID   string
	bundleID string

	mu       sync.Mutex
	token    string
	tokenExp time.Time
}

// New construit le client depuis la clé .p8 (PEM). env : "sandbox" ou
// "production" (les jetons d'un build Xcode direct vivent en sandbox).
func New(keyP8PEM, keyID, teamID, bundleID, env string) (*Client, error) {
	block, _ := pem.Decode([]byte(keyP8PEM))
	if block == nil {
		return nil, fmt.Errorf("apns : clé .p8 illisible (PEM attendu)")
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("apns : clé .p8 invalide : %w", err)
	}
	key, ok := parsed.(*ecdsa.PrivateKey)
	if !ok || key.Curve != elliptic.P256() {
		return nil, fmt.Errorf("apns : la clé .p8 n'est pas une clé EC (ES256)")
	}
	base := "https://api.sandbox.push.apple.com"
	if env != "production" && env != "sandbox" {
		return nil, fmt.Errorf("apns: environnement attendu sandbox ou production")
	}
	if env == "production" {
		base = "https://api.push.apple.com"
	}
	return &Client{
		HTTP:     &http.Client{Timeout: 15 * time.Second},
		BaseURL:  base,
		key:      key,
		keyID:    keyID,
		teamID:   teamID,
		bundleID: bundleID,
	}, nil
}

// bearer renvoie un JWT valide (APNs accepte un jeton pendant 60 min ;
// on le renouvelle à 50 pour la marge).
func (c *Client) bearer() (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token != "" && time.Now().Before(c.tokenExp) {
		return c.token, nil
	}
	b64 := func(v []byte) string { return base64.RawURLEncoding.EncodeToString(v) }
	header, _ := json.Marshal(map[string]string{"alg": "ES256", "kid": c.keyID})
	claims, _ := json.Marshal(map[string]any{"iss": c.teamID, "iat": time.Now().Unix()})
	signing := b64(header) + "." + b64(claims)

	digest := sha256.Sum256([]byte(signing))
	r, s, err := ecdsa.Sign(rand.Reader, c.key, digest[:])
	if err != nil {
		return "", fmt.Errorf("apns : signature JWT : %w", err)
	}
	// ES256 : signature = r ‖ s, chacun sur 32 octets.
	sig := make([]byte, 64)
	r.FillBytes(sig[:32])
	s.FillBytes(sig[32:])

	c.token = signing + "." + b64(sig)
	c.tokenExp = time.Now().Add(50 * time.Minute)
	return c.token, nil
}

// ErrBadToken signale un jeton d'appareil à retirer de la base.
type ErrBadToken struct{ Reason string }

func (e ErrBadToken) Error() string { return "apns : jeton invalide (" + e.Reason + ")" }

// Send pousse une alerte simple (titre + corps) vers un jeton d'appareil.
func (c *Client) Send(ctx context.Context, deviceToken, title, body string, profileID ...string) error {
	token, err := c.bearer()
	if err != nil {
		return err
	}
	message := map[string]any{
		"destination": "alerts",
		"aps": map[string]any{
			"alert":     map[string]string{"title": title, "body": body},
			"sound":     "default",
			"thread-id": "opale-alerts",
		},
	}
	if len(profileID) > 0 {
		message["profile_id"] = profileID[0]
	}
	payload, _ := json.Marshal(message)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.BaseURL+"/3/device/"+deviceToken, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("authorization", "bearer "+token)
	req.Header.Set("apns-topic", c.bundleID)
	req.Header.Set("apns-push-type", "alert")
	req.Header.Set("apns-priority", "10")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("apns injoignable : %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		return nil
	}
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
	var apnsErr struct {
		Reason string `json:"reason"`
	}
	_ = json.Unmarshal(raw, &apnsErr)
	// 410 Gone / BadDeviceToken : l'appareil n'existe plus — à purger.
	if resp.StatusCode == http.StatusGone || apnsErr.Reason == "BadDeviceToken" ||
		apnsErr.Reason == "Unregistered" {
		return ErrBadToken{Reason: apnsErr.Reason}
	}
	return fmt.Errorf("apns : statut %d (%s)", resp.StatusCode, apnsErr.Reason)
}
