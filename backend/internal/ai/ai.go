// Package ai est l'AI ROUTER d'Opale (EIA-010) : il route chaque demande
// d'explication vers le bon niveau de la cascade —
//
//	N2 homelab (Ollama, privé)  →  N3 cloud (Fable 5, données anonymisées)
//
// (le niveau N1 vit sur l'iPhone, côté app.)
//
// Règle non négociable (EIA-040/041) : l'IA n'effectue AUCUN calcul
// financier. Elle reçoit des chiffres déjà calculés par le moteur
// déterministe et se contente de les expliquer. Sans provider disponible,
// l'application reste pleinement fonctionnelle : les appelants ont toujours
// un texte de repli déterministe (EIA-020/021).
package ai

import (
	"context"
	"errors"
	"log/slog"
	"time"
)

// Tiers de la cascade.
const (
	TierHomelab = "n2" // Ollama sur le homelab — données complètes, privées
	TierCloud   = "n3" // Fable 5 — uniquement des données anonymisées
	TierNone    = ""   // aucun provider : repli déterministe
)

// ErrUnavailable : aucun niveau de la cascade ne peut traiter la demande.
var ErrUnavailable = errors.New("ai: aucun niveau IA disponible")

// Request — une demande d'explication.
type Request struct {
	// Task : étiquette courte pour la journalisation (EIA-012).
	Task string
	// System : cadrage (règles, ton). Prompt : contexte chiffré + question.
	System string
	Prompt string
	// AnonymizedPrompt : variante N2-safe du prompt (EIA-031/033). Si vide,
	// la demande est interdite de cloud (EIA-032).
	AnonymizedPrompt string
	// AllowCloud : l'utilisateur a consenti à l'envoi cloud (EIA-022).
	AllowCloud bool
	// MaxTokens : borne de la réponse (défaut raisonnable si 0).
	MaxTokens int
}

// Response — la réponse d'un niveau de la cascade.
type Response struct {
	Text     string `json:"text"`
	Tier     string `json:"tier"`     // n2 | n3
	Provider string `json:"provider"` // ollama | anthropic
	Model    string `json:"model"`
}

// Provider — un niveau de la cascade capable de générer du texte.
type Provider interface {
	Name() string
	Tier() string
	// Available répond vite (sondage mis en cache) : le routeur s'en sert
	// pour choisir un niveau sans bloquer la requête.
	Available(ctx context.Context) bool
	Generate(ctx context.Context, system, prompt string, maxTokens int) (string, error)
}

// Router décide du niveau cible pour chaque demande (EIA-010/011).
type Router struct {
	homelab Provider // peut être nil
	cloud   Provider // peut être nil
	log     *slog.Logger
}

// NewRouter assemble la cascade à partir des providers configurés.
func NewRouter(homelab, cloud Provider, log *slog.Logger) *Router {
	return &Router{homelab: homelab, cloud: cloud, log: log}
}

// HomelabAvailable indique si le niveau N2 répond (pour l'UX EIA-021).
func (r *Router) HomelabAvailable(ctx context.Context) bool {
	return r.homelab != nil && r.homelab.Available(ctx)
}

// CloudConfigured indique si le niveau N3 est configuré.
func (r *Router) CloudConfigured() bool { return r.cloud != nil }

// Explain route la demande : N2 d'abord (privé), N3 ensuite si l'appelant
// l'autorise ET qu'une variante anonymisée existe. Sinon ErrUnavailable —
// l'appelant affiche alors son repli déterministe.
func (r *Router) Explain(ctx context.Context, req Request) (Response, error) {
	if req.MaxTokens <= 0 {
		req.MaxTokens = 700
	}

	// ── N2 : homelab, données complètes, jamais anonymisées (privé). ──────
	if r.homelab != nil && r.homelab.Available(ctx) {
		start := time.Now()
		text, err := r.homelab.Generate(ctx, req.System, req.Prompt, req.MaxTokens)
		if err == nil {
			r.logRoute(req.Task, TierHomelab, "homelab disponible", start)
			return Response{Text: text, Tier: TierHomelab, Provider: r.homelab.Name()}, nil
		}
		r.log.Warn("ai: échec homelab, cascade vers le cloud", "task", req.Task, "err", err)
	}

	// ── N3 : cloud — seulement anonymisé ET consenti (EIA-022/031). ───────
	if r.cloud != nil && req.AllowCloud && req.AnonymizedPrompt != "" {
		start := time.Now()
		text, err := r.cloud.Generate(ctx, req.System, req.AnonymizedPrompt, req.MaxTokens)
		if err == nil {
			r.logRoute(req.Task, TierCloud, "homelab indisponible, cloud consenti et anonymisé", start)
			return Response{Text: text, Tier: TierCloud, Provider: r.cloud.Name()}, nil
		}
		r.log.Warn("ai: échec cloud", "task", req.Task, "err", err)
	}

	r.log.Info("ai: repli déterministe", "task", req.Task,
		"homelab", r.homelab != nil, "cloud", r.cloud != nil, "allow_cloud", req.AllowCloud)
	return Response{}, ErrUnavailable
}

// logRoute journalise la décision de routage (EIA-012).
func (r *Router) logRoute(task, tier, reason string, start time.Time) {
	r.log.Info("ai: routage", "task", task, "tier", tier, "reason", reason,
		"duration", time.Since(start).Round(time.Millisecond).String())
}
