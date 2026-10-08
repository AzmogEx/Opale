package api

// Le confort (P7) : module entrepreneur (EF-036) et synchro bancaire
// GoCardless (EF-071). La banque est optionnelle et cloisonnée : sans
// secrets configurés, l'import CSV (EF-070) reste la voie normale.

import (
	"net/http"

	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/opale-app/opale/internal/bank"
	"github.com/opale-app/opale/internal/money"
	"github.com/opale-app/opale/internal/store"
)

// ── Module entrepreneur (EF-036) ──────────────────────────────────────────────

// PFU 2026 : 12,8 % IR + 18,6 % prélèvements sociaux ; cas général résident.
const pfuBps = 3_140

// companyStatus — une société + les indicateurs dérivés.
//
// Sémantique : la valorisation de l'actif = la valeur de MA part (comme tout
// autre actif — c'est elle qui entre dans le patrimoine net). La valeur de la
// société entière s'en déduit via les parts détenues.
type companyStatus struct {
	store.Company
	// Valeur estimée de la société entière : ma part × 10000 / parts (bps).
	CompanyValue money.Cents `json:"company_value_cents"`
	// Ma part + compte courant d'associé (ce que je récupérerais).
	MyTotal money.Cents `json:"my_total_cents"`
	// Dividendes annuels nets après PFU 31,4 % (indicatif).
	DividendsNet money.Cents `json:"dividends_net_cents"`
}

func (s *Server) handleCompanies(w http.ResponseWriter, r *http.Request) {
	p := profileFromContext(r.Context())
	companies, err := s.store.ListCompanies(r.Context(), p.ID)
	if err != nil {
		s.storeErr(w, err, "companies")
		return
	}
	out := make([]companyStatus, 0, len(companies))
	for _, c := range companies {
		st := companyStatus{Company: c}
		if c.Asset.LatestValue != nil {
			myShare := *c.Asset.LatestValue
			st.CompanyValue, err = scaleCents(myShare, 10000, int64(c.Details.OwnershipBps))
			if err != nil {
				s.storeErr(w, err, "company valuation")
				return
			}
			st.MyTotal, err = money.Add(myShare, c.Details.CCA)
			if err != nil {
				s.storeErr(w, err, "company total")
				return
			}
		} else {
			st.MyTotal = c.Details.CCA
		}
		st.DividendsNet, err = scaleCents(c.Details.AnnualDividends, 10000-pfuBps, 10000)
		if err != nil {
			s.storeErr(w, err, "company dividends")
			return
		}
		out = append(out, st)
	}
	writeJSON(w, http.StatusOK, map[string]any{"companies": out})
}

type companyRequest struct {
	SIREN           string  `json:"siren"`
	OwnershipBps    int     `json:"ownership_bps"`
	CCA             int64   `json:"cca_cents"`
	CCAAssetID      *string `json:"cca_asset_id"`
	AnnualDividends int64   `json:"annual_dividends_cents"`
	MonthlySalary   int64   `json:"monthly_salary_cents"`
}

func (s *Server) handleUpsertCompany(w http.ResponseWriter, r *http.Request) {
	p := profileFromContext(r.Context())
	var req companyRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}
	if req.OwnershipBps <= 0 || req.OwnershipBps > 10_000 {
		writeError(w, http.StatusBadRequest, "invalid_body",
			"ownership_bps doit être entre 1 et 10000 (100 %)")
		return
	}
	if req.CCA < 0 || req.AnnualDividends < 0 || req.MonthlySalary < 0 {
		writeError(w, http.StatusBadRequest, "invalid_body", "les montants doivent être positifs")
		return
	}
	d := store.CompanyDetails{
		AssetID:         chi.URLParam(r, "id"),
		SIREN:           req.SIREN,
		OwnershipBps:    req.OwnershipBps,
		CCA:             money.Cents(req.CCA),
		CCAAssetID:      req.CCAAssetID,
		AnnualDividends: money.Cents(req.AnnualDividends),
		MonthlySalary:   money.Cents(req.MonthlySalary),
	}
	if err := s.store.UpsertCompanyDetails(r.Context(), p.ID, d); err != nil {
		s.storeErr(w, err, "upsert company")
		return
	}
	writeJSON(w, http.StatusOK, d)
}

// ── Synchro bancaire GoCardless (EF-071) ──────────────────────────────────────

