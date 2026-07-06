package api

// La profondeur (P6) : centre immobilier (EF-033), centre investissement
// (EF-034), objets de valeur (EF-035), timeline patrimoniale (EF-045),
// coffre-fort chiffré (EF-064) et plan de transmission (EF-063).
// Tous les indicateurs sont calculés ici en arithmétique entière (ENF-007).

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"sort"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/opale-app/opale/internal/money"
	"github.com/opale-app/opale/internal/store"
	"github.com/opale-app/opale/internal/vault"
)

// ── Centre immobilier (EF-033) ────────────────────────────────────────────────

// propertyStatus — un bien + ses indicateurs déterministes.
type propertyStatus struct {
	store.Property
	// Rendement brut annuel : loyer annuel / prix d'achat (bps ; 0 si inconnu).
	GrossYieldBps int `json:"gross_yield_bps"`
	// Cashflow mensuel : loyer − charges − mensualité − taxe/12.
	MonthlyCashflow money.Cents `json:"monthly_cashflow_cents"`
	// Plus-value latente : valeur estimée − prix d'achat (0 si inconnue).
	CapitalGain money.Cents `json:"capital_gain_cents"`
	// Part réellement possédée : valeur estimée − restant dû.
	Equity money.Cents `json:"equity_cents"`
}

func (s *Server) handleRealEstate(w http.ResponseWriter, r *http.Request) {
	p := profileFromContext(r.Context())
	properties, err := s.store.ListProperties(r.Context(), p.ID)
	if err != nil {
		s.storeErr(w, err, "real estate")
		return
	}

	out := make([]propertyStatus, 0, len(properties))
	for _, prop := range properties {
		st := propertyStatus{Property: prop}
		d := prop.Details

		if d.PurchasePrice > 0 && d.MonthlyRent > 0 {
			st.GrossYieldBps = int(int64(d.MonthlyRent) * 12 * 10_000 / int64(d.PurchasePrice))
		}
		st.MonthlyCashflow = d.MonthlyRent - d.MonthlyCharges - d.MonthlyLoanPayment -
			money.Cents(int64(d.PropertyTaxYearly)/12)

		latest := money.Cents(0)
		if prop.Asset.LatestValue != nil {
			latest = *prop.Asset.LatestValue
		}
		if d.PurchasePrice > 0 && latest > 0 {
			st.CapitalGain = latest - d.PurchasePrice
		}
		st.Equity = latest
		if prop.LoanRemaining != nil {
			st.Equity = latest - *prop.LoanRemaining
		}
		out = append(out, st)
	}
	writeJSON(w, http.StatusOK, map[string]any{"properties": out})
}

type propertyRequest struct {
	PurchasePrice      int64  `json:"purchase_price_cents"`
	PurchaseDate       string `json:"purchase_date"` // yyyy-MM-dd, optionnel
	MonthlyRent        int64  `json:"monthly_rent_cents"`
	MonthlyCharges     int64  `json:"monthly_charges_cents"`
	PropertyTaxYearly  int64  `json:"property_tax_yearly_cents"`
	LiabilityID        string `json:"liability_id"`
	MonthlyLoanPayment int64  `json:"monthly_loan_payment_cents"`
}

func (s *Server) handleUpsertProperty(w http.ResponseWriter, r *http.Request) {
	p := profileFromContext(r.Context())
	var req propertyRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}
	if req.PurchasePrice < 0 || req.MonthlyRent < 0 || req.MonthlyCharges < 0 ||
		req.PropertyTaxYearly < 0 || req.MonthlyLoanPayment < 0 {
		writeError(w, http.StatusBadRequest, "invalid_body", "les montants doivent être positifs")
		return
	}

	d := store.PropertyDetails{
		AssetID:            chi.URLParam(r, "id"),
		PurchasePrice:      money.Cents(req.PurchasePrice),
		MonthlyRent:        money.Cents(req.MonthlyRent),
		MonthlyCharges:     money.Cents(req.MonthlyCharges),
		PropertyTaxYearly:  money.Cents(req.PropertyTaxYearly),
		MonthlyLoanPayment: money.Cents(req.MonthlyLoanPayment),
	}
	if req.PurchaseDate != "" {
		t, err := time.Parse(dayLayout, req.PurchaseDate)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_date", "purchase_date doit être yyyy-MM-dd")
			return
		}
		d.PurchaseDate = &t
	}
	if req.LiabilityID != "" {
		d.LiabilityID = &req.LiabilityID
	}

	if err := s.store.UpsertPropertyDetails(r.Context(), p.ID, d); err != nil {
		s.storeErr(w, err, "upsert property")
		return
	}
	writeJSON(w, http.StatusOK, d)
}

