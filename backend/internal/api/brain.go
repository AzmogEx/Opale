package api

// Le « cerveau » d'Opale (P5) : Financial Twin, radar de risques,
// Mode Décision, bilan mensuel et assistant.
//
// Contrat non négociable (EIA-040/041) : chaque endpoint renvoie d'abord
// des CHIFFRES calculés par le moteur déterministe. L'IA (cascade N2/N3)
// n'intervient que pour les expliquer — et si aucun niveau n'est
// disponible, un texte de repli déterministe prend sa place (EIA-020/021).

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/opale-app/opale/internal/ai"
	"github.com/opale-app/opale/internal/engine"
	"github.com/opale-app/opale/internal/money"
	"github.com/opale-app/opale/internal/twin"
)

// Hypothèses de référence du twin (identiques aux défauts de /v1/projection).
const (
	twinReturnBps = 500 // 5 %/an
	twinSwrBps    = 400 // règle des 4 %
)

// systemPrompt cadre tous les niveaux de la cascade : l'IA explique, ne
// calcule pas, ne conseille pas de produits, répond en français.
const systemPrompt = `Tu es l'assistant patrimonial de l'application Opale.
Règles absolues :
- Tous les chiffres t'ont été fournis par un moteur de calcul déterministe : tu ne calcules JAMAIS rien toi-même, tu ne modifies jamais un chiffre, tu n'en inventes aucun.
- Tu expliques la situation simplement, en français, avec un ton direct, chaleureux et concret (tutoiement).
- Réponse courte : 3 à 6 phrases, sans titres ni listes à puces, sans formules de politesse.
- Tu ne recommandes jamais de produit financier précis ni d'établissement. Pas de conseil fiscal ou juridique.
- Si une information manque, dis-le plutôt que de supposer.`

// ── Financial Twin (EF-060) ───────────────────────────────────────────────────

// buildTwin assemble le double financier complet du profil : toutes les
// mesures viennent du store, tous les verdicts du moteur.
func (s *Server) buildTwin(r *http.Request, profileID string) (twin.Snapshot, error) {
	ctx := r.Context()

	income3M, expenses3M, err := s.store.FlowTotals3M(ctx, profileID)
	if err != nil {
		return twin.Snapshot{}, err
	}
	cash, err := s.store.CashBalance(ctx, profileID)
	if err != nil {
		return twin.Snapshot{}, err
	}
	nw, err := s.store.ComputeNetWorth(ctx, profileID)
	if err != nil {
		return twin.Snapshot{}, err
	}
	kinds, err := s.store.AssetKindValues(ctx, profileID)
	if err != nil {
		return twin.Snapshot{}, err
	}
	flows, err := s.detectRecurring(r, profileID)
	if err != nil {
		return twin.Snapshot{}, err
	}

	// Habitudes mensualisées.
	monthlyIncome := money.Cents(int64(income3M) / 3)
	monthlyExpenses := money.Cents(int64(expenses3M) / 3)
	monthlySavings := monthlyIncome - monthlyExpenses
	savingsRateBps := 0
	if monthlyIncome > 0 {
		savingsRateBps = int(int64(monthlySavings) * 10_000 / int64(monthlyIncome))
	}

	// Charges fixes et sources de revenus (flux récurrents actifs).
	var fixedMonthly int64
	incomeSources := 0
	keys := make([]string, 0, len(flows))
	for _, f := range flows {
		keys = append(keys, f.MerchantKey)
		if !f.Active || f.IntervalDays <= 0 {
			continue
		}
		if f.Amount < 0 {
			fixedMonthly += -int64(f.Amount) * 30 / int64(f.IntervalDays)
		} else {
			incomeSources++
		}
	}

	// Cash projeté à 30 jours (pour le radar).
	daily, err := s.store.AvgDailyVariableSpend(ctx, profileID, keys)
	if err != nil {
		return twin.Snapshot{}, err
	}
	today := time.Now().Truncate(24 * time.Hour)
	proj := engine.ProjectCash(cash, flows, today, today.AddDate(0, 0, 30), daily)

	snap := twin.Snapshot{
		NetWorth:        nw.Net,
		Assets:          nw.AssetsTotal,
		Liabilities:     nw.LiabilitiesTotal,
		Cash:            cash,
		AssetKinds:      kinds,
		MonthlyIncome:   monthlyIncome,
		MonthlyExpenses: monthlyExpenses,
		MonthlySavings:  monthlySavings,
		FixedMonthly:    money.Cents(fixedMonthly),
		SavingsRateBps:  savingsRateBps,
	}

	snap.Health = engine.ComputeHealthScore(engine.HealthInputs{
		Income3M:        income3M,
		Expenses3M:      expenses3M,
		Cash:            cash,
		Assets:          nw.AssetsTotal,
		Liabilities:     nw.LiabilitiesTotal,
		FixedMonthly:    money.Cents(fixedMonthly),
		AssetKindValues: kinds,
	})

	snap.Risks = engine.DetectRisks(engine.RiskInputs{
		Cash:             cash,
		Assets:           nw.AssetsTotal,
		Liabilities:      nw.LiabilitiesTotal,
		Income3M:         income3M,
		Expenses3M:       expenses3M,
		FixedMonthly:     money.Cents(fixedMonthly),
		IncomeSources:    incomeSources,
		ProjectedCash30d: proj.EndCash,
		HasProjection:    true,
		AssetKindValues:  kinds,
	})

	if monthlyExpenses > 0 {
		if ind, err := engine.ComputeIndependence(nw.Net, monthlySavings, monthlyExpenses,
			twinReturnBps, twinSwrBps); err == nil {
			snap.Independence = ind
		}
	}

	if statuses, err := s.goalStatuses(r, profileID); err == nil {
		for _, g := range statuses {
			snap.Goals = append(snap.Goals, twin.Goal{
				Name: g.Name, Target: g.Target, Percent: g.Percent, OnTrack: g.OnTrack,
			})
		}
	}

	return snap, nil
}

