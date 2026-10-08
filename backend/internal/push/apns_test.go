package push

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// testKey génère une clé P-256 encodée comme un vrai fichier .p8 d'Apple.
func testKey(t *testing.T) (string, *ecdsa.PrivateKey) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	pemStr := string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der}))
	return pemStr, key
}

func TestBearerJWTValid(t *testing.T) {
	pemStr, key := testKey(t)
	c, err := New(pemStr, "KEY123", "TEAM456", "app.opale.ios", "sandbox")
	if err != nil {
		t.Fatal(err)
	}
	token, err := c.bearer()
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("JWT en %d parties, attendu 3", len(parts))
	}
	// L'en-tête porte l'algorithme et le key ID.
	headerRaw, _ := base64.RawURLEncoding.DecodeString(parts[0])
	var header map[string]string
	if err := json.Unmarshal(headerRaw, &header); err != nil {
		t.Fatal(err)
	}
	if header["alg"] != "ES256" || header["kid"] != "KEY123" {
		t.Fatalf("en-tête inattendu : %v", header)
	}
	// La signature se vérifie avec la clé publique.
	digest := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	sig, _ := base64.RawURLEncoding.DecodeString(parts[2])
	if len(sig) != 64 {
		t.Fatalf("signature de %d octets, attendu 64", len(sig))
	}
	r := new(big.Int).SetBytes(sig[:32])
	s := new(big.Int).SetBytes(sig[32:])
	if !ecdsa.Verify(&key.PublicKey, digest[:], r, s) {
		t.Fatal("signature JWT invalide")
	}
	// Le jeton est réutilisé tant qu'il est frais.
	token2, _ := c.bearer()
	if token2 != token {
		t.Fatal("le JWT devrait être mis en cache")
	}
}

func TestSendOKAndBadToken(t *testing.T) {
	pemStr, _ := testKey(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("apns-topic") != "app.opale.ios" ||
			!strings.HasPrefix(r.Header.Get("authorization"), "bearer ") {
			t.Errorf("en-têtes APNs manquants : %v", r.Header)
		}
		if strings.HasSuffix(r.URL.Path, "/dead-token") {
			w.WriteHeader(http.StatusGone)
			w.Write([]byte(`{"reason":"Unregistered"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c, err := New(pemStr, "K", "T", "app.opale.ios", "sandbox")
	if err != nil {
		t.Fatal(err)
	}
	c.BaseURL = srv.URL
	c.HTTP = srv.Client()

	if err := c.Send(context.Background(), "good-token", "Titre", "Corps"); err != nil {
		t.Fatalf("envoi OK attendu : %v", err)
	}
	err = c.Send(context.Background(), "dead-token", "Titre", "Corps")
	var bad ErrBadToken
	if !errors.As(err, &bad) {
		t.Fatalf("ErrBadToken attendu, obtenu %v", err)
	}
}

func TestNewRejectsGarbage(t *testing.T) {
	if _, err := New("pas du PEM", "K", "T", "B", "sandbox"); err == nil {
		t.Fatal("clé invalide acceptée")
	}
}