// ── Centre investissement (EF-034) ────────────────────────────────────────────

// investmentKinds : les actifs suivis comme placements.
// (company_share exclu : les parts de société ont leur centre Entreprise.)
var investmentKinds = map[string]bool{
	"pea": true, "cto": true, "life_insurance": true, "crypto": true,
}

// investmentStatus — un placement + sa performance depuis la première valo.
type investmentStatus struct {
	Asset store.Asset `json:"asset"`
	// Première valorisation connue (référence de performance).
	FirstValue money.Cents `json:"first_value_cents"`
	FirstDate  *time.Time  `json:"first_date,omitempty"`
	// Évolution : dernière − première (et en bps de la première).
	Change    money.Cents `json:"change_cents"`
	ChangeBps int         `json:"change_bps"`
	// Part du portefeuille de placements (bps).
	AllocationBps int `json:"allocation_bps"`
}

func (s *Server) handleInvestments(w http.ResponseWriter, r *http.Request) {
	p := profileFromContext(r.Context())
	assets, err := s.store.ListAssets(r.Context(), p.ID)
	if err != nil {
		s.storeErr(w, err, "investments: assets")
		return
	}
	firsts, err := s.store.FirstValuations(r.Context(), p.ID)
	if err != nil {
		s.storeErr(w, err, "investments: firsts")
		return
	}
	firstByAsset := make(map[string]store.FirstValuation, len(firsts))
	for _, f := range firsts {
		firstByAsset[f.AssetID] = f
	}

	var out []investmentStatus
	var total int64
	for _, a := range assets {
		if !investmentKinds[a.Kind] || a.Archived {
			continue
		}
		st := investmentStatus{Asset: a}
		if f, ok := firstByAsset[a.ID]; ok {
			st.FirstValue = f.Value
			asOf := f.AsOf
			st.FirstDate = &asOf
		}
		if a.LatestValue != nil {
			total += int64(*a.LatestValue)
			if st.FirstValue > 0 {
				st.Change = *a.LatestValue - st.FirstValue
				st.ChangeBps = int(int64(st.Change) * 10_000 / int64(st.FirstValue))
			}
		}
		out = append(out, st)
	}
	// Allocation en bps du total des placements.
	if total > 0 {
		for i := range out {
			if out[i].Asset.LatestValue != nil {
				out[i].AllocationBps = int(int64(*out[i].Asset.LatestValue) * 10_000 / total)
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"investments": out,
		"total_cents": total,
	})
}

// ── Objets de valeur (EF-035) ─────────────────────────────────────────────────

// objectStatus — un objet + l'écart entre valeur estimée et prix d'achat.
type objectStatus struct {
	store.ValuableObject
	Change money.Cents `json:"change_cents"`
}

func (s *Server) handleObjects(w http.ResponseWriter, r *http.Request) {
	p := profileFromContext(r.Context())
	objects, err := s.store.ListObjects(r.Context(), p.ID)
	if err != nil {
		s.storeErr(w, err, "objects")
		return
	}
	out := make([]objectStatus, 0, len(objects))
	var total int64
	for _, o := range objects {
		st := objectStatus{ValuableObject: o}
		if o.Asset.LatestValue != nil {
			total += int64(*o.Asset.LatestValue)
			if o.Details.PurchasePrice > 0 {
				st.Change = *o.Asset.LatestValue - o.Details.PurchasePrice
			}
		}
		out = append(out, st)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"objects":     out,
		"total_cents": total,
	})
}

