package vault

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

const testKey = "0f1e2d3c4b5a69788796a5b4c3d2e1f00112233445566778899aabbccddeeff0"

func TestRoundTrip(t *testing.T) {
	v, err := New(testKey)
	if err != nil {
		t.Fatal(err)
	}
	plain := []byte("acte notarié — très confidentiel (N3)")
	sealed, err := v.Encrypt(plain)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(sealed, []byte("notarié")) {
		t.Fatal("le contenu chiffré ne doit pas contenir le clair")
	}
	got, err := v.Decrypt(sealed)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, plain) {
		t.Fatalf("aller-retour cassé : %q", got)
	}
}

func TestTamperDetection(t *testing.T) {
	v, _ := New(testKey)
	sealed, _ := v.Encrypt([]byte("données"))
	sealed[len(sealed)-1] ^= 0xFF // altération d'un octet
	if _, err := v.Decrypt(sealed); !errors.Is(err, ErrCorrupted) {
		t.Fatalf("altération non détectée : %v", err)
	}
}

func TestWrongKey(t *testing.T) {
	v1, _ := New(testKey)
	v2, _ := New(strings.Repeat("ab", 32))
	sealed, _ := v1.Encrypt([]byte("données"))
	if _, err := v2.Decrypt(sealed); !errors.Is(err, ErrCorrupted) {
		t.Fatalf("mauvaise clé non détectée : %v", err)
	}
}

func TestKeyValidation(t *testing.T) {
	if _, err := New(""); !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("clé vide : attendu ErrNotConfigured, obtenu %v", err)
	}
	if _, err := New("trop-court"); err == nil {
		t.Fatal("clé invalide acceptée")
	}
}
