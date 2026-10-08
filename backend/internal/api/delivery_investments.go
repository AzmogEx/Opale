package api

import (
	"github.com/go-chi/chi/v5"
	"github.com/opale-app/opale/internal/store"
	"net/http"
	"time"
)

func (s *Server) handleInvestmentDetail(w http.ResponseWriter, r *http.Request) {
	d, e := s.store.InvestmentDetail(r.Context(), profileFromContext(r.Context()).ID, chi.URLParam(r, "id"))
	if e != nil {
		s.storeErr(w, e, "investment detail")
		return
	}
	writeJSON(w, 200, d)
}
func (s *Server) handleSaveInvestmentFlow(w http.ResponseWriter, r *http.Request) {
	var f store.InvestmentFlow
	e := decodeJSON(r, &f)
	f.AssetID = chi.URLParam(r, "id")
	f.ID = chi.URLParam(r, "flowID")
	d, de := time.Parse(dayLayout, f.Date)
	valid := map[string]bool{"contribution": true, "withdrawal": true, "distribution": true, "fee": true}
	if e != nil || de != nil || d.After(parisToday()) || d.Year() < 1900 || !valid[f.Kind] || f.Amount <= 0 {
		writeError(w, 400, "invalid_flow", "Type, montant positif et date passée valides requis")
		return
	}
	saved, e := s.store.SaveInvestmentFlow(r.Context(), profileFromContext(r.Context()).ID, f)
	if e != nil {
		s.storeErr(w, e, "save investment flow")
		return
	}
	writeJSON(w, 200, saved)
}
func (s *Server) handleDeleteInvestmentFlow(w http.ResponseWriter, r *http.Request) {
	e := s.store.DeleteInvestmentFlow(r.Context(), profileFromContext(r.Context()).ID, chi.URLParam(r, "id"), chi.URLParam(r, "flowID"))
	if e != nil {
		s.storeErr(w, e, "delete investment flow")
		return
	}
	writeJSON(w, 204, nil)
}
func (s *Server) handleInvestmentCoverage(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Complete bool `json:"complete"`
	}
	if e := decodeJSON(r, &req); e != nil {
		writeError(w, 400, "invalid_body", "complete requis")
		return
	}
	if e := s.store.SetInvestmentCoverage(r.Context(), profileFromContext(r.Context()).ID, chi.URLParam(r, "id"), req.Complete); e != nil {
		s.storeErr(w, e, "investment coverage")
		return
	}
	writeJSON(w, 200, map[string]bool{"saved": true})
}