func (s *Server) handleTwin(w http.ResponseWriter, r *http.Request) {
	p := profileFromContext(r.Context())
	snap, err := s.buildTwin(r, p.ID)
	if err != nil {
		s.storeErr(w, err, "twin")
		return
	}
	writeJSON(w, http.StatusOK, snap)
}

// ── Radar de risques (EF-061) ─────────────────────────────────────────────────

func (s *Server) handleRisks(w http.ResponseWriter, r *http.Request) {
	p := profileFromContext(r.Context())
	snap, err := s.buildTwin(r, p.ID)
	if err != nil {
		s.storeErr(w, err, "risks")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"risks": snap.Risks})
}

// ── Mode Décision (EF-052) ────────────────────────────────────────────────────

type decisionRequest struct {
	Label           string `json:"label"`
	OneTimeCost     int64  `json:"one_time_cost_cents"`
	MonthlyCost     int64  `json:"monthly_cost_cents"`
	AnnualReturnBps int    `json:"annual_return_bps"` // défaut : 500
	AllowCloud      bool   `json:"allow_cloud"`       // EIA-022
}

func (s *Server) handleDecision(w http.ResponseWriter, r *http.Request) {
	p := profileFromContext(r.Context())
	var req decisionRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}
	if req.Label == "" {
		writeError(w, http.StatusBadRequest, "invalid_body", "label est requis")
		return
	}
	if req.OneTimeCost == 0 && req.MonthlyCost == 0 {
		writeError(w, http.StatusBadRequest, "invalid_body",
			"one_time_cost_cents ou monthly_cost_cents doit être non nul")
		return
	}
	if req.AnnualReturnBps == 0 {
		req.AnnualReturnBps = twinReturnBps
	}

	snap, err := s.buildTwin(r, p.ID)
	if err != nil {
		s.storeErr(w, err, "decision: twin")
		return
	}

	// 1. Le moteur calcule (EIA-040).
	impact, err := engine.EvaluateDecision(engine.DecisionInputs{
		NetWorth:        snap.NetWorth,
		Cash:            snap.Cash,
		MonthlySavings:  snap.MonthlySavings,
		MonthlyExpenses: snap.MonthlyExpenses,
		AnnualReturnBps: req.AnnualReturnBps,
		SwrBps:          twinSwrBps,
		OneTimeCost:     money.Cents(req.OneTimeCost),
		MonthlyCost:     money.Cents(req.MonthlyCost),
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_decision", err.Error())
		return
	}

	// 2. L'IA explique (EIA-041) — repli : la recommandation gabarit.
	normal := impact.Scenarios[1]
	facts := fmt.Sprintf(
		"Décision étudiée : « %s ». Coût immédiat %s, charge mensuelle %s.\n"+
			"Verdict du moteur : risque %s. Payable cash : %t. Épargne mensuelle après : %s.\n"+
			"Scénario normal (%d bps) : impact %s à 5 ans, %s à 10 ans, indépendance financière retardée de %d mois.\n"+
			"Recommandation du moteur : %s",
		req.Label, eurosText(money.Cents(req.OneTimeCost)), eurosText(money.Cents(req.MonthlyCost)),
		impact.RiskLevel, impact.AffordableCash, eurosText(impact.SavingsAfter),
		normal.ReturnBps, eurosText(normal.Delta5y), eurosText(normal.Delta10y), normal.DelayMonths,
		impact.Recommendation,
	)

	narrative, tier := s.explain(r, ai.Request{
		Task:   "decision",
		System: systemPrompt,
		Prompt: "Explique ce verdict à l'utilisateur. Contexte :\n" + twin.Describe(snap) + "\n" + facts,
		AnonymizedPrompt: "Explique ce verdict à l'utilisateur. Contexte :\n" + twin.Anonymize(snap) + "\n" +
			fmt.Sprintf("Décision : dépense immédiate %s, charge mensuelle %s. Verdict moteur : risque %s, retard d'indépendance %d mois. Recommandation : %s",
				compactText(money.Cents(req.OneTimeCost)), compactText(money.Cents(req.MonthlyCost)),
				impact.RiskLevel, normal.DelayMonths, impact.Recommendation),
		AllowCloud: req.AllowCloud,
	}, impact.Recommendation)

	writeJSON(w, http.StatusOK, map[string]any{
		"label":          req.Label,
		"impact":         impact,
		"narrative":      narrative,
		"narrative_tier": tier,
	})
}

