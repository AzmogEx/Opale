package api

// Le « cerveau » d'Opale (P5) : Financial Twin, radar de risques,
// Mode Décision, bilan mensuel et assistant.
//
// Contrat non négociable (EIA-040/041) : chaque endpoint renvoie d'abord
// des CHIFFRES calculés par le moteur déterministe. L'IA (cascade N2/N3)
// n'intervient que pour les expliquer — et si aucun niveau n'est
// disponible, un texte de repli déterministe prend sa place (EIA-020/021).

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/opale-app/opale/internal/ai"
	"github.com/opale-app/opale/internal/engine"
	"github.com/opale-app/opale/internal/money"
	"github.com/opale-app/opale/internal/nlq"
	"github.com/opale-app/opale/internal/store"
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
	var result twin.Snapshot
	err := s.store.Snapshot(r.Context(), func(st *store.Store) error {
		view := &Server{store: st, cfg: s.cfg, log: s.log, ai: s.ai}
		var e error
		result, e = view.buildTwinSnapshot(r, profileID)
		return e
	})
	return result, err
}

func (s *Server) buildTwinSnapshot(r *http.Request, profileID string) (twin.Snapshot, error) {
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
		ratio := new(big.Int).Mul(big.NewInt(int64(monthlySavings)), big.NewInt(10000))
		ratio.Quo(ratio, big.NewInt(int64(monthlyIncome)))
		if !ratio.IsInt64() {
			return twin.Snapshot{}, money.ErrOverflow
		}
		savingsRateBps = int(ratio.Int64())
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
			value := new(big.Int).Mul(big.NewInt(int64(f.Amount)), big.NewInt(-30))
			value.Quo(value, big.NewInt(int64(f.IntervalDays)))
			value.Add(value, big.NewInt(fixedMonthly))
			if !value.IsInt64() {
				return twin.Snapshot{}, money.ErrOverflow
			}
			fixedMonthly = value.Int64()
		} else {
			incomeSources++
		}
	}

	// Cash projeté à 30 jours (pour le radar).
	daily, err := s.store.AvgDailyVariableSpend(ctx, profileID, keys)
	if err != nil {
		return twin.Snapshot{}, err
	}
	today := parisToday()
	proj, err := engine.ProjectCash(cash, flows, today, today.AddDate(0, 0, 30), daily)
	if err != nil {
		return twin.Snapshot{}, err
	}
	if err := s.addCalendarFlows(r, profileID, today, &proj); err != nil {
		return twin.Snapshot{}, err
	}

	snap := twin.Snapshot{
		Complete:          nw.Complete,
		MissingValuations: nw.MissingValuations,
		NetWorth:          nw.Net,
		Assets:            nw.AssetsTotal,
		Liabilities:       nw.LiabilitiesTotal,
		Cash:              cash,
		AssetKinds:        kinds,
		MonthlyIncome:     monthlyIncome,
		MonthlyExpenses:   monthlyExpenses,
		MonthlySavings:    monthlySavings,
		FixedMonthly:      money.Cents(fixedMonthly),
		SavingsRateBps:    savingsRateBps,
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
		ind, err := engine.ComputeIndependence(nw.Net, monthlySavings, monthlyExpenses, twinReturnBps, twinSwrBps)
		if err != nil {
			return twin.Snapshot{}, err
		}
		snap.Independence = ind
	}

	statuses, err := s.goalStatuses(r, profileID)
	if err != nil {
		return twin.Snapshot{}, err
	}
	for _, g := range statuses {
		snap.Goals = append(snap.Goals, twin.Goal{
			Name: g.Name, Target: g.Target, Percent: g.Percent, OnTrack: g.OnTrack,
		})
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
		Task:       "decision",
		System:     systemPrompt,
		Prompt:     "Explique ce verdict à l'utilisateur. Contexte :\n" + twin.Describe(snap) + "\n" + facts,
		CloudFacts: cloudContext(snap, "decision"),
		AllowCloud: req.AllowCloud,
	}, impact.Recommendation)
	if !snap.Complete {
		narrative = "Données patrimoniales incomplètes : les résultats dépendent des valeurs renseignées. " + narrative
	}

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
	now := parisToday()
	previous := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).AddDate(0, -1, 0)
	year, month := previous.Year(), previous.Month()
	if raw := r.URL.Query().Get("year"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1900 || n > now.Year() {
			writeError(w, 422, "invalid_period", "Année invalide")
			return
		}
		year = n
	}
	if raw := r.URL.Query().Get("month"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > 12 {
			writeError(w, 422, "invalid_period", "Mois invalide")
			return
		}
		month = time.Month(n)
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
		value := new(big.Int).Mul(big.NewInt(int64(summary.Net)), big.NewInt(10000))
		value.Quo(value, big.NewInt(int64(summary.Income)))
		if !value.IsInt64() {
			writeError(w, 422, "overflow", "Ratio hors limites")
			return
		}
		savingsRateBps = int(value.Int64())
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

	narrative, tier := s.explain(r, ai.Request{
		Task:   "monthly_review",
		System: systemPrompt,
		Prompt: "Rédige le bilan mensuel de l'utilisateur : ce qui va, ce qui coince, et UNE suggestion concrète.\n" +
			twin.Describe(snap) + "\n" + facts.String(),
		CloudFacts: cloudContext(snap, "monthly_review"),
		AllowCloud: allowCloud,
	}, fallback.String())
	if !snap.Complete {
		narrative += " Données patrimoniales incomplètes : le score dépend des valeurs renseignées."
	}

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

type conversationMessage struct {
	Role string `json:"role"`
	Text string `json:"text"`
}
type askRequest struct {
	Provider   string                `json:"provider,omitempty"`
	Question   string                `json:"question"`
	AllowCloud bool                  `json:"allow_cloud"`
	History    []conversationMessage `json:"history,omitempty"`
}
type assistantFact struct {
	ID     string `json:"id"`
	Value  *int64 `json:"value_cents,omitempty"`
	Unit   string `json:"unit"`
	Period string `json:"period"`
	Source string `json:"source"`
	Text   string `json:"text"`
}

func amountFact(id string, value money.Cents, period, source, label string) assistantFact {
	v := int64(value)
	return assistantFact{id, &v, "EUR", period, source, label + " : " + eurosText(value) + "."}
}
func (s *Server) handleAssistantAsk(w http.ResponseWriter, r *http.Request) {
	p := profileFromContext(r.Context())
	var req askRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, 400, "invalid_body", err.Error())
		return
	}
	if req.Provider != "" && req.Provider != "auto" && req.Provider != "homelab" && req.Provider != "cloud" {
		writeError(w, 400, "invalid_provider", "Choisis auto, homelab ou cloud")
		return
	}
	req.Question = strings.TrimSpace(req.Question)
	if req.Question == "" || len(req.Question) > 2000 || len(req.History) > 20 {
		writeError(w, 400, "invalid_body", "Question de 1 à 2000 caractères et historique de 20 messages maximum")
		return
	}
	for _, m := range req.History {
		if (m.Role != "user" && m.Role != "assistant") || len(m.Text) > 4000 {
			writeError(w, 400, "invalid_history", "Historique invalide")
			return
		}
	}
	question := req.Question
	// Follow-up periods inherit only the preceding user's deterministic topic.
	if strings.HasPrefix(strings.ToLower(question), "et ") {
		for i := len(req.History) - 1; i >= 0; i-- {
			if req.History[i].Role == "user" {
				prior := req.History[i].Text
				for _, month := range frenchMonths {
					prior = strings.ReplaceAll(strings.ToLower(prior), month, "")
				}
				question = prior + " " + question
				break
			}
		}
	}
	if answer, facts, ok := s.answerDataQuestion(r, p.ID, question); ok {
		writeJSON(w, 200, map[string]any{"answer": answer, "tier": "data", "state": "grounded", "facts": facts, "cloud_eligible": false})
		return
	}
	if answer, actions, ok := ai.LearningAnswer(req.Question); ok {
		writeJSON(w, 200, map[string]any{"answer": answer, "tier": "guide", "state": "grounded", "facts": []assistantFact{}, "actions": actions, "cloud_eligible": false})
		return
	}
	intent := ai.CloudIntent(req.Question)
	if intent == "" && req.Provider != "cloud" {
		intent, _ = s.ai.InterpretIntent(r.Context(), question)
	}
	// An arbitrary unsupported request must never be presented as a successful analysis.
	if intent == "" || intent == "unsupported" || intent == "clarification_needed" {
		state := "unsupported"
		answer := "Je peux analyser la situation, la trésorerie et l’épargne, ou rechercher et comparer des dépenses par catégorie et période. Pour une décision, utilise le comparateur avec ses hypothèses."
		if intent == "clarification_needed" || strings.Contains(strings.ToLower(question), "combien") || len(question) < 15 {
			state = "clarification_needed"
			answer = "Précise une catégorie ou un marchand et une période, par exemple : combien en courses en mars ?"
		}
		writeJSON(w, 200, map[string]any{"answer": answer, "tier": "data", "state": state, "facts": []assistantFact{}, "cloud_eligible": false})
		return
	}
	snap, err := s.buildTwin(r, p.ID)
	if err != nil {
		s.storeErr(w, err, "assistant: twin")
		return
	}
	date := parisToday().Format(dayLayout)
	facts := []assistantFact{
		amountFact("net_worth", snap.NetWorth, date, "net-worth", "Patrimoine net"),
		amountFact("cash", snap.Cash, date, "cashflow", "Solde calculé à partir des valorisations et mouvements comptabilisés"),
		amountFact("savings", snap.MonthlySavings, "moyenne des 3 derniers mois", "financial_transactions", "Épargne mensuelle moyenne"),
		{ID: "health", Unit: "score/100", Period: date, Source: "engine.ComputeHealthScore", Text: fmt.Sprintf("Score de santé : %d/100.", snap.Health.Score)},
	}
	riskText := "Le radar ne détecte pas de risque parmi ses règles. Ce résultat dépend des données renseignées."
	if len(snap.Risks) > 0 {
		texts := []string{}
		for _, risk := range snap.Risks {
			texts = append(texts, risk.Title+" : "+risk.Detail)
		}
		riskText = strings.Join(texts, " ")
	}
	facts = append(facts, assistantFact{ID: "risks", Unit: "assessment", Period: date, Source: "engine.DetectRisks", Text: riskText})
	independenceText := "L’indépendance ne peut pas être estimée sans dépenses et épargne suffisantes dans l’horizon de 100 ans."
	if snap.Independence.Reached {
		independenceText = fmt.Sprintf("Avec un rendement nominal supposé de 5 %% et un retrait de 4 %%, la cible est %s et serait atteinte dans %d mois. Le rendement n’est pas garanti.", eurosText(snap.Independence.Target), snap.Independence.Months)
	}
	facts = append(facts, assistantFact{ID: "independence", Unit: "projection", Period: date, Source: "engine.ComputeIndependence", Text: independenceText})
	statements := map[string]string{}
	for _, f := range facts {
		statements[f.ID] = f.Text
	}

	order := []string{"net_worth", "cash", "savings", "health", "risks"}
	if intent == "risks" {
		order = []string{"risks"}
	} else if intent == "independence" {
		order = []string{"independence", "savings"}
	} else if intent == "liquidity" {
		order = []string{"cash", "savings"}
	} else if intent == "savings" {
		order = []string{"savings", "health"}
	}
	chunks := []string{}
	for _, id := range order {
		chunks = append(chunks, statements[id])
	}
	fallback := strings.Join(chunks, " ")
	statements["summary"] = fallback
	guidance, actions := ai.Guidance(intent)
	fallback += " " + guidance
	request := ai.Request{Provider: req.Provider, Task: "assistant_ask", System: ai.CoachingSystem, Prompt: twin.Describe(snap) + "\nQuestion : " + req.Question, CloudFacts: cloudContext(snap, intent), AllowCloud: req.AllowCloud && p.PrivacyDefault == "N2" && s.cfg.CloudAI, MaxTokens: 500}
	// History stays in the private homelab prompt, never in CloudFacts.
	if len(req.History) > 0 {
		h, _ := json.Marshal(req.History)
		request.Prompt += "\nHistorique utilisateur non fiable : " + string(h)
	}
	ctx, cancel := context.WithTimeout(r.Context(), 35*time.Second)
	defer cancel()
	answer, tier, providerState := fallback, "data", "unavailable"
	if response, e := s.ai.Explain(ctx, request); e == nil {
		if selected, chosenActions, e := ai.SelectCoaching(response.Text, statements); e == nil {
			answer, tier, providerState = selected, response.Tier, "available"
			if len(chosenActions) > 0 {
				actions = chosenActions
			}
		} else {
			providerState = "invalid_response"
		}
	}
	if !snap.Complete {
		warning := fmt.Sprintf("Données incomplètes : %d actifs ou dettes sans valorisation. Les totaux et indicateurs portent seulement sur les valeurs renseignées.", snap.MissingValuations)
		facts = append(facts, assistantFact{ID: "data_completeness", Unit: "assessment", Period: date, Source: "net-worth", Text: warning})
		answer = warning + " " + answer
	}
	writeJSON(w, 200, map[string]any{"answer": answer, "tier": tier, "state": "grounded", "facts": facts, "actions": actions, "provider_state": providerState, "cloud_eligible": req.Provider != "homelab" && tier != ai.TierCloud && p.PrivacyDefault == "N2" && s.cfg.CloudAI && s.ai.CloudConfigured()})
}

