// Package bank intègre GoCardless Bank Account Data (DSP2) — EF-071.
//
// Optionnel comme le reste : sans OPALE_GC_SECRET_ID/KEY, la synchro
// bancaire est simplement désactivée et l'import CSV (EF-070) reste la voie
// normale. Aucun identifiant bancaire ne transite par Opale : l'utilisateur
// s'authentifie chez SA banque via le lien GoCardless (redirection DSP2).
package bank

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/opale-app/opale/internal/money"
)

// DefaultBaseURL — l'API GoCardless Bank Account Data.
const DefaultBaseURL = "https://bankaccountdata.gocardless.com"

// GoCardless — client minimal (jeton mis en cache, renouvelé à l'expiration).
type GoCardless struct {
	baseURL   string
	secretID  string
	secretKey string
	client    *http.Client

	mu          sync.Mutex
	accessToken string
	expiresAt   time.Time
}

// New construit le client. baseURL vide = API officielle.
func New(baseURL, secretID, secretKey string) *GoCardless {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	return &GoCardless{
		baseURL:   strings.TrimRight(baseURL, "/"),
		secretID:  secretID,
		secretKey: secretKey,
		client:    &http.Client{Timeout: 30 * time.Second},
	}
}

// token renvoie un jeton d'accès valide (POST /token/new/ si besoin).
func (g *GoCardless) token(ctx context.Context) (string, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.accessToken != "" && time.Now().Before(g.expiresAt.Add(-1*time.Minute)) {
		return g.accessToken, nil
	}

	var out struct {
		Access        string `json:"access"`
		AccessExpires int    `json:"access_expires"`
	}
	if err := g.call(ctx, http.MethodPost, "/api/v2/token/new/", "",
		map[string]string{"secret_id": g.secretID, "secret_key": g.secretKey}, &out); err != nil {
		return "", fmt.Errorf("bank: jeton GoCardless : %w", err)
	}
	g.accessToken = out.Access
	g.expiresAt = time.Now().Add(time.Duration(out.AccessExpires) * time.Second)
	return g.accessToken, nil
}

// call — appel JSON générique (auth Bearer si token non vide).
func (g *GoCardless) call(ctx context.Context, method, path, token string, body, out any) error {
	var reader *bytes.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(raw)
	} else {
		reader = bytes.NewReader(nil)
	}
	req, err := http.NewRequestWithContext(ctx, method, g.baseURL+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := g.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("gocardless: %s %s → statut %d", method, path, resp.StatusCode)
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// Institution — une banque proposée par GoCardless.
type Institution struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Logo string `json:"logo"`
}

// Institutions liste les banques d'un pays (ex. « fr »).
func (g *GoCardless) Institutions(ctx context.Context, country string) ([]Institution, error) {
	token, err := g.token(ctx)
	if err != nil {
		return nil, err
	}
	var out []Institution
	if err := g.call(ctx, http.MethodGet,
		"/api/v2/institutions/?country="+strings.ToLower(country), token, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// Requisition — une demande d'accès DSP2 (l'utilisateur consent chez sa banque).
type Requisition struct {
	ID       string   `json:"id"`
	Link     string   `json:"link"`
	Status   string   `json:"status"` // CR (créée) → LN (liée)
	Accounts []string `json:"accounts"`
}

// CreateRequisition ouvre la demande d'accès et renvoie le lien de consentement.
func (g *GoCardless) CreateRequisition(ctx context.Context, institutionID, redirect string) (Requisition, error) {
	token, err := g.token(ctx)
	if err != nil {
		return Requisition{}, err
	}
	var out Requisition
	if err := g.call(ctx, http.MethodPost, "/api/v2/requisitions/", token, map[string]string{
		"institution_id": institutionID,
		"redirect":       redirect,
	}, &out); err != nil {
		return Requisition{}, err
	}
	return out, nil
}

// GetRequisition relit l'état d'une réquisition (et ses comptes une fois liée).
func (g *GoCardless) GetRequisition(ctx context.Context, id string) (Requisition, error) {
	token, err := g.token(ctx)
	if err != nil {
		return Requisition{}, err
	}
	var out Requisition
	if err := g.call(ctx, http.MethodGet, "/api/v2/requisitions/"+id+"/", token, nil, &out); err != nil {
		return Requisition{}, err
	}
	return out, nil
}

// Movement — un mouvement bancaire normalisé (centimes, libellé brut).
type Movement struct {
	Amount     money.Cents
	OccurredOn time.Time
	RawLabel   string
}

// Transactions récupère les mouvements comptabilisés d'un compte.
func (g *GoCardless) Transactions(ctx context.Context, accountID string) ([]Movement, error) {
	token, err := g.token(ctx)
	if err != nil {
		return nil, err
	}
	var out struct {
		Transactions struct {
			Booked []struct {
				TransactionAmount struct {
					Amount string `json:"amount"`
				} `json:"transactionAmount"`
				BookingDate    string `json:"bookingDate"`
				Remittance     string `json:"remittanceInformationUnstructured"`
				CreditorName   string `json:"creditorName"`
				DebtorName     string `json:"debtorName"`
			} `json:"booked"`
		} `json:"transactions"`
	}
	if err := g.call(ctx, http.MethodGet,
		"/api/v2/accounts/"+accountID+"/transactions/", token, nil, &out); err != nil {
		return nil, err
	}

	movements := make([]Movement, 0, len(out.Transactions.Booked))
	for _, t := range out.Transactions.Booked {
		amount, err := money.Parse(t.TransactionAmount.Amount)
		if err != nil {
			return nil, fmt.Errorf("bank: montant illisible %q : %w", t.TransactionAmount.Amount, err)
		}
		day, err := time.Parse("2006-01-02", t.BookingDate)
		if err != nil {
			return nil, fmt.Errorf("bank: date illisible %q : %w", t.BookingDate, err)
		}
		label := strings.TrimSpace(t.Remittance)
		if label == "" {
			label = strings.TrimSpace(t.CreditorName)
		}
		if label == "" {
			label = strings.TrimSpace(t.DebtorName)
		}
		if label == "" {
			label = "Mouvement bancaire"
		}
		movements = append(movements, Movement{Amount: amount, OccurredOn: day, RawLabel: label})
	}
	return movements, nil
}
