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

	return c, nil
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
