package api

import (
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/opale-app/opale/internal/engine"
	"github.com/opale-app/opale/internal/money"
	"github.com/opale-app/opale/internal/store"
)

// ── Enveloppes (EF-028) ───────────────────────────────────────────────────────

func (s *Server) handleEnvelopeStatuses(w http.ResponseWriter, r *http.Request) {
	p := profileFromContext(r.Context())
	now := time.Now()
	year, month := now.Year(), int(now.Month())
	if raw := r.URL.Query().Get("year"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil {
			year = n
		}
	}
	if raw := r.URL.Query().Get("month"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n >= 1 && n <= 12 {
			month = n
		}
	}
	statuses, err := s.store.EnvelopeStatuses(r.Context(), p.ID, year, time.Month(month))
	if err != nil {
		s.storeErr(w, err, "envelope statuses")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"envelopes": statuses})
}

type envelopeRequest struct {
	CategoryID string `json:"category_id"`
	Budget     int64  `json:"monthly_budget_cents"`
}

func (s *Server) handleUpsertEnvelope(w http.ResponseWriter, r *http.Request) {
	p := profileFromContext(r.Context())
	var req envelopeRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}
	if req.CategoryID == "" || req.Budget <= 0 {
		writeError(w, http.StatusBadRequest, "invalid_body",
			"category_id et monthly_budget_cents (> 0) sont requis")
		return
	}
	env, err := s.store.UpsertEnvelope(r.Context(), p.ID, req.CategoryID, money.Cents(req.Budget))
	if err != nil {
		s.storeErr(w, err, "upsert envelope")
		return
	}
	writeJSON(w, http.StatusOK, env)
}

func (s *Server) handleDeleteEnvelope(w http.ResponseWriter, r *http.Request) {
	p := profileFromContext(r.Context())
	if err := s.store.DeleteEnvelope(r.Context(), p.ID, chi.URLParam(r, "id")); err != nil {
		s.storeErr(w, err, "delete envelope")
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}

// ── Récurrents & cashflow (EF-025/026/027) ────────────────────────────────────

// detectRecurring centralise la détection pour les endpoints qui en dépendent.
func (s *Server) detectRecurring(r *http.Request, profileID string) ([]engine.RecurringFlow, error) {
	obs, err := s.store.RecurringObservations(r.Context(), profileID)
	if err != nil {
		return nil, err
	}
	return engine.DetectRecurring(obs, time.Now()), nil
}

func (s *Server) handleRecurring(w http.ResponseWriter, r *http.Request) {
	p := profileFromContext(r.Context())
	flows, err := s.detectRecurring(r, p.ID)
	if err != nil {
		s.storeErr(w, err, "recurring")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"recurring": flows})
}

func (s *Server) handleCashflow(w http.ResponseWriter, r *http.Request) {
	p := profileFromContext(r.Context())

	days := 30
	if raw := r.URL.Query().Get("days"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n >= 1 && n <= 365 {
			days = n
		}
	}

	flows, err := s.detectRecurring(r, p.ID)
	if err != nil {
		s.storeErr(w, err, "cashflow: recurring")
		return
	}
	cash, err := s.store.CashBalance(r.Context(), p.ID)
	if err != nil {
		s.storeErr(w, err, "cashflow: cash")
		return
	}
	keys := make([]string, 0, len(flows))
	for _, f := range flows {
		keys = append(keys, f.MerchantKey)
	}
	daily, err := s.store.AvgDailyVariableSpend(r.Context(), p.ID, keys)
	if err != nil {
		s.storeErr(w, err, "cashflow: variable spend")
		return
	}

	today := time.Now().Truncate(24 * time.Hour)
	proj := engine.ProjectCash(cash, flows, today, today.AddDate(0, 0, days), daily)
	writeJSON(w, http.StatusOK, proj)
}

// ── Score de santé (EF-015) ───────────────────────────────────────────────────

