package api

import (
	"github.com/opale-app/opale/internal/store"
	"net/http"
)

func (s *Server) handleJourney(w http.ResponseWriter, r *http.Request) {
	j, e := s.store.FinancialJourney(r.Context(), profileFromContext(r.Context()).ID)
	if e != nil {
		s.financialError(w, e)
		return
	}
	writeJSON(w, 200, j)
}
func (s *Server) handleSaveJourney(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 8<<10)
	var j store.FinancialJourney
	if e := decodeJSON(r, &j); e != nil {
		writeError(w, 400, "invalid_body", "Parcours invalide")
		return
	}
	j, e := s.store.SaveFinancialJourney(r.Context(), profileFromContext(r.Context()).ID, j)
	if e != nil {
		s.financialError(w, e)
		return
	}
	writeJSON(w, 200, j)
}
