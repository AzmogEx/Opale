package ai

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// Anthropic — niveau N3 facultatif, avec modèle configurable côté serveur.
//
// GARDE-FOU : ce provider ne reçoit JAMAIS de données brutes — le routeur
// ne lui transmet que les agrégats minimisés (EIA-031/033).
type Anthropic struct {
	client anthropic.Client
	model  anthropic.Model
}

// ErrRefused : la requête a été déclinée par les classificateurs de sûreté
// (y compris par le modèle de repli).
var ErrRefused = errors.New("ai: requête refusée par le modèle cloud")

// NewAnthropic construit le provider N3.
func NewAnthropic(apiKey string, options ...option.RequestOption) *Anthropic {
	return NewAnthropicWithModel(apiKey, "claude-sonnet-5-5", options...)
}
func NewAnthropicWithModel(apiKey, model string, options ...option.RequestOption) *Anthropic {
	if model == "" {
		model = "claude-sonnet-5-5"
	}
	options = append([]option.RequestOption{option.WithAPIKey(apiKey), option.WithMaxRetries(1), option.WithRequestTimeout(25 * time.Second)}, options...)
	return &Anthropic{
		client: anthropic.NewClient(options...),
		model:  anthropic.Model(model),
	}
}

func (a *Anthropic) Name() string { return "anthropic" }
func (a *Anthropic) Tier() string { return TierCloud }

// Available : le niveau cloud est « disponible » dès qu'il est configuré ;
// les erreurs réseau sont gérées à l'appel (pas de sondage payant).
func (a *Anthropic) Available(context.Context) bool { return true }

// Generate appelle le modèle configuré, sans dépendance à une API bêta
// propre à un modèle. Une erreur ou un refus déclenche le repli du routeur.
func (a *Anthropic) Generate(ctx context.Context, system, prompt string, maxTokens int) (string, error) {
	resp, err := a.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     a.model,
		MaxTokens: int64(maxTokens),
		System:    []anthropic.TextBlockParam{{Text: system}},
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},
	})
	if err != nil {
		return "", err
	}

	// Toujours vérifier le stop_reason avant de lire le contenu :
	// un refus arrive en HTTP 200 avec un contenu vide ou partiel.
	if string(resp.StopReason) == "refusal" {
		return "", ErrRefused
	}

	var b strings.Builder
	for _, block := range resp.Content {
		if t, ok := block.AsAny().(anthropic.TextBlock); ok {
			b.WriteString(t.Text)
		}
	}
	text := strings.TrimSpace(b.String())
	if text == "" {
		return "", errors.New("ai: réponse cloud vide")
	}
	return text, nil
}