func (s *Server) handleHealthScore(w http.ResponseWriter, r *http.Request) {
	p := profileFromContext(r.Context())

	income, expenses, err := s.store.FlowTotals3M(r.Context(), p.ID)
	if err != nil {
		s.storeErr(w, err, "health: flows")
		return
	}
	cash, err := s.store.CashBalance(r.Context(), p.ID)
	if err != nil {
		s.storeErr(w, err, "health: cash")
		return
	}
	nw, err := s.store.ComputeNetWorth(r.Context(), p.ID)
	if err != nil {
		s.storeErr(w, err, "health: net worth")
		return
	}
	kinds, err := s.store.AssetKindValues(r.Context(), p.ID)
	if err != nil {
		s.storeErr(w, err, "health: kinds")
		return
	}
	flows, err := s.detectRecurring(r, p.ID)
	if err != nil {
		s.storeErr(w, err, "health: recurring")
		return
	}

	// Charges fixes mensualisées : Σ |montant| × 30 / intervalle (flux actifs négatifs).
	var fixedMonthly int64
	for _, f := range flows {
		if f.Active && f.Amount < 0 && f.IntervalDays > 0 {
			fixedMonthly += -int64(f.Amount) * 30 / int64(f.IntervalDays)
		}
	}

	score := engine.ComputeHealthScore(engine.HealthInputs{
		Income3M:        income,
		Expenses3M:      expenses,
		Cash:            cash,
		Assets:          nw.AssetsTotal,
		Liabilities:     nw.LiabilitiesTotal,
		FixedMonthly:    money.Cents(fixedMonthly),
		AssetKindValues: kinds,
	})
	writeJSON(w, http.StatusOK, score)
}

// ── Objectifs (EF-042) ────────────────────────────────────────────────────────

// goalStatus — objectif enrichi de sa progression (calcul déterministe).
type goalStatus struct {
	store.Goal
	Progress money.Cents `json:"progress_cents"`
	Percent  int         `json:"percent"` // 0..100 (borné)
	OnTrack  *bool       `json:"on_track,omitempty"`
}

func (s *Server) goalStatuses(r *http.Request, profileID string) ([]goalStatus, error) {
	goals, err := s.store.ListGoals(r.Context(), profileID)
	if err != nil {
		return nil, err
	}
	var nw *store.NetWorth
	out := make([]goalStatus, 0, len(goals))
	for _, g := range goals {
		st := goalStatus{Goal: g}

		// Progression : l'actif source si défini, sinon le patrimoine net.
		if g.AssetID != nil {
			v, err := s.store.AssetLatestValue(r.Context(), profileID, *g.AssetID)
			if err != nil {
				return nil, err
			}
			st.Progress = v
		} else {
			if nw == nil {
				v, err := s.store.ComputeNetWorth(r.Context(), profileID)
				if err != nil {
					return nil, err
				}
				nw = &v
			}
			st.Progress = nw.Net
		}

		if g.Target > 0 {
			pct := int(int64(st.Progress) * 100 / int64(g.Target))
			st.Percent = min(max(pct, 0), 100)
		}

		// En avance / en retard : progression comparée à la droite temps→cible.
		if g.TargetDate != nil {
			total := g.TargetDate.Sub(g.CreatedAt)
			elapsed := time.Since(g.CreatedAt)
			if total > 0 && elapsed > 0 {
				expected := int64(g.Target) * int64(elapsed) / int64(total)
				onTrack := int64(st.Progress) >= expected
				st.OnTrack = &onTrack
			}
		}
		out = append(out, st)
	}
	return out, nil
}

func (s *Server) handleListGoals(w http.ResponseWriter, r *http.Request) {
	p := profileFromContext(r.Context())
	statuses, err := s.goalStatuses(r, p.ID)
	if err != nil {
		s.storeErr(w, err, "list goals")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"goals": statuses})
}

type goalRequest struct {
	Name       string `json:"name"`
	Icon       string `json:"icon"`
	Target     int64  `json:"target_cents"`
	TargetDate string `json:"target_date"` // yyyy-MM-dd, optionnel
	AssetID    string `json:"asset_id"`    // optionnel
}

