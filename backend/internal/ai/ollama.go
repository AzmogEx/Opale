package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// Ollama — niveau N2 de la cascade (EIA-002) : un serveur Ollama sur le
// homelab (RTX 5080). Les données y restent privées : aucun besoin
// d'anonymisation à ce niveau.
type Ollama struct {
	baseURL string
	model   string
	client  *http.Client

	// Sondage de disponibilité mis en cache (le routeur l'appelle à chaque
	// requête ; on ne re-sonde le serveur qu'à intervalle raisonnable).
	mu          sync.Mutex
	lastProbe   time.Time
	lastHealthy bool
}

// NewOllama construit le provider N2. baseURL ex. « http://homelab:11434 ».
func NewOllama(baseURL, model string) *Ollama {
	return &Ollama{
		baseURL: strings.TrimRight(baseURL, "/"),
		model:   model,
		client:  &http.Client{Timeout: 90 * time.Second},
	}
}

func (o *Ollama) Name() string { return "ollama" }
func (o *Ollama) Tier() string { return TierHomelab }

// Available sonde GET /api/tags avec un délai court, résultat gardé 30 s.
func (o *Ollama) Available(ctx context.Context) bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	if time.Since(o.lastProbe) < 30*time.Second {
		return o.lastHealthy
	}
	o.lastProbe = time.Now()

	probeCtx, cancel := context.WithTimeout(ctx, 1500*time.Millisecond)
	defer cancel()
	req, err := http.NewRequestWithContext(probeCtx, http.MethodGet, o.baseURL+"/api/tags", nil)
	if err != nil {
		o.lastHealthy = false
		return false
	}
	resp, err := o.client.Do(req)
	if err != nil {
		o.lastHealthy = false
		return false
	}
	resp.Body.Close()
	o.lastHealthy = resp.StatusCode == http.StatusOK
	return o.lastHealthy
}

// Generate appelle POST /api/chat (réponse non streamée).
func (o *Ollama) Generate(ctx context.Context, system, prompt string, maxTokens int) (string, error) {
	body, err := json.Marshal(map[string]any{
		"model":  o.model,
		"stream": false,
		"messages": []map[string]string{
			{"role": "system", "content": system},
			{"role": "user", "content": prompt},
		},
		"options": map[string]any{
			"num_predict": maxTokens,
		},
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, o.baseURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := o.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ollama: statut %d", resp.StatusCode)
	}

	var out struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	text := strings.TrimSpace(out.Message.Content)
	if text == "" {
		return "", fmt.Errorf("ollama: réponse vide")
	}
	return text, nil
}
