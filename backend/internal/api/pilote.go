package api

// Le pilote automatique (P8) : cours automatiques, snapshots mensuels,
// allocation cible, alertes personnalisées, module fiscal, simulateur de
// crédit, bilan annuel « Wrapped », profil démo et enregistrement push.

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/opale-app/opale/internal/categorize"
	"github.com/opale-app/opale/internal/engine"
	"github.com/opale-app/opale/internal/money"
	"github.com/opale-app/opale/internal/store"
	"github.com/opale-app/opale/internal/tax"
)

// ── Cours automatiques ────────────────────────────────────────────────────────

// handleQuotesRefresh force un cycle de rafraîchissement (BCE + CoinGecko).
func (s *Server) handleQuotesRefresh(w http.ResponseWriter, r *http.Request) {
	if !s.cfg.AutoQuotes || s.jobs == nil {
		writeError(w, http.StatusServiceUnavailable, "quotes_disabled",
			"cours automatiques désactivés — OPALE_AUTO_QUOTES=on pour les activer")
		return
	}
	if err := s.jobs.RefreshQuotesForProfile(r.Context(), profileFromContext(r.Context()).ID); err != nil {
		s.log.Warn("refresh quotes", "err", err)
		writeError(w, http.StatusBadGateway, "quotes_failed", "sources de cours injoignables : "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"refreshed": true})
}

// handleSetAssetQuote active/coupe le suivi automatique d'un actif crypto.
func (s *Server) handleSetAssetQuote(w http.ResponseWriter, r *http.Request) {
	p := profileFromContext(r.Context())
	var req struct {
		Symbol        string `json:"symbol"`         // identifiant CoinGecko (« bitcoin »)
		QuantityMicro int64  `json:"quantity_micro"` // 0,5 BTC = 500000
	}
	if err := decodeJSON(r, &req); err != nil || req.QuantityMicro < 0 {
		writeError(w, http.StatusBadRequest, "invalid_body",
			"symbol (identifiant CoinGecko) et quantity_micro (≥ 0) sont requis")
		return
	}
	req.Symbol = strings.ToLower(strings.TrimSpace(req.Symbol))
	if err := s.store.SetAssetQuote(r.Context(), p.ID, chi.URLParam(r, "id"), req.Symbol, req.QuantityMicro); err != nil {
		s.storeErr(w, err, "set asset quote")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"symbol": req.Symbol, "quantity_micro": req.QuantityMicro})
}

// handleSnapshots renvoie les snapshots mensuels pris par le serveur.
func (s *Server) handleSnapshots(w http.ResponseWriter, r *http.Request) {
	p := profileFromContext(r.Context())
	limit := 60
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 && n <= 240 {
			limit = n
		}
	}
	snaps, err := s.store.ListMonthlySnapshots(r.Context(), p.ID, limit)
	if err != nil {
		s.storeErr(w, err, "list snapshots")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"snapshots": snaps})
}

// ── Allocation cible ──────────────────────────────────────────────────────────

// assetClass regroupe les kinds en 5 classes de portefeuille.
func assetClass(kind string) string {
	switch kind {
	case "pea", "cto", "life_insurance":
		return "stocks"
	case "real_estate":
		return "real_estate"
	case "crypto":
		return "crypto"
	case "checking", "savings":
		return "cash"
	default:
		return "other"
	}
}

var allocationClasses = []string{"stocks", "real_estate", "crypto", "cash", "other"}

// handleAllocation compare la répartition réelle du portefeuille à la cible.
func (s *Server) handleAllocation(w http.ResponseWriter, r *http.Request) {
	p := profileFromContext(r.Context())
	assets, err := s.store.ListAssets(r.Context(), p.ID)
	if err != nil {
		s.storeErr(w, err, "allocation assets")
		return
	}
	targets, err := s.store.ListAllocationTargets(r.Context(), p.ID)
	if err != nil {
		s.storeErr(w, err, "allocation targets")
		return
	}
	targetByClass := map[string]int{}
	for _, t := range targets {
		targetByClass[t.Class] = t.TargetBps
	}

	// Répartition réelle sur les dernières valorisations.
	actual := map[string]int64{}
	var total int64
	for _, a := range assets {
		if a.Archived || a.LatestValue == nil {
			continue
		}
		actual[assetClass(a.Kind)] += int64(*a.LatestValue)
		total += int64(*a.LatestValue)
	}

	type line struct {
		Class       string      `json:"class"`
		ActualCents money.Cents `json:"actual_cents"`
		ActualBps   int         `json:"actual_bps"`
		TargetBps   int         `json:"target_bps"`
		// Dérive : ce qu'il faudrait déplacer VERS cette classe (négatif =
		// en excès) pour retrouver la cible, en centimes.
		DriftCents money.Cents `json:"drift_cents"`
	}
	lines := []line{}
	for _, class := range allocationClasses {
		l := line{Class: class, ActualCents: money.Cents(actual[class]), TargetBps: targetByClass[class]}
		if total > 0 {
			l.ActualBps = int(actual[class] * 10_000 / total)
			targetCents := total * int64(l.TargetBps) / 10_000
			l.DriftCents = money.Cents(targetCents - actual[class])
		}
		lines = append(lines, l)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"total_cents": money.Cents(total),
		"lines":       lines,
		"has_targets": len(targets) > 0,
	})
}