func (s *Server) handleCreateGoal(w http.ResponseWriter, r *http.Request) {
	p := profileFromContext(r.Context())
	var req goalRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}
	if req.Name == "" || req.Target <= 0 {
		writeError(w, http.StatusBadRequest, "invalid_body", "name et target_cents (> 0) requis")
		return
	}
	if req.Icon == "" {
		req.Icon = "target"
	}
	var targetDate *time.Time
	if req.TargetDate != "" {
		t, err := time.Parse(dayLayout, req.TargetDate)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_date", "target_date doit être yyyy-MM-dd")
			return
		}
		targetDate = &t
	}
	var assetID *string
	if req.AssetID != "" {
		assetID = &req.AssetID
	}

	g, err := s.store.CreateGoal(r.Context(), p.ID, req.Name, req.Icon,
		money.Cents(req.Target), targetDate, assetID)
	if err != nil {
		s.storeErr(w, err, "create goal")
		return
	}
	writeJSON(w, http.StatusCreated, g)
}

func (s *Server) handleDeleteGoal(w http.ResponseWriter, r *http.Request) {
	p := profileFromContext(r.Context())
	if err := s.store.DeleteGoal(r.Context(), p.ID, chi.URLParam(r, "id")); err != nil {
		s.storeErr(w, err, "delete goal")
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}

// ── Alertes (EF-053) ──────────────────────────────────────────────────────────

// alert — une alerte intelligente calculée à la volée.
type alert struct {
	Kind     string `json:"kind"`     // envelope_overrun | low_cash | goal_late
	Severity string `json:"severity"` // warning | critical
	Title    string `json:"title"`
	Detail   string `json:"detail"`
}

func (s *Server) handleAlerts(w http.ResponseWriter, r *http.Request) {
	p := profileFromContext(r.Context())
	alerts := []alert{}
	now := time.Now()

	// 1. Enveloppes dépassées ce mois (EF-053).
	if statuses, err := s.store.EnvelopeStatuses(r.Context(), p.ID, now.Year(), now.Month()); err == nil {
		for _, st := range statuses {
			if st.Remaining < 0 {
				alerts = append(alerts, alert{
					Kind:     "envelope_overrun",
					Severity: "warning",
					Title:    "Enveloppe « " + st.CategoryName + " » dépassée",
					Detail:   st.Spent.String() + " € dépensés sur " + st.Budget.String() + " € budgétés",
				})
			}
		}
	}

	// 2. Solde bas projeté à 30 jours (EF-027 + EF-053).
	if flows, err := s.detectRecurring(r, p.ID); err == nil {
		if cash, err := s.store.CashBalance(r.Context(), p.ID); err == nil {
			keys := make([]string, 0, len(flows))
			for _, f := range flows {
				keys = append(keys, f.MerchantKey)
			}
			daily, _ := s.store.AvgDailyVariableSpend(r.Context(), p.ID, keys)
			today := now.Truncate(24 * time.Hour)
			proj := engine.ProjectCash(cash, flows, today, today.AddDate(0, 0, 30), daily)
			if proj.EndCash < 0 {
				alerts = append(alerts, alert{
					Kind:     "low_cash",
					Severity: "critical",
					Title:    "Solde négatif prévu",
					Detail:   "Cash projeté à 30 jours : " + proj.EndCash.String() + " €",
				})
			}
		}
	}

	// 3. Objectifs en retard (EF-042 + EF-053).
	if statuses, err := s.goalStatuses(r, p.ID); err == nil {
		for _, st := range statuses {
			if st.OnTrack != nil && !*st.OnTrack {
				alerts = append(alerts, alert{
					Kind:     "goal_late",
					Severity: "warning",
					Title:    "Objectif « " + st.Name + " » en retard",
					Detail:   "Progression " + strconv.Itoa(st.Percent) + " % — en dessous de la trajectoire prévue",
				})
			}
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{"alerts": alerts})
}

// ── Analyses (écran Analyses — le cœur addictif) ──────────────────────────────

// handleAnalytics agrège le mois demandé : dépenses par catégorie, top
// marchands, et comparaison avec le mois précédent. Calculs SQL/entiers.
func (s *Server) handleAnalytics(w http.ResponseWriter, r *http.Request) {
	p := profileFromContext(r.Context())

	now := time.Now()
	year, month := now.Year(), now.Month()
	if raw := r.URL.Query().Get("year"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil {
			year = n
		}
	}
	if raw := r.URL.Query().Get("month"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n >= 1 && n <= 12 {
			month = time.Month(n)
		}
	}
	prev := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC).AddDate(0, -1, 0)

	summary, err := s.store.ComputeMonthSummary(r.Context(), p.ID, year, month)
	if err != nil {
		s.storeErr(w, err, "analytics: summary")
		return
	}
	prevSummary, err := s.store.ComputeMonthSummary(r.Context(), p.ID, prev.Year(), prev.Month())
	if err != nil {
		s.storeErr(w, err, "analytics: prev")
		return
	}
	categories, err := s.store.SpendingByCategory(r.Context(), p.ID, year, month, 12)
	if err != nil {
		s.storeErr(w, err, "analytics: categories")
		return
	}
	merchants, err := s.store.TopMerchants(r.Context(), p.ID, year, month, 8)
	if err != nil {
		s.storeErr(w, err, "analytics: merchants")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"year":            year,
		"month":           int(month),
		"summary":         summary,
		"previous":        prevSummary,
		"categories":      categories,
		"top_merchants":   merchants,
	})
}

