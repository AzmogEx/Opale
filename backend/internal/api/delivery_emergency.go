package api

import (
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/opale-app/opale/internal/store"
	"mime"
	"net/http"
)

func (s *Server) handleBeneficiaries(w http.ResponseWriter, r *http.Request) {
	b, e := s.store.ListBeneficiaries(r.Context(), profileFromContext(r.Context()).ID)
	if e != nil {
		s.storeErr(w, e, "beneficiaries")
		return
	}
	writeJSON(w, 200, map[string]any{"beneficiaries": b})
}
func (s *Server) handleSaveBeneficiary(w http.ResponseWriter, r *http.Request) {
	var b store.Beneficiary
	if e := decodeJSON(r, &b); e != nil || b.ContactID == "" || b.DocumentID == "" || b.ShareBps <= 0 || b.ShareBps > 10000 {
		writeError(w, 400, "invalid_beneficiary", "Contact, contrat et part (1–10000 bps) requis")
		return
	}
	p := profileFromContext(r.Context())
	out, e := s.store.SaveBeneficiary(r.Context(), p.ID, b)
	if e != nil {
		if errors.Is(e, store.ErrInvalid) {
			writeError(w, 422, "invalid_shares", "Les parts dépassent 100 % pour ce contrat")
			return
		}
		s.storeErr(w, e, "save beneficiary")
		return
	}
	s.journal(r, &p.ID, "beneficiary_saved", b.DocumentID)
	writeJSON(w, 200, out)
}
func (s *Server) handleDeleteBeneficiary(w http.ResponseWriter, r *http.Request) {
	p := profileFromContext(r.Context())
	id := chi.URLParam(r, "id")
	if e := s.store.DeleteBeneficiary(r.Context(), p.ID, id); e != nil {
		s.storeErr(w, e, "delete beneficiary")
		return
	}
	s.journal(r, &p.ID, "beneficiary_deleted", id)
	writeJSON(w, 204, nil)
}
func (s *Server) handleEmergencyGrants(w http.ResponseWriter, r *http.Request) {
	out, e := s.store.ListEmergencyGrants(r.Context(), profileFromContext(r.Context()).ID)
	if e != nil {
		s.storeErr(w, e, "emergency grants")
		return
	}
	writeJSON(w, 200, map[string]any{"grants": out})
}
func (s *Server) handleCreateEmergencyGrant(w http.ResponseWriter, r *http.Request) {
	var g store.EmergencyGrant
	if e := decodeJSON(r, &g); e != nil {
		writeError(w, 400, "invalid_grant", "Droit d’urgence invalide")
		return
	}
	p := profileFromContext(r.Context())
	out, e := s.store.CreateEmergencyGrant(r.Context(), p.ID, g)
	if e != nil {
		if errors.Is(e, store.ErrInvalid) {
			writeError(w, 422, "invalid_grant", "Destinataire distinct, documents/actifs sélectionnés et expiration dans l’année requis")
			return
		}
		s.storeErr(w, e, "create emergency grant")
		return
	}
	s.journal(r, &p.ID, "emergency_grant_created", out.ID)
	writeJSON(w, 201, out)
}
func (s *Server) handleActivateEmergencyGrant(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Active bool `json:"active"`
	}
	if r.Method != http.MethodDelete {
		if e := decodeJSON(r, &req); e != nil {
			writeError(w, 400, "invalid_body", "active requis")
			return
		}
	}
	p := profileFromContext(r.Context())
	id := chi.URLParam(r, "id")
	if e := s.store.ActivateEmergencyGrant(r.Context(), p.ID, id, req.Active); e != nil {
		s.storeErr(w, e, "activate emergency grant")
		return
	}
	event := "emergency_grant_revoked"
	if req.Active {
		event = "emergency_grant_activated"
	}
	s.journal(r, &p.ID, event, id)
	writeJSON(w, 200, map[string]bool{"active": req.Active})
}
func containsID(ids []string, id string) bool {
	for _, v := range ids {
		if v == id {
			return true
		}
	}
	return false
}
func (s *Server) handleEmergencyRead(w http.ResponseWriter, r *http.Request) {
	viewer := profileFromContext(r.Context())
	g, e := s.store.ReadEmergencyGrant(r.Context(), viewer.ID, chi.URLParam(r, "id"))
	if e != nil {
		s.storeErr(w, e, "emergency read")
		return
	}
	allAssets, e := s.store.ListAssets(r.Context(), g.OwnerProfileID)
	if e != nil {
		s.storeErr(w, e, "emergency assets")
		return
	}
	assets := []store.Asset{}
	for _, a := range allAssets {
		if !containsID(g.AssetIDs, a.ID) {
			continue
		}
		a.Note = ""
		a.ProfileID = ""
		assets = append(assets, a)
	}
	// Filter first, never expose unrelated metadata to the recipient.
	all, e := s.store.ListDocuments(r.Context(), g.OwnerProfileID)
	if e != nil {
		s.storeErr(w, e, "emergency documents")
		return
	}
	docs := []store.Document{}
	for _, d := range all {
		if containsID(g.DocumentIDs, d.ID) {
			d.AssetName = ""
			d.AssetID = nil
			docs = append(docs, d)
		}
	}
	if _, e = s.store.ReadEmergencyGrant(r.Context(), viewer.ID, g.ID); e != nil {
		s.storeErr(w, e, "emergency revoked")
		return
	}
	s.journal(r, &g.OwnerProfileID, "emergency_read", g.ID)
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, 200, map[string]any{"grant": g, "assets": assets, "documents": docs})
}
func (s *Server) handleEmergencyDocument(w http.ResponseWriter, r *http.Request) {
	v := s.requireVault(w)
	if v == nil {
		return
	}
	g, e := s.store.ReadEmergencyGrant(r.Context(), profileFromContext(r.Context()).ID, chi.URLParam(r, "id"))
	if e != nil {
		s.storeErr(w, e, "emergency document grant")
		return
	}
	id := chi.URLParam(r, "documentID")
	if !containsID(g.DocumentIDs, id) {
		writeError(w, 404, "not_found", "Document introuvable")
		return
	}
	doc, encrypted, e := s.store.DocumentContent(r.Context(), g.OwnerProfileID, id)
	if e != nil {
		s.storeErr(w, e, "emergency document")
		return
	}
	plain, e := v.Decrypt(encrypted)
	if e != nil {
		writeError(w, 500, "vault_error", "Document indisponible")
		return
	}
	// Recheck immediately before disclosure, including expiration/revocation.
	if _, e = s.store.ReadEmergencyGrant(r.Context(), profileFromContext(r.Context()).ID, g.ID); e != nil {
		s.storeErr(w, e, "emergency revoked")
		return
	}
	s.journal(r, &g.OwnerProfileID, "emergency_document_downloaded", g.ID+":"+id)
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", doc.Mime)
	w.Header().Set("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": doc.Name}))
	w.WriteHeader(200)
	_, _ = w.Write(plain)
}