// ── Bilan mensuel (EF-062) ────────────────────────────────────────────────────

func (s *Server) handleMonthlyReview(w http.ResponseWriter, r *http.Request) {
	p := profileFromContext(r.Context())

	// Mois demandé (défaut : le mois précédent, celui qu'on « clôture »).
	now := time.Now()
	year, month := now.AddDate(0, -1, 0).Year(), now.AddDate(0, -1, 0).Month()
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
	allowCloud := r.URL.Query().Get("allow_cloud") == "true"

	summary, err := s.store.ComputeMonthSummary(r.Context(), p.ID, year, month)
	if err != nil {
		s.storeErr(w, err, "review: summary")
		return
	}
	topCategories, err := s.store.SpendingByCategory(r.Context(), p.ID, year, month, 3)
	if err != nil {
		s.storeErr(w, err, "review: categories")
		return
	}
	snap, err := s.buildTwin(r, p.ID)
	if err != nil {
		s.storeErr(w, err, "review: twin")
		return
	}

	savingsRateBps := 0
	if summary.Income > 0 {
		savingsRateBps = int(int64(summary.Net) * 10_000 / int64(summary.Income))
	}

	// Repli déterministe : un bilan gabarit, purement factuel.
	monthName := frenchMonths[month-1]
	var fallback strings.Builder
	fmt.Fprintf(&fallback, "En %s : %s de revenus, %s de dépenses, soit %s mis de côté (taux d'épargne %d %%).",
		monthName, eurosText(summary.Income), eurosText(summary.Expenses), eurosText(summary.Net), savingsRateBps/100)
	if len(topCategories) > 0 {
		fmt.Fprintf(&fallback, " Premier poste de dépense : %s (%s).",
			strings.ToLower(topCategories[0].Name), eurosText(topCategories[0].Total))
	}
	fmt.Fprintf(&fallback, " Score de santé financière : %d/100.", snap.Health.Score)

	// Les faits chiffrés transmis à l'IA pour rédaction (EIA-041).
	var facts strings.Builder
	fmt.Fprintf(&facts, "Bilan du mois de %s %d (chiffres du moteur) :\n", monthName, year)
	fmt.Fprintf(&facts, "- Revenus %s, dépenses %s, épargne %s (taux %d %%)\n",
		eurosText(summary.Income), eurosText(summary.Expenses), eurosText(summary.Net), savingsRateBps/100)
	for _, c := range topCategories {
		fmt.Fprintf(&facts, "- Poste « %s » : %s\n", c.Name, eurosText(c.Total))
	}
	anonFacts := fmt.Sprintf(
		"Bilan du mois (chiffres agrégés) : revenus %s, dépenses %s, épargne %s, taux d'épargne %d %%.",
		compactText(summary.Income), compactText(summary.Expenses), compactText(summary.Net), savingsRateBps/100)

	narrative, tier := s.explain(r, ai.Request{
		Task:   "monthly_review",
		System: systemPrompt,
		Prompt: "Rédige le bilan mensuel de l'utilisateur : ce qui va, ce qui coince, et UNE suggestion concrète.\n" +
			twin.Describe(snap) + "\n" + facts.String(),
		AnonymizedPrompt: "Rédige le bilan mensuel de l'utilisateur : ce qui va, ce qui coince, et UNE suggestion concrète.\n" +
			twin.Anonymize(snap) + "\n" + anonFacts,
		AllowCloud: allowCloud,
	}, fallback.String())

	writeJSON(w, http.StatusOK, map[string]any{
		"year":             year,
		"month":            int(month),
		"summary":          summary,
		"savings_rate_bps": savingsRateBps,
		"top_categories":   topCategories,
		"health_score":     snap.Health.Score,
		"narrative":        narrative,
		"narrative_tier":   tier,
	})
}

