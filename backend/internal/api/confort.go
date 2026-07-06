package api

// Le confort (P7) : module entrepreneur (EF-036) et synchro bancaire
// GoCardless (EF-071). La banque est optionnelle et cloisonnée : sans
// secrets configurés, l'import CSV (EF-070) reste la voie normale.

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/opale-app/opale/internal/categorize"
	"github.com/opale-app/opale/internal/money"
	"github.com/opale-app/opale/internal/store"
)

// ── Module entrepreneur (EF-036) ──────────────────────────────────────────────

// PFU : prélèvement forfaitaire unique sur les dividendes (30 %).
const pfuBps = 3_000

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
	// Dividendes annuels nets après PFU 30 % (indicatif).
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
			myShare := int64(*c.Asset.LatestValue)
			st.CompanyValue = money.Cents(myShare * 10_000 / int64(c.Details.OwnershipBps))
			st.MyTotal = money.Cents(myShare) + c.Details.CCA
		} else {
			st.MyTotal = c.Details.CCA
		}
		st.DividendsNet = money.Cents(int64(c.Details.AnnualDividends) * (10_000 - pfuBps) / 10_000)
		out = append(out, st)
	}
	writeJSON(w, http.StatusOK, map[string]any{"companies": out})
}

type companyRequest struct {
	SIREN           string `json:"siren"`
	OwnershipBps    int    `json:"ownership_bps"`
	CCA             int64  `json:"cca_cents"`
	AnnualDividends int64  `json:"annual_dividends_cents"`
	MonthlySalary   int64  `json:"monthly_salary_cents"`
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
	AssetID         string `json:"asset_id"`  // compte Opale qui recevra les mouvements
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

	links, err := s.store.ListBankLinks(r.Context(), p.ID)
	if err != nil {
		s.storeErr(w, err, "bank sync: links")
		return
	}
	if len(links) == 0 {
		writeError(w, http.StatusBadRequest, "no_bank_linked", "Aucune banque connectée.")
		return
	}

	rules, err := s.store.MerchantRuleNames(r.Context(), p.ID)
	if err != nil {
		s.storeErr(w, err, "bank sync: rules")
		return
	}
	byName, err := s.categoriesByName(r, p.ID)
	if err != nil {
		s.storeErr(w, err, "bank sync: categories")
		return
	}

	type syncResult struct {
		LinkID      string `json:"link_id"`
		Institution string `json:"institution"`
		Status      string `json:"status"` // synced | pending_consent | error
		Imported    int    `json:"imported"`
		Duplicates  int    `json:"duplicates"`
		Error       string `json:"error,omitempty"`
	}
	results := make([]syncResult, 0, len(links))

	for _, link := range links {
		res := syncResult{LinkID: link.ID, Institution: link.InstitutionName}

		requisition, err := s.bank.GetRequisition(r.Context(), link.RequisitionID)
		if err != nil {
			res.Status, res.Error = "error", err.Error()
			results = append(results, res)
			continue
		}
		if len(requisition.Accounts) == 0 {
			// L'utilisateur n'a pas (encore) donné son consentement.
			res.Status = "pending_consent"
			results = append(results, res)
			continue
		}

		var prepared []store.NewTransaction
		for _, accountID := range requisition.Accounts {
			movements, err := s.bank.Transactions(r.Context(), accountID)
			if err != nil {
				res.Status, res.Error = "error", err.Error()
				break
			}
			for _, m := range movements {
				prepared = append(prepared, store.NewTransaction{
					AssetID:     link.AssetID,
					Amount:      m.Amount,
					OccurredOn:  m.OccurredOn,
					Label:       categorize.CleanLabel(m.RawLabel),
					RawLabel:    m.RawLabel,
					MerchantKey: categorize.MerchantKey(m.RawLabel),
					CategoryID:  s.suggestCategoryID(r, p.ID, m.RawLabel, m.Amount, rules, byName),
				})
			}
		}
		if res.Status == "error" {
			results = append(results, res)
			continue
		}

		imported, err := s.store.ImportTransactions(r.Context(), p.ID, prepared)
		if err != nil {
			res.Status, res.Error = "error", err.Error()
			results = append(results, res)
			continue
		}
		res.Status = "synced"
		res.Imported = imported.Imported
		res.Duplicates = imported.Duplicates
		_ = s.store.MarkBankLinkSynced(r.Context(), p.ID, link.ID)
		results = append(results, res)
	}

	writeJSON(w, http.StatusOK, map[string]any{"results": results})
}

func (s *Server) handleBankDisconnect(w http.ResponseWriter, r *http.Request) {
	p := profileFromContext(r.Context())
	if err := s.store.DeleteBankLink(r.Context(), p.ID, chi.URLParam(r, "id")); err != nil {
		s.storeErr(w, err, "bank disconnect")
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}
