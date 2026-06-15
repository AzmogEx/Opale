package api

import "net/http"

// handleHealthz : le service répond (liveness).
func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// handleReadyz : le service et la base sont prêts (readiness).
func (s *Server) handleReadyz(w http.ResponseWriter, r *http.Request) {
	if err := s.store.Ping(r.Context()); err != nil {
		writeError(w, http.StatusServiceUnavailable, "db_unavailable", "base de données indisponible")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}
