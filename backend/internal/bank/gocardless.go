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
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
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
	err := g.callOnce(ctx, method, path, token, body, out)
	var status *APIError
	if token != "" && errors.As(err, &status) && status.Status == 401 {
		g.mu.Lock()
		if g.accessToken == token {
			g.accessToken = ""
			g.expiresAt = time.Time{}
		}
		g.mu.Unlock()
		fresh, e := g.token(ctx)
		if e != nil {
			return e
		}
		return g.callOnce(ctx, method, path, fresh, body, out)
	}
	return err
}

func (g *GoCardless) callOnce(ctx context.Context, method, path, token string, body, out any) error {
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
		retry := time.Duration(0)
		for _, header := range []string{"Retry-After", "X-Ratelimit-Account-Success-Reset", "HTTP_X_RATELIMIT_ACCOUNT_SUCCESS_RESET", "X-Ratelimit-Reset"} {
			if n, e := strconv.Atoi(resp.Header.Get(header)); e == nil && n > 0 {
				retry = max(retry, time.Duration(n)*time.Second)
			}
		}
		return &APIError{Status: resp.StatusCode, RetryAfter: retry}
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(io.LimitReader(resp.Body, 32<<20)).Decode(out)
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
		"/api/v2/institutions/?country="+url.QueryEscape(strings.ToLower(country)), token, nil, &out); err != nil {
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
	if err := g.call(ctx, http.MethodGet, "/api/v2/requisitions/"+url.PathEscape(id)+"/", token, nil, &out); err != nil {
		return Requisition{}, err
	}
	return out, nil
}

// APIError deliberately omits the bank response body: it may contain personal
// account references. Retry timing is persisted by the worker.
type APIError struct {
	Status     int
	RetryAfter time.Duration
}

func (e *APIError) Error() string {
	return fmt.Sprintf("Fournisseur bancaire indisponible (HTTP %d)", e.Status)
}

// Movement retains exact provider amount/currency and stable identifiers.
// Pending items are displayed separately; they never alter booked cash.
type Movement struct {
	Amount     money.Cents
	RawAmount  string
	Currency   string
	OccurredOn time.Time
	RawLabel   string
	SourceID   string
	Status     string
}
type providerMovement struct {
	Amount struct {
		Amount   string `json:"amount"`
		Currency string `json:"currency"`
	} `json:"transactionAmount"`
	BookingDate    string   `json:"bookingDate"`
	ValueDate      string   `json:"valueDate"`
	Label          string   `json:"remittanceInformationUnstructured"`
	Labels         []string `json:"remittanceInformationUnstructuredArray"`
	Creditor       string   `json:"creditorName"`
	Debtor         string   `json:"debtorName"`
	TransactionID  string   `json:"transactionId"`
	InternalID     string   `json:"internalTransactionId"`
	EntryReference string   `json:"entryReference"`
}

func (g *GoCardless) Transactions(ctx context.Context, account string) ([]Movement, error) {
	token, e := g.token(ctx)
	if e != nil {
		return nil, e
	}
	var out struct {
		Transactions struct {
			Booked  []providerMovement `json:"booked"`
			Pending []providerMovement `json:"pending"`
		} `json:"transactions"`
		Next *string `json:"next"`
	}
	if e = g.call(ctx, http.MethodGet, "/api/v2/accounts/"+url.PathEscape(account)+"/transactions/", token, nil, &out); e != nil {
		return nil, e
	}
	// GoCardless documents date_from/date_to, not paginated transactions. Fail
	// explicitly if the provider introduces pagination instead of truncating.
	if out.Next != nil && *out.Next != "" {
		return nil, fmt.Errorf("Pagination fournisseur inattendue : import non appliqué")
	}
	movements := []Movement{}
	for _, set := range []struct {
		status string
		items  []providerMovement
	}{{"booked", out.Transactions.Booked}, {"pending", out.Transactions.Pending}} {
		for _, t := range set.items {
			date := t.BookingDate
			if date == "" {
				date = t.ValueDate
			}
			day, e := time.Parse("2006-01-02", date)
			if e != nil && set.status == "booked" {
				return nil, fmt.Errorf("Date d’opération comptabilisée absente ou invalide")
			}
			label := strings.TrimSpace(t.Label)
			if label == "" {
				label = strings.Join(t.Labels, " ")
			}
			if label == "" {
				label = t.Creditor
			}
			if label == "" {
				label = t.Debtor
			}
			if label == "" {
				label = "Mouvement bancaire"
			}
			id := t.InternalID
			if id == "" {
				id = t.TransactionID
			}
			if id == "" {
				id = t.EntryReference
			}
			if id != "" {
				id = "gocardless:" + account + ":" + id
			}
			amount, _ := money.Parse(t.Amount.Amount)
			movements = append(movements, Movement{Amount: amount, RawAmount: t.Amount.Amount, Currency: t.Amount.Currency, OccurredOn: day, RawLabel: label, SourceID: id, Status: set.status})
		}
	}
	return movements, nil
}

