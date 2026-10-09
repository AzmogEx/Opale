package api

import (
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/opale-app/opale/internal/store"
	"net/http"
	"strconv"
)

func (s *Server) financialError(w http.ResponseWriter, e error) {
	if errors.Is(e, store.ErrFinancialConflict) {
		writeError(w, 409, "financial_conflict", "Cet élément a changé. Recharge la liste avant de réessayer.")
		return
	}
	s.onboardingError(w, e)
}
func (s *Server) handleContracts(w http.ResponseWriter, r *http.Request) {
	p := profileFromContext(r.Context()).ID
	items, e := s.store.ListFinancialContracts(r.Context(), p)
	if e != nil {
		s.financialError(w, e)
		return
	}
	writeJSON(w, 200, map[string]any{"contracts": items, "alerts": store.FinancialContractAlerts(items, parisToday())})
}
func (s *Server) handleSaveContract(w http.ResponseWriter, r *http.Request) {
	var c store.FinancialContract
	r.Body = http.MaxBytesReader(w, r.Body, 32<<10)
	if e := decodeJSON(r, &c); e != nil {
		writeError(w, 400, "invalid_body", "Contrat invalide")
		return
	}
	c.ID = chi.URLParam(r, "id")
	saved, e := s.store.SaveFinancialContract(r.Context(), profileFromContext(r.Context()).ID, c)
	if e != nil {
		s.financialError(w, e)
		return
	}
	writeJSON(w, 200, saved)
}
func financialRevision(w http.ResponseWriter, r *http.Request) (int64, bool) {
	if !requestUUID.MatchString(chi.URLParam(r, "id")) {
		writeError(w, 400, "invalid_id", "Identifiant invalide")
		return 0, false
	}
	rev, e := strconv.ParseInt(r.URL.Query().Get("revision"), 10, 64)
	if e != nil || rev < 1 {
		writeError(w, 400, "invalid_revision", "Révision requise")
		return 0, false
	}
	return rev, true
}
func (s *Server) handleDeleteContract(w http.ResponseWriter, r *http.Request) {
	rev, ok := financialRevision(w, r)
	if !ok {
		return
	}
	e := s.store.DeleteFinancialContract(r.Context(), profileFromContext(r.Context()).ID, chi.URLParam(r, "id"), rev)
	if e != nil {
		s.financialError(w, e)
		return
	}
	writeJSON(w, 204, nil)
}
func (s *Server) handleContractPrices(w http.ResponseWriter, r *http.Request) {
	if !requestUUID.MatchString(chi.URLParam(r, "id")) {
		writeError(w, 400, "invalid_id", "Identifiant invalide")
		return
	}
	prices, e := s.store.ContractPrices(r.Context(), profileFromContext(r.Context()).ID, chi.URLParam(r, "id"))
	if e != nil {
		s.financialError(w, e)
		return
	}
	writeJSON(w, 200, map[string]any{"prices": prices})
}
func (s *Server) handleResolveContractPrice(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Revision      int64  `json:"revision"`
		TransactionID string `json:"transaction_id"`
		Accept        bool   `json:"accept"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	if e := decodeJSON(r, &body); e != nil || !requestUUID.MatchString(body.TransactionID) || !requestUUID.MatchString(chi.URLParam(r, "id")) || body.Revision < 1 {
		writeError(w, 400, "invalid_body", "Révision et opération requises")
		return
	}
	e := s.store.ResolveContractPrice(r.Context(), profileFromContext(r.Context()).ID, chi.URLParam(r, "id"), body.TransactionID, body.Revision, body.Accept)
	if e != nil {
		s.financialError(w, e)
		return
	}
	writeJSON(w, 200, map[string]bool{"saved": true})
}
func (s *Server) handleVariableIncomes(w http.ResponseWriter, r *http.Request) {
	items, e := s.store.ListVariableIncomes(r.Context(), profileFromContext(r.Context()).ID)
	if e != nil {
		s.financialError(w, e)
		return
	}
	writeJSON(w, 200, map[string]any{"incomes": items})
}
func (s *Server) handleSaveVariableIncome(w http.ResponseWriter, r *http.Request) {
	var v store.VariableIncome
	r.Body = http.MaxBytesReader(w, r.Body, 32<<10)
	if e := decodeJSON(r, &v); e != nil {
		writeError(w, 400, "invalid_body", "Revenu invalide")
		return
	}
	v.ID = chi.URLParam(r, "id")
	saved, e := s.store.SaveVariableIncome(r.Context(), profileFromContext(r.Context()).ID, v)
	if e != nil {
		s.financialError(w, e)
		return
	}
	writeJSON(w, 200, saved)
}
func (s *Server) handleDeleteVariableIncome(w http.ResponseWriter, r *http.Request) {
	rev, ok := financialRevision(w, r)
	if !ok {
		return
	}
	e := s.store.DeleteVariableIncome(r.Context(), profileFromContext(r.Context()).ID, chi.URLParam(r, "id"), rev)
	if e != nil {
		s.financialError(w, e)
		return
	}
	writeJSON(w, 204, nil)
}