// ── Abonnements (gestionnaire d'abonnements) ──────────────────────────────────

// subscriptionStatus — un abonnement détecté + son poids sur la liberté.
type subscriptionStatus struct {
	engine.RecurringFlow
	// Coût mensualisé (|montant| × 30 / intervalle).
	MonthlyCost money.Cents `json:"monthly_cost_cents"`
	YearlyCost  money.Cents `json:"yearly_cost_cents"`
	// Résilier cet abonnement avance la liberté de N mois (0 si inconnu).
	FreedomGainMonths int `json:"freedom_gain_months"`
}

// handleSubscriptions liste les prélèvements récurrents actifs avec leur
// coût réel et le gain d'indépendance si on les résilie — LE chiffre qui
// fait réfléchir, calculé par le moteur (EIA-040).
func (s *Server) handleSubscriptions(w http.ResponseWriter, r *http.Request) {
	p := profileFromContext(r.Context())

	flows, err := s.detectRecurring(r, p.ID)
	if err != nil {
		s.storeErr(w, err, "subscriptions: recurring")
		return
	}

	// La situation de référence (mêmes hypothèses que le twin).
	snap, err := s.buildTwin(r, p.ID)
	if err != nil {
		s.storeErr(w, err, "subscriptions: twin")
		return
	}
	var baseMonths int
	baseReached := false
	if snap.MonthlyExpenses > 0 {
		if base, err := engine.ComputeIndependence(snap.NetWorth, snap.MonthlySavings,
			snap.MonthlyExpenses, twinReturnBps, twinSwrBps); err == nil && base.Reached {
			baseMonths, baseReached = base.Months, true
		}
	}

	var out []subscriptionStatus
	var totalMonthly int64
	for _, f := range flows {
		if !f.Active || f.Amount >= 0 || f.IntervalDays <= 0 {
			continue
		}
		monthly := -int64(f.Amount) * 30 / int64(f.IntervalDays)
		st := subscriptionStatus{
			RecurringFlow: f,
			MonthlyCost:   money.Cents(monthly),
			YearlyCost:    money.Cents(monthly * 12),
		}
		// Résilier = épargner `monthly` de plus ET dépenser autant de moins.
		if baseReached && snap.MonthlyExpenses > money.Cents(monthly) {
			if after, err := engine.ComputeIndependence(
				snap.NetWorth,
				snap.MonthlySavings+money.Cents(monthly),
				snap.MonthlyExpenses-money.Cents(monthly),
				twinReturnBps, twinSwrBps,
			); err == nil && after.Reached && baseMonths > after.Months {
				st.FreedomGainMonths = baseMonths - after.Months
			}
		}
		totalMonthly += monthly
		out = append(out, st)
	}

	// Les plus chers d'abord.
	sort.Slice(out, func(i, j int) bool { return out[i].MonthlyCost > out[j].MonthlyCost })

	writeJSON(w, http.StatusOK, map[string]any{
		"subscriptions":       out,
		"total_monthly_cents": totalMonthly,
		"total_yearly_cents":  totalMonthly * 12,
	})
}