// ParseMinor converts a provider decimal exactly, using the account currency's
// exponent. It rejects excess precision instead of rounding silently.
var decimalAmount = regexp.MustCompile(`^[+-]?[0-9]+(\.[0-9]+)?$`)

func ParseMinor(raw string, exponent int) (money.Cents, error) {
	if len(raw) > 80 || !decimalAmount.MatchString(strings.TrimSpace(raw)) {
		return 0, fmt.Errorf("Montant fournisseur invalide")
	}
	if exponent < 0 || exponent > 6 {
		return 0, fmt.Errorf("Précision de devise inconnue")
	}
	value, ok := new(big.Rat).SetString(strings.TrimSpace(raw))
	if !ok {
		return 0, fmt.Errorf("Montant fournisseur invalide")
	}
	scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(exponent)), nil)
	value.Mul(value, new(big.Rat).SetInt(scale))
	if !value.IsInt() || !value.Num().IsInt64() {
		return 0, fmt.Errorf("Montant fournisseur hors précision ou capacité")
	}
	return money.Cents(value.Num().Int64()), nil
}

type Balance struct {
	RawAmount      string
	Currency       string
	Date           string
	Kind           string
	CreditIncluded bool
}

type AccountDetails struct{ Currency, Name, Status string }

// AccountDetails fetches only mapping metadata. Owner names/addresses and the
// full account number are discarded; the visible reference contains four digits.
func (g *GoCardless) AccountDetails(ctx context.Context, account string) (AccountDetails, error) {
	token, e := g.token(ctx)
	if e != nil {
		return AccountDetails{}, e
	}
	var out struct {
		Account struct {
			Currency    string `json:"currency"`
			Name        string `json:"name"`
			DisplayName string `json:"displayName"`
			IBAN        string `json:"iban"`
			BBAN        string `json:"bban"`
			Status      string `json:"status"`
		} `json:"account"`
	}
	if e = g.call(ctx, http.MethodGet, "/api/v2/accounts/"+url.PathEscape(account)+"/details/", token, nil, &out); e != nil {
		return AccountDetails{}, e
	}
	name := out.Account.DisplayName
	if name == "" {
		name = out.Account.Name
	}
	if name == "" {
		name = "Compte bancaire"
	}
	ref := strings.ReplaceAll(out.Account.IBAN, " ", "")
	if ref == "" {
		ref = out.Account.BBAN
	}
	if len(ref) >= 4 {
		if out.Account.IBAN != "" {
			name = strings.ReplaceAll(name, out.Account.IBAN, "Compte bancaire")
		}
		name = strings.ReplaceAll(name, ref, "Compte bancaire") + " · •••• " + ref[len(ref)-4:]
	}
	return AccountDetails{out.Account.Currency, name, out.Account.Status}, nil
}

func (g *GoCardless) Balances(ctx context.Context, account string) ([]Balance, error) {
	token, e := g.token(ctx)
	if e != nil {
		return nil, e
	}
	var out struct {
		Balances []struct {
			Amount struct {
				Amount   string `json:"amount"`
				Currency string `json:"currency"`
			} `json:"balanceAmount"`
			Kind   string `json:"balanceType"`
			Date   string `json:"referenceDate"`
			Credit bool   `json:"creditLimitIncluded"`
		} `json:"balances"`
	}
	if e = g.call(ctx, http.MethodGet, "/api/v2/accounts/"+url.PathEscape(account)+"/balances/", token, nil, &out); e != nil {
		return nil, e
	}
	balances := []Balance{}
	for _, b := range out.Balances {
		balances = append(balances, Balance{b.Amount.Amount, b.Amount.Currency, b.Date, b.Kind, b.Credit})
	}
	return balances, nil
}
func (g *GoCardless) DeleteRequisition(ctx context.Context, id string) error {
	token, e := g.token(ctx)
	if e != nil {
		return e
	}
	return g.call(ctx, http.MethodDelete, "/api/v2/requisitions/"+url.PathEscape(id)+"/", token, nil, nil)
}
