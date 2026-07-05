// Package api expose l'API HTTP REST d'Opale.
package api

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/opale-app/opale/internal/config"
	"github.com/opale-app/opale/internal/store"
)

// Server porte les dépendances des handlers HTTP.
type Server struct {
	store *store.Store
	cfg   config.Config
	log   *slog.Logger
}

// NewServer construit le serveur d'API.
func NewServer(st *store.Store, cfg config.Config, log *slog.Logger) *Server {
	return &Server{store: st, cfg: cfg, log: log}
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