type objectRequest struct {
	Category      string `json:"category"`
	Brand         string `json:"brand"`
	PurchasePrice int64  `json:"purchase_price_cents"`
	PurchaseDate  string `json:"purchase_date"`
	Insured       bool   `json:"insured"`
}

func (s *Server) handleUpsertObject(w http.ResponseWriter, r *http.Request) {
	p := profileFromContext(r.Context())
	var req objectRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}
	if req.PurchasePrice < 0 {
		writeError(w, http.StatusBadRequest, "invalid_body", "purchase_price_cents doit être positif")
		return
	}
	d := store.ObjectDetails{
		AssetID:       chi.URLParam(r, "id"),
		Category:      req.Category,
		Brand:         req.Brand,
		PurchasePrice: money.Cents(req.PurchasePrice),
		Insured:       req.Insured,
	}
	if req.PurchaseDate != "" {
		t, err := time.Parse(dayLayout, req.PurchaseDate)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_date", "purchase_date doit être yyyy-MM-dd")
			return
		}
		d.PurchaseDate = &t
	}
	if err := s.store.UpsertObjectDetails(r.Context(), p.ID, d); err != nil {
		s.storeErr(w, err, "upsert object")
		return
	}
	writeJSON(w, http.StatusOK, d)
}

// ── Timeline patrimoniale (EF-045) ────────────────────────────────────────────

// timelineEvent — un jalon de la vie financière, passé ou futur.
type timelineEvent struct {
	Date   string      `json:"date"` // yyyy-MM-dd
	Kind   string      `json:"kind"` // acquisition | milestone | goal | independence
	Title  string      `json:"title"`
	Detail string      `json:"detail,omitempty"`
	Amount money.Cents `json:"amount_cents,omitempty"`
	Future bool        `json:"future"`
}

// milestones : paliers de patrimoine net (en euros) façon jalons (EF-014).
var milestoneEuros = []int64{10_000, 25_000, 50_000, 100_000, 250_000, 500_000, 1_000_000}

func (s *Server) handleTimeline(w http.ResponseWriter, r *http.Request) {
	p := profileFromContext(r.Context())
	today := time.Now()
	var events []timelineEvent

	// 1. Acquisitions : première valorisation de chaque actif.
	firsts, err := s.store.FirstValuations(r.Context(), p.ID)
	if err != nil {
		s.storeErr(w, err, "timeline: firsts")
		return
	}
	for _, f := range firsts {
		title := "Entrée de « " + f.AssetName + " »"
		if f.AssetKind == "real_estate" {
			title = "Achat immobilier — " + f.AssetName
		}
		events = append(events, timelineEvent{
			Date:   f.AsOf.Format(dayLayout),
			Kind:   "acquisition",
			Title:  title,
			Amount: f.Value,
		})
	}

	// 2. Jalons de patrimoine : premier mois où chaque palier est franchi.
	if hist, err := s.store.ComputeNetWorthHistory(r.Context(), p.ID, 120); err == nil {
		reached := make(map[int64]bool)
		for _, pt := range hist.Points {
			for _, m := range milestoneEuros {
				if !reached[m] && int64(pt.Net) >= m*100 {
					reached[m] = true
					events = append(events, timelineEvent{
						Date:   pt.AsOf.Format(dayLayout),
						Kind:   "milestone",
						Title:  "Palier franchi : " + compactText(money.Cents(m*100)) + " € de patrimoine",
						Amount: money.Cents(m * 100),
					})
				}
			}
		}
	}

	// 3. Futur : objectifs datés, puis date d'indépendance projetée.
	if goals, err := s.goalStatuses(r, p.ID); err == nil {
		for _, g := range goals {
			if g.TargetDate == nil {
				continue
			}
			detail := "En avance sur le plan"
			if g.OnTrack != nil && !*g.OnTrack {
				detail = "En retard sur le plan"
			}
			events = append(events, timelineEvent{
				Date:   g.TargetDate.Format(dayLayout),
				Kind:   "goal",
				Title:  "Objectif « " + g.Name + " »",
				Detail: detail,
				Amount: g.Target,
				Future: g.TargetDate.After(today),
			})
		}
	}
	if snap, err := s.buildTwin(r, p.ID); err == nil &&
		snap.Independence.Reached && snap.Independence.Months > 0 {
		date := today.AddDate(0, snap.Independence.Months, 0)
		events = append(events, timelineEvent{
			Date:   date.Format(dayLayout),
			Kind:   "independence",
			Title:  "Indépendance financière projetée",
			Detail: "Au rythme d'épargne actuel (hypothèses de référence)",
			Amount: snap.Independence.Target,
			Future: true,
		})
	}

	sort.SliceStable(events, func(i, j int) bool { return events[i].Date < events[j].Date })
	writeJSON(w, http.StatusOK, map[string]any{"events": events})
}

