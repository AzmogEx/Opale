// Package api expose l'API HTTP REST d'Opale.
package api

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/opale-app/opale/internal/ai"
	"github.com/opale-app/opale/internal/bank"
	"github.com/opale-app/opale/internal/config"
	"github.com/opale-app/opale/internal/store"
	"github.com/opale-app/opale/internal/vault"
)

// Server porte les dépendances des handlers HTTP.
type Server struct {
	store  *store.Store
	cfg    config.Config
	log    *slog.Logger
	ai     *ai.Router
	vault  *vault.Vault     // nil = coffre-fort désactivé (EF-064)
	bank   *bank.GoCardless // nil = synchro bancaire désactivée (EF-071)
	logins *loginLimiter    // anti brute-force du PIN (audit)
}

// journal trace un événement sensible dans le journal d'accès (ENF-004),
// sans jamais bloquer la requête en cours.
func (s *Server) journal(r *http.Request, profileID *string, event, detail string) {
	if err := s.store.LogAccess(r.Context(), profileID, event, detail); err != nil {
		s.log.Warn("journal d'accès", "event", event, "err", err)
	}
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

	var v *vault.Vault
	if cfg.VaultKey != "" {
		var err error
		if v, err = vault.New(cfg.VaultKey); err != nil {
			log.Error("vault: clé invalide — coffre-fort désactivé", "err", err)
			v = nil
		} else {
			log.Info("vault: coffre-fort activé (AES-256-GCM)")
		}
	}

	var gc *bank.GoCardless
	if cfg.GCSecretID != "" && cfg.GCSecretKey != "" {
		gc = bank.New(cfg.GCBaseURL, cfg.GCSecretID, cfg.GCSecretKey)
		log.Info("bank: synchro GoCardless configurée")
	}

	return &Server{store: st, cfg: cfg, log: log,
		ai: ai.NewRouter(homelab, cloud, log), vault: v, bank: gc,
		logins: newLoginLimiter()}
}

// Routes construit le routeur HTTP complet.
func (s *Server) Routes() http.Handler {
	r := chi.NewRouter()
	r.Use(requestID, s.logRequests, s.recoverer)
	if s.cfg.CORSOrigins != "" {
		r.Use(corsMiddleware(s.cfg.CORSOrigins))
	}

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
			r.Get("/export", s.handleExport)
			r.Get("/access-log", s.handleAccessLog)
			r.Delete("/me/data", s.handleResetData)
			r.Delete("/me", s.handleDeleteMe)
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

			// Espace partagé (EF-007) & devises (EF-008)
			r.Route("/spaces", func(r chi.Router) {
				r.Get("/", s.handleListSpaces)
				r.Post("/", s.handleCreateSpace)
				r.Get("/{id}", s.handleSpaceDetail)
				r.Post("/{id}/members", s.handleAddSpaceMember)
				r.Delete("/{id}/members/{profileID}", s.handleRemoveSpaceMember)
			})
			r.Route("/fx", func(r chi.Router) {
				r.Get("/", s.handleListFX)
				r.Put("/{currency}", s.handleUpsertFX)
				r.Delete("/{currency}", s.handleDeleteFX)
			})

			// Le confort (P7)
			r.Post("/scenarios/compare", s.handleCompareScenarios)
			r.Get("/company", s.handleCompanies)
			r.Route("/bank", func(r chi.Router) {
				r.Get("/status", s.handleBankStatus)
				r.Get("/institutions", s.handleBankInstitutions)
				r.Post("/connect", s.handleBankConnect)
				r.Post("/sync", s.handleBankSync)
				r.Delete("/links/{id}", s.handleBankDisconnect)
			})

			// La profondeur (P6)
			r.Get("/real-estate", s.handleRealEstate)
			r.Get("/investments", s.handleInvestments)
			r.Get("/objects", s.handleObjects)
			r.Get("/timeline", s.handleTimeline)
			r.Get("/transmission", s.handleTransmission)

			r.Route("/documents", func(r chi.Router) {
				r.Get("/", s.handleListDocuments)
				r.Post("/", s.handleCreateDocument)
				r.Get("/{id}/content", s.handleDocumentContent)
				r.Delete("/{id}", s.handleDeleteDocument)
			})

			r.Route("/contacts", func(r chi.Router) {
				r.Get("/", s.handleListContacts)
				r.Post("/", s.handleCreateContact)
				r.Delete("/{id}", s.handleDeleteContact)
			})

			// Pilotage (P4)
			r.Get("/recurring", s.handleRecurring)
			r.Get("/cashflow", s.handleCashflow)
			r.Get("/health-score", s.handleHealthScore)
			r.Get("/analytics", s.handleAnalytics)
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
					r.Put("/space", s.handleSetTransactionSpace)
					r.Post("/split", s.handleSplitTransaction)
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
					// Détails P6 : bien immobilier / objet de valeur.
					r.Put("/property", s.handleUpsertProperty)
					r.Put("/object", s.handleUpsertObject)
					// Détails P7 : parts de société.
					r.Put("/company", s.handleUpsertCompany)
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
