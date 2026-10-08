package api

import (
	"github.com/go-chi/chi/v5"
	"github.com/opale-app/opale/internal/auth"
	"net/http"
	"strings"
	"time"
)

func (s *Server) handleSaveCategory(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
		Icon string `json:"icon"`
	}
	if decodeJSON(r, &req) != nil || strings.TrimSpace(req.Name) == "" {
		writeError(w, 400, "invalid_body", "nom requis")
		return
	}
	if req.Icon == "" {
		req.Icon = "tag"
	}
	v, e := s.store.SaveCategory(r.Context(), profileFromContext(r.Context()).ID, chi.URLParam(r, "id"), strings.TrimSpace(req.Name), req.Icon)
	if e != nil {
		s.storeErr(w, e, "save category")
		return
	}
	writeJSON(w, 200, v)
}
func (s *Server) handleDeleteCategory(w http.ResponseWriter, r *http.Request) {
	if e := s.store.DeleteCategory(r.Context(), profileFromContext(r.Context()).ID, chi.URLParam(r, "id")); e != nil {
		s.storeErr(w, e, "delete category")
		return
	}
	writeJSON(w, 204, nil)
}
func (s *Server) handleListRules(w http.ResponseWriter, r *http.Request) {
	v, e := s.store.ListRules(r.Context(), profileFromContext(r.Context()).ID)
	if e != nil {
		s.storeErr(w, e, "rules")
		return
	}
	writeJSON(w, 200, map[string]any{"rules": v})
}
func (s *Server) handleSaveRule(w http.ResponseWriter, r *http.Request) {
	var req struct {
		MerchantKey string `json:"merchant_key"`
		CategoryID  string `json:"category_id"`
	}
	if decodeJSON(r, &req) != nil || strings.TrimSpace(req.MerchantKey) == "" || req.CategoryID == "" {
		writeError(w, 400, "invalid_body", "marchand et catégorie requis")
		return
	}
	v, e := s.store.SaveRule(r.Context(), profileFromContext(r.Context()).ID, chi.URLParam(r, "id"), strings.TrimSpace(req.MerchantKey), req.CategoryID)
	if e != nil {
		s.storeErr(w, e, "save rule")
		return
	}
	writeJSON(w, 200, v)
}
func (s *Server) handleDeleteRule(w http.ResponseWriter, r *http.Request) {
	if e := s.store.DeleteRule(r.Context(), profileFromContext(r.Context()).ID, chi.URLParam(r, "id")); e != nil {
		s.storeErr(w, e, "delete rule")
		return
	}
	writeJSON(w, 204, nil)
}
func (s *Server) handleUpdateMe(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name    *string `json:"name"`
		PIN     *string `json:"pin"`
		Privacy *string `json:"privacy_default"`
	}
	if decodeJSON(r, &req) != nil {
		writeError(w, 400, "invalid_body", "corps invalide")
		return
	}
	if req.Name != nil && strings.TrimSpace(*req.Name) == "" {
		writeError(w, 400, "invalid_name", "nom requis")
		return
	}
	if req.Privacy != nil && *req.Privacy != "N1" && *req.Privacy != "N2" && *req.Privacy != "N3" {
		writeError(w, 400, "invalid_privacy", "N1 local, N2 cloud minimisé possible ou N3 local strict")
		return
	}
	var hash *string
	if req.PIN != nil {
		if len(*req.PIN) < 4 || len(*req.PIN) > 72 {
			writeError(w, 400, "invalid_pin", "code de 4 à 72 caractères")
			return
		}
		v, e := auth.HashPIN(*req.PIN)
		if e != nil {
			s.storeErr(w, e, "hash")
			return
		}
		hash = &v
	}
	p, e := s.store.UpdateProfile(r.Context(), profileFromContext(r.Context()).ID, req.Name, hash, req.Privacy)
	if e != nil {
		s.storeErr(w, e, "profile update")
		return
	}
	writeJSON(w, 200, p)
}
func (s *Server) handleUpdateValuation(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Value *int64  `json:"value_cents"`
		Day   *string `json:"as_of"`
		Note  *string `json:"note"`
	}
	if decodeJSON(r, &req) != nil {
		writeError(w, 400, "invalid_body", "corps invalide")
		return
	}
	var day *time.Time
	if req.Day != nil {
		v, e := time.Parse(dayLayout, *req.Day)
		if e != nil {
			writeError(w, 400, "invalid_date", "date civile AAAA-MM-JJ requise")
			return
		}
		day = &v
	}
	v, e := s.store.UpdateValuation(r.Context(), profileFromContext(r.Context()).ID, chi.URLParam(r, "id"), req.Value, day, req.Note)
	if e != nil {
		s.storeErr(w, e, "valuation update")
		return
	}
	writeJSON(w, 200, v)
}
func (s *Server) handleDeleteValuation(w http.ResponseWriter, r *http.Request) {
	if e := s.store.DeleteValuation(r.Context(), profileFromContext(r.Context()).ID, chi.URLParam(r, "id")); e != nil {
		s.storeErr(w, e, "valuation delete")
		return
	}
	writeJSON(w, 204, nil)
}
func (s *Server) handlePushUnregister(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token string `json:"token"`
	}
	if decodeJSON(r, &req) != nil || req.Token == "" {
		writeError(w, 400, "invalid_body", "token requis")
		return
	}
	if e := s.store.DeleteProfilePushToken(r.Context(), profileFromContext(r.Context()).ID, req.Token); e != nil {
		s.storeErr(w, e, "push unregister")
		return
	}
	writeJSON(w, 204, nil)
}

func (s *Server) handleQuoteMetadata(w http.ResponseWriter, r *http.Request) {
	v, e := s.store.QuoteMetadata(r.Context(), profileFromContext(r.Context()).ID, chi.URLParam(r, "id"))
	if e != nil {
		s.storeErr(w, e, "quote metadata")
		return
	}
	writeJSON(w, 200, v)
}