// handleSetAllocation remplace les cibles (somme ≤ 10 000 bps).
func (s *Server) handleSetAllocation(w http.ResponseWriter, r *http.Request) {
	p := profileFromContext(r.Context())
	var req struct {
		Targets []store.AllocationTarget `json:"targets"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "targets est requis")
		return
	}
	valid := map[string]bool{}
	for _, c := range allocationClasses {
		valid[c] = true
	}
	sum := 0
	for _, t := range req.Targets {
		if !valid[t.Class] || t.TargetBps < 0 || t.TargetBps > 10_000 {
			writeError(w, http.StatusBadRequest, "invalid_target",
				"classe inconnue ou cible hors [0, 10000] bps : "+t.Class)
			return
		}
		sum += t.TargetBps
	}
	if sum > 10_000 {
		writeError(w, http.StatusBadRequest, "invalid_sum", "la somme des cibles dépasse 100 %")
		return
	}
	if err := s.store.SetAllocationTargets(r.Context(), p.ID, req.Targets); err != nil {
		s.storeErr(w, err, "set allocation")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"saved": len(req.Targets)})
}

// ── Alertes personnalisées ────────────────────────────────────────────────────

var customAlertKinds = map[string]bool{
	"cash_below": true, "net_worth_below": true, "expenses_month_above": true,
}

func (s *Server) handleListCustomAlerts(w http.ResponseWriter, r *http.Request) {
	p := profileFromContext(r.Context())
	alerts, err := s.store.ListCustomAlerts(r.Context(), p.ID, false)
	if err != nil {
		s.storeErr(w, err, "list custom alerts")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"alerts": alerts})
}

func (s *Server) handleCreateCustomAlert(w http.ResponseWriter, r *http.Request) {
	p := profileFromContext(r.Context())
	var req struct {
		Kind           string `json:"kind"`
		ThresholdCents int64  `json:"threshold_cents"`
	}
	if err := decodeJSON(r, &req); err != nil || !customAlertKinds[req.Kind] {
		writeError(w, http.StatusBadRequest, "invalid_body",
			"kind (cash_below | net_worth_below | expenses_month_above) et threshold_cents sont requis")
		return
	}
	alert, err := s.store.CreateCustomAlert(r.Context(), p.ID, req.Kind, money.Cents(req.ThresholdCents))
	if err != nil {
		s.storeErr(w, err, "create custom alert")
		return
	}
	writeJSON(w, http.StatusCreated, alert)
}

func (s *Server) handleUpdateCustomAlert(w http.ResponseWriter, r *http.Request) {
	p := profileFromContext(r.Context())
	var req struct {
		ThresholdCents int64 `json:"threshold_cents"`
		Enabled        bool  `json:"enabled"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "threshold_cents et enabled sont requis")
		return
	}
	if err := s.store.UpdateCustomAlert(r.Context(), p.ID, chi.URLParam(r, "id"),
		money.Cents(req.ThresholdCents), req.Enabled); err != nil {
		s.storeErr(w, err, "update custom alert")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"updated": true})
}

func (s *Server) handleDeleteCustomAlert(w http.ResponseWriter, r *http.Request) {
	p := profileFromContext(r.Context())
	if err := s.store.DeleteCustomAlert(r.Context(), p.ID, chi.URLParam(r, "id")); err != nil {
		s.storeErr(w, err, "delete custom alert")
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}

// ── Module fiscal ─────────────────────────────────────────────────────────────

// handleTaxDeadlines renvoie les échéances fiscales de l'année (indicatives).
func (s *Server) handleTaxDeadlines(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"deadlines": tax.Deadlines})
}

// ── Simulateur de crédit ──────────────────────────────────────────────────────

