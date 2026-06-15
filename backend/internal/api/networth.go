package api

import "net/http"

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
