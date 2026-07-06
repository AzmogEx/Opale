// Package vault chiffre les documents du coffre-fort patrimonial (EF-064).
//
// AES-256-GCM, clé de 32 octets fournie hors code via OPALE_VAULT_KEY
// (64 caractères hexadécimaux — cf. cahier des charges §16 : secrets en
// variables d'environnement). Le contenu stocké en base est
// nonce (12 octets) || ciphertext ; le clair n'atteint jamais le disque.
package vault

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
)

// ErrNotConfigured : aucune clé de coffre n'est configurée.
var ErrNotConfigured = errors.New("vault: OPALE_VAULT_KEY absente — coffre-fort désactivé")

// ErrCorrupted : contenu illisible (clé différente ou données altérées).
var ErrCorrupted = errors.New("vault: document illisible (clé changée ou données altérées)")

// Vault chiffre et déchiffre avec une clé AES-256 fixe.
type Vault struct {
	aead cipher.AEAD
}

// New construit un coffre depuis la clé hexadécimale (64 caractères).
// Renvoie ErrNotConfigured si la clé est vide.
func New(hexKey string) (*Vault, error) {
	if hexKey == "" {
		return nil, ErrNotConfigured
	}
	key, err := hex.DecodeString(hexKey)
	if err != nil || len(key) != 32 {
		return nil, fmt.Errorf("vault: OPALE_VAULT_KEY doit faire 64 caractères hexadécimaux (32 octets)")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Vault{aead: aead}, nil
}

// Encrypt scelle le clair : nonce || ciphertext.
func (v *Vault) Encrypt(plain []byte) ([]byte, error) {
	nonce := make([]byte, v.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	return v.aead.Seal(nonce, nonce, plain, nil), nil
}

// Decrypt ouvre nonce || ciphertext et vérifie l'intégrité (GCM).
func (v *Vault) Decrypt(sealed []byte) ([]byte, error) {
	if len(sealed) < v.aead.NonceSize() {
		return nil, ErrCorrupted
	}
	nonce, ciphertext := sealed[:v.aead.NonceSize()], sealed[v.aead.NonceSize():]
	plain, err := v.aead.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, ErrCorrupted
	}
	return plain, nil
}