// ── Coffre-fort (EF-064) ──────────────────────────────────────────────────────

// maxDocumentBytes : borne de taille d'un document (10 Mo).
const maxDocumentBytes = 10 << 20

// requireVault renvoie le coffre ou répond 503 si non configuré.
func (s *Server) requireVault(w http.ResponseWriter) *vault.Vault {
	if s.vault == nil {
		writeError(w, http.StatusServiceUnavailable, "vault_not_configured",
			"Le coffre-fort est désactivé : définis OPALE_VAULT_KEY (64 caractères hexadécimaux).")
		return nil
	}
	return s.vault
}

func (s *Server) handleListDocuments(w http.ResponseWriter, r *http.Request) {
	p := profileFromContext(r.Context())
	docs, err := s.store.ListDocuments(r.Context(), p.ID)
	if err != nil {
		s.storeErr(w, err, "list documents")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"documents":        docs,
		"vault_configured": s.vault != nil,
	})
}

type documentRequest struct {
	Name          string `json:"name"`
	Kind          string `json:"kind"`
	Mime          string `json:"mime"`
	AssetID       string `json:"asset_id"`
	ContentBase64 string `json:"content_base64"`
}

func (s *Server) handleCreateDocument(w http.ResponseWriter, r *http.Request) {
	v := s.requireVault(w)
	if v == nil {
		return
	}
	p := profileFromContext(r.Context())

	r.Body = http.MaxBytesReader(w, r.Body, maxDocumentBytes*4/3+4_096)
	var req documentRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}
	if req.Name == "" || req.ContentBase64 == "" {
		writeError(w, http.StatusBadRequest, "invalid_body", "name et content_base64 sont requis")
		return
	}
	if !store.DocumentKinds[req.Kind] {
		req.Kind = "other"
	}
	if req.Mime == "" {
		req.Mime = "application/octet-stream"
	}
	plain, err := base64.StdEncoding.DecodeString(req.ContentBase64)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_content", "content_base64 illisible")
		return
	}
	if len(plain) == 0 || len(plain) > maxDocumentBytes {
		writeError(w, http.StatusBadRequest, "invalid_content", "document vide ou > 10 Mo")
		return
	}

	sum := sha256.Sum256(plain)
	encrypted, err := v.Encrypt(plain)
	if err != nil {
		s.storeErr(w, err, "encrypt document")
		return
	}

	doc := store.Document{
		Name:      req.Name,
		Kind:      req.Kind,
		Mime:      req.Mime,
		SizeBytes: int64(len(plain)),
		SHA256:    hex.EncodeToString(sum[:]),
	}
	if req.AssetID != "" {
		doc.AssetID = &req.AssetID
	}
	created, err := s.store.CreateDocument(r.Context(), p.ID, doc, encrypted)
	if err != nil {
		s.storeErr(w, err, "create document")
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) handleDocumentContent(w http.ResponseWriter, r *http.Request) {
	v := s.requireVault(w)
	if v == nil {
		return
	}
	p := profileFromContext(r.Context())
	doc, encrypted, err := s.store.DocumentContent(r.Context(), p.ID, chi.URLParam(r, "id"))
	if err != nil {
		s.storeErr(w, err, "document content")
		return
	}
	plain, err := v.Decrypt(encrypted)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "vault_corrupted", err.Error())
		return
	}
	w.Header().Set("Content-Type", doc.Mime)
	w.Header().Set("Content-Disposition", `attachment; filename="`+doc.Name+`"`)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(plain)
}

