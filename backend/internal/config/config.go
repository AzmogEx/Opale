// Package config charge la configuration du backend depuis l'environnement.
// Aucun secret n'est codé en dur (cf. cahier des charges §16 — secrets hors code).
package config

import (
	"fmt"
	"os"
	"time"
)

// Config regroupe tous les paramètres d'exécution du backend.
type Config struct {
	HTTPAddr   string
	Env        string
	LogLevel   string
	DatabaseURL string
	SessionTTL time.Duration

	// IA en cascade (P5, EIA-001→003).
	// N2 — homelab : URL d'un serveur Ollama (vide = niveau indisponible).
	OllamaURL   string
	OllamaModel string
	// N3 — cloud : clé API Anthropic (vide = niveau indisponible) et
	// interrupteur global (EIA-022 : le cloud reste opt-in).
	AnthropicAPIKey string
	CloudAI         bool

	// Coffre-fort (P6, EF-064) : clé AES-256 en hexadécimal (64 caractères).
	// Vide = coffre désactivé (les endpoints documents renvoient 503).
	VaultKey string

	// Synchro bancaire GoCardless (P7, EF-071). Vide = désactivée.
	GCSecretID  string
	GCSecretKey string
	GCBaseURL   string // vide = API officielle (surchargé en test)

	// CORS (future app web) : origines autorisées, séparées par des
	// virgules. Vide = CORS désactivé.
	CORSOrigins string
}

// Load lit la configuration depuis les variables d'environnement (préfixe OPALE_),
// en appliquant des valeurs par défaut raisonnables pour le développement local.
func Load() (Config, error) {
	c := Config{
		HTTPAddr:   env("OPALE_HTTP_ADDR", ":8080"),
		Env:        env("OPALE_ENV", "dev"),
		LogLevel:   env("OPALE_LOG_LEVEL", "info"),
		DatabaseURL: os.Getenv("OPALE_DATABASE_URL"),
	}

	if c.DatabaseURL == "" {
		c.DatabaseURL = fmt.Sprintf(
			"postgres://%s:%s@%s:%s/%s?sslmode=%s",
			env("OPALE_DB_USER", "opale"),
			env("OPALE_DB_PASSWORD", "opale"),
			env("OPALE_DB_HOST", "localhost"),
			env("OPALE_DB_PORT", "5432"),
			env("OPALE_DB_NAME", "opale"),
			env("OPALE_DB_SSLMODE", "disable"),
		)
	}

	ttl, err := time.ParseDuration(env("OPALE_SESSION_TTL", "720h"))
	if err != nil {
		return Config{}, fmt.Errorf("OPALE_SESSION_TTL invalide : %w", err)
	}
	c.SessionTTL = ttl

	// IA en cascade — tout est optionnel : sans provider, l'app reste
	// pleinement fonctionnelle sur le moteur déterministe (EIA-021).
	c.OllamaURL = os.Getenv("OPALE_OLLAMA_URL")
	c.OllamaModel = env("OPALE_OLLAMA_MODEL", "llama3.1:8b")
	c.AnthropicAPIKey = os.Getenv("OPALE_ANTHROPIC_API_KEY")
	c.CloudAI = env("OPALE_CLOUD_AI", "on") != "off"
	c.VaultKey = os.Getenv("OPALE_VAULT_KEY")

	c.GCSecretID = os.Getenv("OPALE_GC_SECRET_ID")
	c.GCSecretKey = os.Getenv("OPALE_GC_SECRET_KEY")
	c.GCBaseURL = os.Getenv("OPALE_GC_BASE_URL")
	c.CORSOrigins = os.Getenv("OPALE_CORS_ORIGINS")

	return c, nil
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
