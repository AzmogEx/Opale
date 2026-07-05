package ai

import (
	"context"
	"errors"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// Anthropic — niveau N3 de la cascade (EIA-003) : Claude Fable 5, réservé
// aux demandes complexes, avec repli serveur vers Opus 4.8 en cas de refus.
//
// GARDE-FOU : ce provider ne reçoit JAMAIS de données brutes — le routeur
// ne lui transmet que le prompt anonymisé (EIA-031/033).
type Anthropic struct {
	client anthropic.Client
	model  anthropic.Model
}

// ErrRefused : la requête a été déclinée par les classificateurs de sûreté
// (y compris par le modèle de repli).
var ErrRefused = errors.New("ai: requête refusée par le modèle cloud")

// NewAnthropic construit le provider N3.
func NewAnthropic(apiKey string) *Anthropic {
	return &Anthropic{
		client: anthropic.NewClient(option.WithAPIKey(apiKey)),
		model:  anthropic.ModelClaudeFable5,
	}
}

func (a *Anthropic) Name() string { return "anthropic" }
func (a *Anthropic) Tier() string { return TierCloud }

// Available : le niveau cloud est « disponible » dès qu'il est configuré ;
// les erreurs réseau sont gérées à l'appel (pas de sondage payant).
func (a *Anthropic) Available(context.Context) bool { return true }

// Generate appelle Fable 5. Le thinking est toujours actif sur ce modèle
// (on omet donc le paramètre), et le repli serveur vers Opus 4.8 est activé
// par défaut pour que les faux positifs de refus ne cassent pas la réponse.
func (a *Anthropic) Generate(ctx context.Context, system, prompt string, maxTokens int) (string, error) {
	resp, err := a.client.Beta.Messages.New(ctx, anthropic.BetaMessageNewParams{
		Model:     a.model,
		MaxTokens: int64(maxTokens),
		Betas:     []anthropic.AnthropicBeta{anthropic.AnthropicBetaServerSideFallback2026_06_01},
		Fallbacks: []anthropic.BetaFallbackParam{{Model: anthropic.ModelClaudeOpus4_8}},
		System:    []anthropic.BetaTextBlockParam{{Text: system}},
		Messages: []anthropic.BetaMessageParam{
			anthropic.NewBetaUserMessage(anthropic.NewBetaTextBlock(prompt)),
		},
	})
	if err != nil {
		return "", err
	}

	// Toujours vérifier le stop_reason avant de lire le contenu :
	// un refus arrive en HTTP 200 avec un contenu vide ou partiel.
	if resp.StopReason == anthropic.BetaStopReasonRefusal {
		return "", ErrRefused
	}

	var b strings.Builder
	for _, block := range resp.Content {
		if t, ok := block.AsAny().(anthropic.BetaTextBlock); ok {
			b.WriteString(t.Text)
		}
	}
	text := strings.TrimSpace(b.String())
	if text == "" {
		return "", errors.New("ai: réponse cloud vide")
	}
	return text, nil
}
