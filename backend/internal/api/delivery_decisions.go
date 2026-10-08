package api

import (
	"github.com/opale-app/opale/internal/engine"
	"net/http"
)

func (s *Server) handleDecisionCompare(w http.ResponseWriter, r *http.Request) {
	var in engine.ComparisonInput
	if e := decodeJSON(r, &in); e != nil {
		writeError(w, 400, "invalid_body", "Hypothèses de comparaison invalides")
		return
	}
	out, e := engine.CompareDecisions(in)
	if e != nil {
		writeError(w, 400, "invalid_comparison", "Vérifie les montants, taux, durée (1–600 mois) et financement")
		return
	}
	writeJSON(w, 200, out)
}