func (s *Server) handleLoanSimulate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PrincipalCents int64 `json:"principal_cents"`
		AnnualRateBps  int   `json:"annual_rate_bps"`
		Months         int   `json:"months"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body",
			"principal_cents, annual_rate_bps et months sont requis")
		return
	}
	result, err := engine.SimulateLoan(money.Cents(req.PrincipalCents), req.AnnualRateBps, req.Months)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input",
			"paramètres hors bornes (principal > 0, taux 0–20 %, durée 1–600 mois)")
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleLoanCapacity(w http.ResponseWriter, r *http.Request) {
	var req struct {
		MonthlyPaymentCents int64 `json:"monthly_payment_cents"`
		AnnualRateBps       int   `json:"annual_rate_bps"`
		Months              int   `json:"months"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body",
			"monthly_payment_cents, annual_rate_bps et months sont requis")
		return
	}
	capacity, err := engine.LoanCapacity(money.Cents(req.MonthlyPaymentCents), req.AnnualRateBps, req.Months)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_input",
			"paramètres hors bornes (mensualité > 0, taux 0–20 %, durée 1–600 mois)")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"capacity_cents": capacity})
}

// ── Bilan annuel « Wrapped » ──────────────────────────────────────────────────

// handleWrapped compile l'année : totaux, top catégories/marchands, épargne.
// Chiffres 100 % moteur (EIA-040) — l'écran iOS ne fait que les mettre en scène.
func (s *Server) handleWrapped(w http.ResponseWriter, r *http.Request) {
	p := profileFromContext(r.Context())
	year := parisToday().Year()
	if raw := r.URL.Query().Get("year"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n >= 2000 && n <= parisToday().Year() {
			year = n
		}
	}
	result, err := s.store.Wrapped(r.Context(), p.ID, year)
	if err != nil {
		s.storeErr(w, err, "wrapped")
		return
	}
	writeJSON(w, 200, result)
}

// ── Push (APNs) ───────────────────────────────────────────────────────────────

// handlePushRegister enregistre le jeton APNs de l'appareil.
func (s *Server) handlePushRegister(w http.ResponseWriter, r *http.Request) {
	p := profileFromContext(r.Context())
	var req struct {
		Token    string `json:"token"`
		Platform string `json:"platform"`
	}
	if err := decodeJSON(r, &req); err != nil || strings.TrimSpace(req.Token) == "" {
		writeError(w, http.StatusBadRequest, "invalid_body", "token est requis")
		return
	}
	if !validPushToken(req.Token) {
		writeError(w, 400, "invalid_token", "Jeton APNs hexadécimal invalide")
		return
	}
	if req.Platform == "" {
		req.Platform = "ios"
	}
	if err := s.store.UpsertPushToken(r.Context(), p.ID, strings.TrimSpace(req.Token), req.Platform); err != nil {
		s.storeErr(w, err, "push register")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"registered":   true,
		"apns_enabled": s.cfg.APNSEnabled(),
	})
}

// ── Profil démo ───────────────────────────────────────────────────────────────

// handleCreateDemoProfile crée une identité isolée à durée limitée avec un PIN aléatoire.
func (s *Server) handleCreateDemoProfile(w http.ResponseWriter, r *http.Request) {
	p, pin, token, expires, err := s.store.CreateDemo(r.Context(), func(st *store.Store, id string) error {
		local := &Server{store: st, cfg: s.cfg, log: s.log, ai: s.ai, vault: s.vault, bank: s.bank, logins: s.logins, jobs: s.jobs}
		return local.seedDemoData(r, id)
	})
	if err != nil {
		if strings.Contains(err.Error(), "rate limited") {
			writeError(w, 429, "rate_limited", "Réessaie dans une minute.")
			return
		}
		s.storeErr(w, err, "demo creation")
		return
	}
	writeJSON(w, 201, map[string]any{"profile": p, "pin": pin, "token": token, "expires_at": expires, "is_demo": true})
}