func (s *Server) handleDeleteDocument(w http.ResponseWriter, r *http.Request) {
	p := profileFromContext(r.Context())
	if err := s.store.DeleteDocument(r.Context(), p.ID, chi.URLParam(r, "id")); err != nil {
		s.storeErr(w, err, "delete document")
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}

// ── Transmission (EF-063) ─────────────────────────────────────────────────────

type contactRequest struct {
	Name  string `json:"name"`
	Role  string `json:"role"`
	Phone string `json:"phone"`
	Email string `json:"email"`
	Note  string `json:"note"`
}

func (s *Server) handleListContacts(w http.ResponseWriter, r *http.Request) {
	p := profileFromContext(r.Context())
	contacts, err := s.store.ListContacts(r.Context(), p.ID)
	if err != nil {
		s.storeErr(w, err, "list contacts")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"contacts": contacts})
}

func (s *Server) handleCreateContact(w http.ResponseWriter, r *http.Request) {
	p := profileFromContext(r.Context())
	var req contactRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "invalid_body", "name est requis")
		return
	}
	if !store.ContactRoles[req.Role] {
		req.Role = "other"
	}
	c, err := s.store.CreateContact(r.Context(), p.ID, store.Contact{
		Name: req.Name, Role: req.Role, Phone: req.Phone, Email: req.Email, Note: req.Note,
	})
	if err != nil {
		s.storeErr(w, err, "create contact")
		return
	}
	writeJSON(w, http.StatusCreated, c)
}

func (s *Server) handleDeleteContact(w http.ResponseWriter, r *http.Request) {
	p := profileFromContext(r.Context())
	if err := s.store.DeleteContact(r.Context(), p.ID, chi.URLParam(r, "id")); err != nil {
		s.storeErr(w, err, "delete contact")
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}

// handleTransmission — le dossier « si un jour » : tout ce qu'un proche doit
// savoir, assemblé depuis les données existantes (EF-063).
func (s *Server) handleTransmission(w http.ResponseWriter, r *http.Request) {
	p := profileFromContext(r.Context())

	contacts, err := s.store.ListContacts(r.Context(), p.ID)
	if err != nil {
		s.storeErr(w, err, "transmission: contacts")
		return
	}
	assets, err := s.store.ListAssets(r.Context(), p.ID)
	if err != nil {
		s.storeErr(w, err, "transmission: assets")
		return
	}
	liabilities, err := s.store.ListLiabilities(r.Context(), p.ID)
	if err != nil {
		s.storeErr(w, err, "transmission: liabilities")
		return
	}
	docs, err := s.store.ListDocuments(r.Context(), p.ID)
	if err != nil {
		s.storeErr(w, err, "transmission: documents")
		return
	}
	nw, err := s.store.ComputeNetWorth(r.Context(), p.ID)
	if err != nil {
		s.storeErr(w, err, "transmission: net worth")
		return
	}

	// Nombre de documents rattachés par actif.
	docsByAsset := map[string]int{}
	for _, d := range docs {
		if d.AssetID != nil {
			docsByAsset[*d.AssetID]++
		}
	}
	type transmissionAsset struct {
		store.Asset
		DocumentCount int `json:"document_count"`
	}
	outAssets := make([]transmissionAsset, 0, len(assets))
	for _, a := range assets {
		if a.Archived {
			continue
		}
		outAssets = append(outAssets, transmissionAsset{Asset: a, DocumentCount: docsByAsset[a.ID]})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"net_worth":      nw,
		"contacts":       contacts,
		"assets":         outAssets,
		"liabilities":    liabilities,
		"document_count": len(docs),
	})
}
