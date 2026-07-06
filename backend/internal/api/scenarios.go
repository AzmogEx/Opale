package api

// Comparateur de scénarios (EF-044) : deux futurs côte à côte.
// Chaque scénario est projeté par le moteur déterministe (mêmes règles que
// /v1/projection) ; l'API renvoie les deux courbes, les deux dates
// d'indépendance et les écarts à 5/10/20 ans. Aucune IA ici : que du moteur.

import (
	"net/http"

	"github.com/opale-app/opale/internal/engine"
	"github.com/opale-app/opale/internal/money"
)

// scenarioRequest — un futur possible, décrit par ses hypothèses.
type scenarioRequest struct {
	Label string `json:"label"`
	// Hypothèses mensuelles (centimes / bps).
	MonthlySavings  int64 `json:"monthly_savings_cents"`
	MonthlyExpenses int64 `json:"monthly_expenses_cents"`
	AnnualReturnBps int   `json:"annual_return_bps"`
	// Mouvement immédiat optionnel (achat = positif, apport reçu = négatif).
	OneTimeCost int64 `json:"one_time_cost_cents"`
}

type compareRequest struct {
	A scenarioRequest `json:"a"`
	B scenarioRequest `json:"b"`
	// Horizon de la comparaison en mois (défaut 240 = 20 ans, max 360).
	Months int `json:"months"`
}

// scenarioResult — la trajectoire complète d'un scénario.
type scenarioResult struct {
	Label        string                   `json:"label"`
	Points       []engine.ProjectionPoint `json:"points"`
	Independence engine.Independence      `json:"independence"`
	At5y         money.Cents              `json:"at_5y_cents"`
	At10y        money.Cents              `json:"at_10y_cents"`
	AtEnd        money.Cents              `json:"at_end_cents"`
}

func (s *Server) handleCompareScenarios(w http.ResponseWriter, r *http.Request) {
	p := profileFromContext(r.Context())
	var req compareRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}
	if req.Months <= 0 {
		req.Months = 240
	}
	if req.Months > 360 {
		req.Months = 360
	}

	// Point de départ commun : le patrimoine net réel du profil.
	nw, err := s.store.ComputeNetWorth(r.Context(), p.ID)
	if err != nil {
		s.storeErr(w, err, "compare: net worth")
		return
	}

	run := func(sc scenarioRequest, fallbackLabel string) (scenarioResult, bool) {
		if sc.Label == "" {
			sc.Label = fallbackLabel
		}
		if sc.AnnualReturnBps == 0 {
			sc.AnnualReturnBps = twinReturnBps
		}
		start := nw.Net - money.Cents(sc.OneTimeCost)
		points, err := engine.Project(start, money.Cents(sc.MonthlySavings), sc.AnnualReturnBps, req.Months)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_scenario",
				"scénario « "+sc.Label+" » : "+err.Error())
			return scenarioResult{}, false
		}
		res := scenarioResult{
			Label:  sc.Label,
			Points: points,
			At5y:   netAtMonth(points, 60),
			At10y:  netAtMonth(points, 120),
			AtEnd:  points[len(points)-1].Net,
		}
		if sc.MonthlyExpenses > 0 {
			if ind, err := engine.ComputeIndependence(start, money.Cents(sc.MonthlySavings),
				money.Cents(sc.MonthlyExpenses), sc.AnnualReturnBps, twinSwrBps); err == nil {
				res.Independence = ind
			}
		}
		return res, true
	}

	a, ok := run(req.A, "Scénario A")
	if !ok {
		return
	}
	b, ok := run(req.B, "Scénario B")
	if !ok {
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"start_cents": nw.Net,
		"months":      req.Months,
		"a":           a,
		"b":           b,
		// Écarts B − A (positif = B gagne).
		"delta_5y_cents":  b.At5y - a.At5y,
		"delta_10y_cents": b.At10y - a.At10y,
		"delta_end_cents": b.AtEnd - a.AtEnd,
	})
}

// netAtMonth — patrimoine au mois donné (dernier point si horizon plus court).
func netAtMonth(points []engine.ProjectionPoint, month int) money.Cents {
	if month < len(points) {
		return points[month].Net
	}
	return points[len(points)-1].Net
}