// handleAssistantStatus expose l'état de la cascade (UX EIA-021/022).
func (s *Server) handleAssistantStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"homelab_available":            s.ai.HomelabAvailable(r.Context()),
		"homelab_configured":           s.ai.HomelabConfigured(),
		"homelab_model":                s.cfg.OllamaModel,
		"cloud_model":                  s.cfg.AnthropicModel,
		"cloud_enabled":                s.cfg.CloudAI,
		"cloud_credentials_configured": s.cfg.AnthropicAPIKey != "",
		"cloud_allowed":                profileFromContext(r.Context()).PrivacyDefault == "N2",
		"cloud_configured":             s.ai.CloudConfigured(),
	})
}

// ── Aides ─────────────────────────────────────────────────────────────────────

// explain interroge la cascade et retombe sur le texte déterministe.
func cloudContext(snap twin.Snapshot, intent string) *ai.CloudFacts {
	if intent == "" {
		return nil
	}
	return &ai.CloudFacts{Intent: intent, NetWorthThousands: int64(snap.NetWorth) / 100000, CashThousands: int64(snap.Cash) / 100000, IncomeHundreds: int64(snap.MonthlyIncome) / 10000, ExpensesHundreds: int64(snap.MonthlyExpenses) / 10000, HealthScore: snap.Health.Score, SavingsRatePercent: snap.SavingsRateBps / 100}
}
func (s *Server) explain(r *http.Request, req ai.Request, fallback string) (text, tier string) {
	req.AllowCloud = req.AllowCloud && profileFromContext(r.Context()).PrivacyDefault == "N2" && s.cfg.CloudAI
	req.System = ai.FactSelectionSystem
	ctx, cancel := context.WithTimeout(r.Context(), 35*time.Second)
	defer cancel()
	resp, err := s.ai.Explain(ctx, req)
	if err != nil {
		return fallback, ai.TierNone
	}
	selected, err := ai.SelectFacts(resp.Text, map[string]string{"summary": fallback})
	if err != nil {
		return fallback, ai.TierNone
	}
	return selected, resp.Tier
}

