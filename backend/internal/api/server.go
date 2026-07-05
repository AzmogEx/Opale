// Package api expose l'API HTTP REST d'Opale.
package api

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/opale-app/opale/internal/ai"
	"github.com/opale-app/opale/internal/config"
	"github.com/opale-app/opale/internal/store"
)

// Server porte les dépendances des handlers HTTP.
type Server struct {
	store *store.Store
	cfg   config.Config
	log   *slog.Logger
	ai    *ai.Router
}

// NewServer construit le serveur d'API. La cascade IA est assemblée depuis
// la configuration : chaque niveau absent est simplement ignoré (EIA-021).
func NewServer(st *store.Store, cfg config.Config, log *slog.Logger) *Server {
	var homelab, cloud ai.Provider
	if cfg.OllamaURL != "" {
		homelab = ai.NewOllama(cfg.OllamaURL, cfg.OllamaModel)
		log.Info("ai: niveau N2 (homelab) configuré", "url", cfg.OllamaURL, "model", cfg.OllamaModel)
	}
	if cfg.AnthropicAPIKey != "" && cfg.CloudAI {
		cloud = ai.NewAnthropic(cfg.AnthropicAPIKey)
		log.Info("ai: niveau N3 (cloud Fable 5) configuré")
	}
	return &Server{store: st, cfg: cfg, log: log, ai: ai.NewRouter(homelab, cloud, log)}
}

// Routes construit le routeur HTTP complet.
func (s *Server) Routes() http.Handler {
	r := chi.NewRouter()
	r.Use(requestID, s.logRequests, s.recoverer)

	// Sondes de disponibilité (non authentifiées).
	r.Get("/healthz", s.handleHealthz)
	r.Get("/readyz", s.handleReadyz)

	r.Route("/v1", func(r chi.Router) {
		// Public : sélection et création de profil, connexion.
		r.Get("/profiles", s.handleListProfiles)
		r.Post("/profiles", s.handleCreateProfile)
		r.Post("/auth/login", s.handleLogin)

		// Authentifié : tout le reste est cloisonné par profil (EF-001).
		r.Group(func(r chi.Router) {
			r.Use(s.requireAuth)

			r.Post("/auth/logout", s.handleLogout)
			r.Get("/me", s.handleMe)
			r.Get("/net-worth", s.handleNetWorth)
			r.Get("/net-worth/history", s.handleNetWorthHistory)
			r.Get("/projection", s.handleProjection)
			r.Get("/categories", s.handleListCategories)

			// Le cerveau (P5)
			r.Get("/twin", s.handleTwin)
			r.Get("/risks", s.handleRisks)
			r.Post("/decision", s.handleDecision)
			r.Get("/monthly-review", s.handleMonthlyReview)
			r.Post("/assistant/ask", s.handleAssistantAsk)
			r.Get("/assistant/status", s.handleAssistantStatus)

			// Pilotage (P4)
			r.Get("/recurring", s.handleRecurring)
			r.Get("/cashflow", s.handleCashflow)
			r.Get("/health-score", s.handleHealthScore)
			r.Get("/alerts", s.handleAlerts)

			r.Route("/envelopes", func(r chi.Router) {
				r.Get("/", s.handleEnvelopeStatuses)
				r.Put("/", s.handleUpsertEnvelope)
				r.Delete("/{id}", s.handleDeleteEnvelope)
			})

			r.Route("/goals", func(r chi.Router) {
				r.Get("/", s.handleListGoals)
				r.Post("/", s.handleCreateGoal)
				r.Delete("/{id}", s.handleDeleteGoal)
			})

			r.Route("/transactions", func(r chi.Router) {
				r.Get("/", s.handleListTransactions)
				r.Post("/", s.handleCreateTransaction)
				r.Get("/summary", s.handleMonthSummary)
				r.Post("/import", s.handleImportCSV)
				r.Route("/{id}", func(r chi.Router) {
					r.Patch("/", s.handleUpdateTransaction)
					r.Delete("/", s.handleDeleteTransaction)
				})
			})

			r.Route("/assets", func(r chi.Router) {
				r.Get("/", s.handleListAssets)
				r.Post("/", s.handleCreateAsset)
				r.Route("/{id}", func(r chi.Router) {
					r.Get("/", s.handleGetAsset)
					r.Patch("/", s.handleUpdateAsset)
					r.Delete("/", s.handleDeleteAsset)
					r.Get("/valuations", s.handleListAssetValuations)
					r.Post("/valuations", s.handleAddAssetValuation)
				})
			})

			r.Route("/liabilities", func(r chi.Router) {
				r.Get("/", s.handleListLiabilities)
				r.Post("/", s.handleCreateLiability)
				r.Route("/{id}", func(r chi.Router) {
					r.Get("/", s.handleGetLiability)
					r.Patch("/", s.handleUpdateLiability)
					r.Delete("/", s.handleDeleteLiability)
					r.Get("/valuations", s.handleListLiabilityValuations)
					r.Post("/valuations", s.handleAddLiabilityValuation)
				})
			})
		})
	})

	return r
}