// seedDemoData amorce un foyer crédible : comptes, patrimoine, crédit,
// 6 mois de mouvements récurrents.
func (s *Server) seedDemoData(r *http.Request, profileID string) error {
	ctx := r.Context()
	now := parisToday()

	// Catégories du référentiel pour catégoriser les mouvements.
	categories, err := s.store.ListCategories(ctx, profileID)
	if err != nil {
		return err
	}
	catID := func(name string) *string {
		for _, c := range categories {
			if strings.EqualFold(c.Name, name) {
				id := c.ID
				return &id
			}
		}
		return nil
	}

	// Actifs.
	checking, err := s.store.CreateAsset(ctx, profileID, "Compte courant", "checking", "EUR", "")
	if err != nil {
		return err
	}
	savings, err := s.store.CreateAsset(ctx, profileID, "Livret A", "savings", "EUR", "")
	if err != nil {
		return err
	}
	pea, err := s.store.CreateAsset(ctx, profileID, "PEA", "pea", "EUR", "ETF monde")
	if err != nil {
		return err
	}
	home, err := s.store.CreateAsset(ctx, profileID, "Appartement", "real_estate", "EUR", "Résidence principale")
	if err != nil {
		return err
	}
	crypto, err := s.store.CreateAsset(ctx, profileID, "Bitcoin", "crypto", "EUR", "")
	if err != nil {
		return err
	}
	mortgage, err := s.store.CreateLiability(ctx, profileID, "Crédit immobilier", "mortgage", "EUR", "Taux 3,2 %")
	if err != nil {
		return err
	}

	// Valorisations (avec un léger historique pour la courbe).
	type val struct {
		assetID   string
		liability bool
		cents     money.Cents
	}
	baseline := []val{
		{checking.ID, false, 245_000},   // 2 450 €
		{savings.ID, false, 1_250_000},  // 12 500 €
		{pea.ID, false, 1_830_000},      // 18 300 €
		{home.ID, false, 21_500_000},    // 215 000 €
		{crypto.ID, false, 320_000},     // 3 200 €
		{mortgage.ID, true, 14_200_000}, // 142 000 € restants
	}
	for monthsAgo := 5; monthsAgo >= 0; monthsAgo-- {
		asOf := now.AddDate(0, -monthsAgo, 0)
		for _, v := range baseline {
			// Trajectoire douce : le patrimoine grimpe, le crédit descend.
			drift := int64(5-monthsAgo) * 1_500 * 100 / 5
			value := int64(v.cents) + drift
			if v.liability {
				value = int64(v.cents) + int64(monthsAgo)*40_000 // il baisse
				if _, err := s.store.AddLiabilityValuation(ctx, profileID, v.assetID,
					money.Cents(value), asOf, ""); err != nil {
					return err
				}
				continue
			}
			if _, err := s.store.AddAssetValuation(ctx, profileID, v.assetID,
				money.Cents(value), asOf, ""); err != nil {
				return err
			}
		}
	}

	// 6 mois de mouvements récurrents + quelques extras.
	type move struct {
		label string
		cents money.Cents
		cat   string
		day   int
	}
	monthly := []move{
		{"Salaire ACME", 268_000, "Revenus", 2},
		{"Loyer perçu studio", 62_000, "Revenus", 5},
		{"Crédit immobilier", -98_500, "Logement", 6},
		{"Carrefour", -18_450, "Courses", 8},
		{"Carrefour", -14_230, "Courses", 22},
		{"Netflix", -1_399, "Loisirs", 10},
		{"Spotify", -1_099, "Loisirs", 12},
		{"EDF", -8_900, "Logement", 15},
		{"Essence TotalEnergies", -7_600, "Transport", 18},
		{"Restaurant", -5_400, "Loisirs", 20},
		{"Pharmacie", -2_150, "Santé", 25},
	}
	for monthsAgo := 5; monthsAgo >= 0; monthsAgo-- {
		base := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, -monthsAgo, 0)
		for _, m := range monthly {
			occurred := base.AddDate(0, 0, m.day-1)
			if occurred.After(now) {
				continue
			}
			if _, err := s.store.CreateTransaction(ctx, profileID, store.NewTransaction{
				AssetID:     checking.ID,
				Amount:      m.cents,
				OccurredOn:  occurred,
				Label:       m.label,
				RawLabel:    m.label,
				MerchantKey: categorize.MerchantKey(m.label),
				CategoryID:  catID(m.cat),
			}); err != nil {
				return err
			}
		}
	}

	// Un objectif d'allocation pour faire vivre l'écran rééquilibrage.
	return s.store.SetAllocationTargets(ctx, profileID, []store.AllocationTarget{
		{Class: "stocks", TargetBps: 4_000},
		{Class: "real_estate", TargetBps: 3_000},
		{Class: "cash", TargetBps: 2_000},
		{Class: "crypto", TargetBps: 500},
		{Class: "other", TargetBps: 500},
	})
}

func validPushToken(token string) bool {
	if len(token) < 32 || len(token) > 200 {
		return false
	}
	for _, c := range token {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F') {
			return false
		}
	}
	return true
}
