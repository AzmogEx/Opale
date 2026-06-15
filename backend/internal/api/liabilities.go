package api

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/opale-app/opale/internal/money"
	"github.com/opale-app/opale/internal/store"
)

func (s *Server) handleCreateLiability(w http.ResponseWriter, r *http.Request) {
	var req assetRequest // même forme : name, kind, currency, note
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "corps JSON invalide")
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "invalid_name", "le nom est requis")
		return
	}
	if !store.LiabilityKinds[req.Kind] {
		writeError(w, http.StatusBadRequest, "invalid_kind", "type de passif inconnu")
		return
	}
	currency := normalizeCurrency(req.Currency)
	if currency == "" {
		writeError(w, http.StatusBadRequest, "invalid_currency", "devise invalide (code à 3 lettres)")
		return
	}

	p := profileFromContext(r.Context())
	liab, err := s.store.CreateLiability(r.Context(), p.ID, req.Name, req.Kind, currency, req.Note)
	if err != nil {
		s.storeErr(w, err, "create liability")
		return
	}
	writeJSON(w, http.StatusCreated, liab)
}

func (s *Server) handleListLiabilities(w http.ResponseWriter, r *http.Request) {
	p := profileFromContext(r.Context())
	liabs, err := s.store.ListLiabilities(r.Context(), p.ID)
	if err != nil {
		s.storeErr(w, err, "list liabilities")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"liabilities": liabs})
}

func (s *Server) handleGetLiability(w http.ResponseWriter, r *http.Request) {
	p := profileFromContext(r.Context())
	liab, err := s.store.GetLiability(r.Context(), p.ID, chi.URLParam(r, "id"))
	if err != nil {
		s.storeErr(w, err, "get liability")
		return
	}
	writeJSON(w, http.StatusOK, liab)
}

func (s *Server) handleUpdateLiability(w http.ResponseWriter, r *http.Request) {
	var req updateAssetRequest // name, note, archived
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "corps JSON invalide")
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "invalid_name", "le nom est requis")
		return
	}
	p := profileFromContext(r.Context())
	liab, err := s.store.UpdateLiability(r.Context(), p.ID, chi.URLParam(r, "id"), req.Name, req.Note, req.Archived)
	if err != nil {
		s.storeErr(w, err, "update liability")
		return
	}
	writeJSON(w, http.StatusOK, liab)
}

func (s *Server) handleDeleteLiability(w http.ResponseWriter, r *http.Request) {
	p := profileFromContext(r.Context())
	if err := s.store.DeleteLiability(r.Context(), p.ID, chi.URLParam(r, "id")); err != nil {
		s.storeErr(w, err, "delete liability")
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}

func (s *Server) handleAddLiabilityValuation(w http.ResponseWriter, r *http.Request) {
	req, asOf, ok := s.parseValuation(w, r)
	if !ok {
		return
	}
	p := profileFromContext(r.Context())
	v, err := s.store.AddLiabilityValuation(r.Context(), p.ID, chi.URLParam(r, "id"), money.Cents(req.ValueCents), asOf, req.Note)
	if err != nil {
		s.storeErr(w, err, "add liability valuation")
		return
	}
	writeJSON(w, http.StatusCreated, v)
}

func (s *Server) handleListLiabilityValuations(w http.ResponseWriter, r *http.Request) {
	p := profileFromContext(r.Context())
	vals, err := s.store.ListLiabilityValuations(r.Context(), p.ID, chi.URLParam(r, "id"))
	if err != nil {
		s.storeErr(w, err, "list liability valuations")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"valuations": vals})
}
