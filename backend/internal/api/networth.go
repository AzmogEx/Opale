package api

import (
	"net/http"
	"strconv"
)

// handleNetWorth renvoie le patrimoine net du profil (CA-1), calculé de manière
// déterministe par le store (jamais par l'IA — EIA-040/041).
func (s *Server) handleNetWorth(w http.ResponseWriter, r *http.Request) {
	p := profileFromContext(r.Context())
	nw, err := s.store.ComputeNetWorth(r.Context(), p.ID)
	if err != nil {
		s.storeErr(w, err, "compute net worth")
		return
	}
	writeJSON(w, http.StatusOK, nw)
}

// handleNetWorthHistory renvoie la série temporelle du patrimoine net (EF-013),
// un point par fin de mois. Paramètre ?months=N (défaut 12, borné à [1, 120]).
func (s *Server) handleNetWorthHistory(w http.ResponseWriter, r *http.Request) {
	months := 12
	if raw := r.URL.Query().Get("months"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_months", "months doit être un entier")
			return
		}
		months = n
	}
	if months < 1 {
		months = 1
	}
	if months > 120 {
		months = 120
	}

	p := profileFromContext(r.Context())
	h, err := s.store.ComputeNetWorthHistory(r.Context(), p.ID, months)
	if err != nil {
		s.storeErr(w, err, "compute net worth history")
		return
	}
	writeJSON(w, http.StatusOK, h)
}