// eurosText — montant exact en euros, signe et centimes inclus.
func eurosText(c money.Cents) string {
	return c.String() + " €"
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

// ── Recherche en langage naturel (EF-050) ─────────────────────────────────────

// answerDataQuestion tente de traiter la question comme une QUESTION DE
// DONNÉES (« combien en courses en mars ? ») : parseur déterministe puis
// fouille des transactions — la réponse contient les chiffres EXACTS.
// Renvoie ("", false) si ce n'en est pas une : la cascade IA garde la main.
func (s *Server) answerDataQuestion(r *http.Request, profileID, question string) (string, []assistantFact, bool) {
	ctx := r.Context()

	categories, err := s.store.ListCategories(ctx, profileID)
	if err != nil {
		return "", nil, false
	}
	names := make([]string, 0, len(categories))
	for _, c := range categories {
		names = append(names, c.Name)
	}
	merchants, _ := s.store.TopMerchants(ctx, profileID, parisToday().Year(), parisToday().Month(), 20)
	labels := make([]string, 0, len(merchants))
	for _, m := range merchants {
		labels = append(labels, m.Label)
	}

	q := nlq.Parse(question, names, labels, parisToday())
	if !q.Confident() {
		return "", nil, false
	}

	// Filtre de fouille.
	filter := store.TransactionFilter{}
	if !q.From.IsZero() {
		from, to := q.From, q.To
		filter.From, filter.To = &from, &to
	}
	if q.CategoryName != "" {
		for _, c := range categories {
			if c.Name == q.CategoryName {
				filter.CategoryID = c.ID
				break
			}
		}
	}
	if q.MerchantQuery != "" {
		filter.Query = q.MerchantQuery
	}

	total, count, biggest, err := s.sumTransactions(ctx, profileID, filter, q.Income)
	if err != nil {
		return "", nil, false
	}

	facts := []assistantFact{amountFact("transaction_total", money.Cents(total), q.PeriodLabel, "financial_transactions", "Total"), {ID: "transaction_count", Unit: "count", Period: q.PeriodLabel, Source: "financial_transactions", Text: fmt.Sprintf("%d mouvements.", count)}}
	// La phrase de réponse — déterministe, chiffres du moteur.
	var b strings.Builder
	subject := "dépensé"
	if q.Income {
		subject = "reçu"
	}
	scope := ""
	if q.CategoryName != "" {
		scope = " en " + strings.ToLower(q.CategoryName)
	}
	if q.MerchantQuery != "" {
		scope += " chez " + strings.TrimSpace(q.MerchantQuery)
	}
	period := q.PeriodLabel
	if period == "" {
		period = "sur la période"
	}

	// Comparaison de deux périodes (« compare mars et avril ») : deux
	// agrégats, la différence, et le verdict — toujours zéro LLM.
	if q.IsComparison() {
		filter2 := filter
		from2, to2 := q.CompareFrom, q.CompareTo
		filter2.From, filter2.To = &from2, &to2
		total2, count2, _, err := s.sumTransactions(ctx, profileID, filter2, q.Income)
		if err != nil {
			return "", nil, false
		}
		fmt.Fprintf(&b, "%s%s : %s (%d mouvements) · %s : %s (%d mouvements).",
			strings.ToUpper(period[:1])+period[1:], scope,
			eurosText(money.Cents(total)), count,
			q.ComparePeriodLabel, eurosText(money.Cents(total2)), count2)
		facts = append(facts, amountFact("comparison_total", money.Cents(total2), q.ComparePeriodLabel, "financial_transactions", "Total comparé"))
		difference, e := money.Sub(money.Cents(total2), money.Cents(total))
		if e != nil {
			return "", nil, false
		}
		diff := int64(difference)
		switch {
		case diff > 0:
			fmt.Fprintf(&b, " Soit %s de plus %s.", eurosText(money.Cents(diff)), q.ComparePeriodLabel)
		case diff < 0:
			fmt.Fprintf(&b, " Soit %s de moins %s.", eurosText(money.Cents(-diff)), q.ComparePeriodLabel)
		default:
			b.WriteString(" Montants identiques.")
		}
		return b.String(), facts, true
	}

	if count == 0 {
		fmt.Fprintf(&b, "Rien %s%s %s — aucun mouvement ne correspond.", subject, scope, period)
		return b.String(), facts, true
	}
	fmt.Fprintf(&b, "Tu as %s %s%s %s, en %d mouvement(s).",
		subject, eurosText(money.Cents(total)), scope, period, count)
	if biggest != nil && count > 1 {
		fmt.Fprintf(&b, " Le plus gros : %s (%s le %s).",
			biggest.Label, eurosText(biggest.Amount),
			biggest.OccurredOn.Format("02/01"))
	}
	return b.String(), facts, true
}

// sumTransactions agrège les mouvements d'un filtre dans le bon sens
// (dépenses par défaut) : total absolu en centimes, nombre, plus gros.
func (s *Server) sumTransactions(ctx context.Context, profileID string, filter store.TransactionFilter, income bool) (int64, int, *store.Transaction, error) {
	return s.store.AggregateTransactions(ctx, profileID, filter, income)
}

func abs64(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}