// requireBank répond 503 si la synchro bancaire n'est pas configurée.
func (s *Server) requireBank(w http.ResponseWriter) bool {
	if s.bank == nil {
		writeError(w, http.StatusServiceUnavailable, "bank_not_configured",
			"Synchro bancaire désactivée : définis OPALE_GC_SECRET_ID et OPALE_GC_SECRET_KEY (GoCardless).")
		return false
	}
	return true
}

func (s *Server) handleBankStatus(w http.ResponseWriter, r *http.Request) {
	p := profileFromContext(r.Context())
	links, err := s.store.ListBankLinks(r.Context(), p.ID)
	if err != nil {
		s.storeErr(w, err, "bank status")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"configured": s.bank != nil,
		"links":      links,
	})
}

func (s *Server) handleBankInstitutions(w http.ResponseWriter, r *http.Request) {
	if !s.requireBank(w) {
		return
	}
	country := r.URL.Query().Get("country")
	if country == "" {
		country = "fr"
	}
	institutions, err := s.bank.Institutions(r.Context(), country)
	if err != nil {
		writeError(w, http.StatusBadGateway, "bank_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"institutions": institutions})
}

type bankConnectRequest struct {
	InstitutionID   string `json:"institution_id"`
	InstitutionName string `json:"institution_name"`
	AssetID         string `json:"asset_id"` // compte Opale qui recevra les mouvements
	Redirect        string `json:"redirect"` // où revenir après le consentement
}

func (s *Server) handleBankConnect(w http.ResponseWriter, r *http.Request) {
	if !s.requireBank(w) {
		return
	}
	p := profileFromContext(r.Context())
	var req bankConnectRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}
	if req.InstitutionID == "" || req.AssetID == "" {
		writeError(w, http.StatusBadRequest, "invalid_body", "institution_id et asset_id sont requis")
		return
	}
	if req.Redirect == "" {
		req.Redirect = "opale://bank-linked"
	}
	// L'actif cible doit appartenir au profil.
	if _, err := s.store.GetAsset(r.Context(), p.ID, req.AssetID); err != nil {
		s.storeErr(w, err, "bank connect: asset")
		return
	}

	requisition, err := s.bank.CreateRequisition(r.Context(), req.InstitutionID, req.Redirect)
	if err != nil {
		writeError(w, http.StatusBadGateway, "bank_error", err.Error())
		return
	}
	link, err := s.store.CreateBankLink(r.Context(), p.ID, store.BankLink{
		AssetID:         req.AssetID,
		RequisitionID:   requisition.ID,
		InstitutionID:   req.InstitutionID,
		InstitutionName: req.InstitutionName,
	})
	if err != nil {
		s.storeErr(w, err, "bank connect: link")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"link":         link,
		"consent_link": requisition.Link,
	})
}

// handleBankSync tire les mouvements de toutes les banques liées et les
// importe avec la même chaîne que le CSV : nettoyage + catégorisation +
// déduplication (EF-070/022).
func (s *Server) handleBankSync(w http.ResponseWriter, r *http.Request) {
	if !s.requireBank(w) {
		return
	}
	p := profileFromContext(r.Context())
	links, e := s.store.ListBankLinks(r.Context(), p.ID)
	if e != nil {
		s.storeErr(w, e, "bank links")
		return
	}
	results := []bankSyncResult{}
	for _, link := range links {
		results = append(results, s.syncBankLink(r.Context(), p.ID, link))
	}
	writeJSON(w, 200, map[string]any{"results": results})
}

func (s *Server) handleBankDisconnect(w http.ResponseWriter, r *http.Request) {
	if !s.requireBank(w) {
		return
	}
	p := profileFromContext(r.Context())
	id := chi.URLParam(r, "id")
	link, e := s.store.BankLink(r.Context(), p.ID, id)
	if e != nil {
		s.storeErr(w, e, "bank disconnect")
		return
	}
	if e = s.bank.DeleteRequisition(r.Context(), link.RequisitionID); e != nil {
		var provider *bank.APIError
		if !errors.As(e, &provider) || provider.Status != 404 {
			writeError(w, 502, "revoke_failed", "La révocation bancaire n’a pas abouti, réessaie avant de supprimer le lien")
			return
		}
	}
	if e = s.store.DeleteBankLink(r.Context(), p.ID, id); e != nil {
		s.storeErr(w, e, "delete bank link")
		return
	}
	s.journal(r, &p.ID, "bank_revoked", id)
	writeJSON(w, 204, nil)
}
