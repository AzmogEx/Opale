package api

import (
	"net/http"
	"strconv"

	"github.com/opale-app/opale/internal/engine"
	"github.com/opale-app/opale/internal/money"
)

// projectionResponse — réponse de GET /v1/projection (EF-040/041).
type projectionResponse struct {
	// Situation de départ : patrimoine net actuel (calculé par le store).
	StartNet money.Cents `json:"start_net_cents"`
	// Hypothèses renvoyées telles qu'appliquées (bornées).
	MonthlySavings  money.Cents `json:"monthly_savings_cents"`
	AnnualReturnBps int         `json:"annual_return_bps"`
	MonthlyExpenses money.Cents `json:"monthly_expenses_cents"`
	SWRBps          int         `json:"swr_bps"`
	// Courbe projetée (un point par mois, borné par `months`).
	Points []engine.ProjectionPoint `json:"points"`
	// Indépendance financière (règle du taux de retrait sûr).
	Independence engine.Independence `json:"independence"`
}

// handleProjection projette le patrimoine net (EF-041) et calcule la date
// d'indépendance financière (EF-040). Tous les calculs viennent du moteur
// déterministe `engine` (EIA-040) — jamais de l'IA.
//
// Paramètres (query) :
//
//	monthly_savings_cents   épargne mensuelle (défaut 0)
//	annual_return_bps       rendement annuel en bps (défaut 500 = 5 %)
//	monthly_expenses_cents  dépenses mensuelles (requis pour l'indépendance)
//	months                  horizon de la courbe (défaut 360, max 1200)
//	swr_bps                 taux de retrait sûr (défaut 400 = 4 %)
func (s *Server) handleProjection(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	intParam := func(name string, def int) (int, bool) {
		raw := q.Get(name)
		if raw == "" {
			return def, true
		}
		n, err := strconv.Atoi(raw)
		if err != nil {
			return 0, false
		}
		return n, true
	}

	monthlySavings, ok1 := intParam("monthly_savings_cents", 0)
	annualReturnBps, ok2 := intParam("annual_return_bps", 500)
	monthlyExpenses, ok3 := intParam("monthly_expenses_cents", 0)
	months, ok4 := intParam("months", 360)
	swrBps, ok5 := intParam("swr_bps", 400)
	if !ok1 || !ok2 || !ok3 || !ok4 || !ok5 {
		writeError(w, http.StatusBadRequest, "invalid_param", "paramètre numérique invalide")
		return
	}
	if months < 1 {
		months = 1
	}
	if months > engine.MaxProjectionMonths {
		months = engine.MaxProjectionMonths
	}

	// Point de départ : le patrimoine net actuel, calculé par le store (CA-1).
	p := profileFromContext(r.Context())
	nw, err := s.store.ComputeNetWorth(r.Context(), p.ID)
	if err != nil {
		s.storeErr(w, err, "compute net worth")
		return
	}

	points, err := engine.Project(nw.Net, money.Cents(monthlySavings), annualReturnBps, months)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_projection", err.Error())
		return
	}

	resp := projectionResponse{
		StartNet:        nw.Net,
		MonthlySavings:  money.Cents(monthlySavings),
		AnnualReturnBps: annualReturnBps,
		MonthlyExpenses: money.Cents(monthlyExpenses),
		SWRBps:          swrBps,
		Points:          points,
	}

	// Indépendance financière : seulement si les dépenses sont renseignées.
	if monthlyExpenses > 0 {
		ind, err := engine.ComputeIndependence(
			nw.Net,
			money.Cents(monthlySavings),
			money.Cents(monthlyExpenses),
			annualReturnBps, swrBps,
		)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_independence", err.Error())
			return
		}
		resp.Independence = ind
	}

	writeJSON(w, http.StatusOK, resp)
}