// ── Assistant (EF-050/051) ────────────────────────────────────────────────────

type askRequest struct {
	Question   string `json:"question"`
	AllowCloud bool   `json:"allow_cloud"` // EIA-022
}

func (s *Server) handleAssistantAsk(w http.ResponseWriter, r *http.Request) {
	p := profileFromContext(r.Context())
	var req askRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}
	req.Question = strings.TrimSpace(req.Question)
	if req.Question == "" {
		writeError(w, http.StatusBadRequest, "invalid_body", "question est requise")
		return
	}
	if len(req.Question) > 2_000 {
		writeError(w, http.StatusBadRequest, "invalid_body", "question trop longue (2000 caractères max)")
		return
	}

	snap, err := s.buildTwin(r, p.ID)
	if err != nil {
		s.storeErr(w, err, "assistant: twin")
		return
	}

	// Repli déterministe (EIA-021) : l'essentiel du twin, sans IA.
	fallback := fmt.Sprintf(
		"L'analyse IA est indisponible pour le moment (homelab hors ligne%s). "+
			"Voici l'essentiel calculé par le moteur : patrimoine net %s, cash %s, "+
			"épargne mensuelle %s (taux %d %%), score de santé %d/100.",
		cloudHint(s.ai, req.AllowCloud),
		eurosText(snap.NetWorth), eurosText(snap.Cash),
		eurosText(snap.MonthlySavings), snap.SavingsRateBps/100, snap.Health.Score)

	answer, tier := s.explain(r, ai.Request{
		Task:   "assistant_ask",
		System: systemPrompt,
		Prompt: twin.Describe(snap) + "\nQuestion de l'utilisateur : " + req.Question,
		AnonymizedPrompt: twin.Anonymize(snap) +
			"\nQuestion de l'utilisateur (ne réutilise aucun nom propre qu'elle contiendrait) : " + req.Question,
		AllowCloud: req.AllowCloud,
		MaxTokens:  900,
	}, fallback)

	writeJSON(w, http.StatusOK, map[string]any{
		"answer": answer,
		"tier":   tier,
	})
}

// handleAssistantStatus expose l'état de la cascade (UX EIA-021/022).
func (s *Server) handleAssistantStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"homelab_available": s.ai.HomelabAvailable(r.Context()),
		"cloud_configured":  s.ai.CloudConfigured(),
	})
}

// ── Aides ─────────────────────────────────────────────────────────────────────

// explain interroge la cascade et retombe sur le texte déterministe.
func (s *Server) explain(r *http.Request, req ai.Request, fallback string) (text, tier string) {
	resp, err := s.ai.Explain(r.Context(), req)
	if err != nil {
		return fallback, ai.TierNone
	}
	return resp.Text, resp.Tier
}

// cloudHint complète le message de repli selon l'état du cloud (EIA-021/022).
func cloudHint(router *ai.Router, allowCloud bool) string {
	if !router.CloudConfigured() {
		return ", cloud non configuré"
	}
	if !allowCloud {
		return ", cloud non autorisé pour cette question"
	}
	return ""
}

// eurosText — montant lisible en euros entiers, signe inclus.
func eurosText(c money.Cents) string {
	return fmt.Sprintf("%d €", int64(c)/100)
}

// compactText — montant arrondi façon « 42k » (contextes anonymisés).
func compactText(c money.Cents) string {
	e := int64(c) / 100
	if e >= 1_000 || e <= -1_000 {
		return fmt.Sprintf("%dk", (e+500)/1_000)
	}
	return fmt.Sprintf("%d", e)
}

var frenchMonths = [12]string{
	"janvier", "février", "mars", "avril", "mai", "juin",
	"juillet", "août", "septembre", "octobre", "novembre", "décembre",
}
