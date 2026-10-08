package api

import (
	"github.com/go-chi/chi/v5"
	"github.com/opale-app/opale/internal/store"
	"net/http"
	"strings"
)

func (s *Server) handleUpdateContact(w http.ResponseWriter, r *http.Request) {
	var req contactRequest
	if e := decodeJSON(r, &req); e != nil || strings.TrimSpace(req.Name) == "" || !store.ContactRoles[req.Role] {
		writeError(w, 400, "invalid_contact", "Nom et rôle valides requis")
		return
	}
	c, e := s.store.UpdateContact(r.Context(), profileFromContext(r.Context()).ID, chi.URLParam(r, "id"), store.Contact{Name: strings.TrimSpace(req.Name), Role: req.Role, Phone: req.Phone, Email: req.Email, Note: req.Note})
	if e != nil {
		s.storeErr(w, e, "update contact")
		return
	}
	writeJSON(w, 200, c)
}
func (s *Server) handleUpdateDocument(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name    string `json:"name"`
		Kind    string `json:"kind"`
		AssetID string `json:"asset_id"`
	}
	if e := decodeJSON(r, &req); e != nil || strings.TrimSpace(req.Name) == "" || !store.DocumentKinds[req.Kind] {
		writeError(w, 400, "invalid_document", "Nom et type valides requis")
		return
	}
	var asset *string
	if req.AssetID != "" {
		asset = &req.AssetID
	}
	if e := s.store.UpdateDocument(r.Context(), profileFromContext(r.Context()).ID, chi.URLParam(r, "id"), strings.TrimSpace(req.Name), req.Kind, asset); e != nil {
		s.storeErr(w, e, "update document")
		return
	}
	writeJSON(w, 200, map[string]bool{"saved": true})
}
