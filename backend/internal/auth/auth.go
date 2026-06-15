// Package auth fournit le hachage des codes (PIN) et la génération/validation des
// jetons de session opaques (EF-002).
package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// HashPIN hache un code/PIN avec bcrypt.
func HashPIN(pin string) (string, error) {
	h, err := bcrypt.GenerateFromPassword([]byte(pin), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("auth: hachage du PIN : %w", err)
	}
	return string(h), nil
}

// CheckPIN vérifie un PIN contre son hash bcrypt.
func CheckPIN(hash, pin string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pin)) == nil
}

// NewToken génère un jeton de session opaque (256 bits, encodé en hexadécimal).
// Le jeton clair est renvoyé au client ; seule sa version hachée est stockée.
func NewToken() (token string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("auth: génération du jeton : %w", err)
	}
	return hex.EncodeToString(b), nil
}

// HashToken renvoie le SHA-256 d'un jeton (pour stockage et comparaison).
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
