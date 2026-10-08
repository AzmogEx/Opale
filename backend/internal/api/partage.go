package api

// Espace partagé (EF-007) et multi-devises (EF-008).
//
// Cloisonnement du partage : seules les transactions explicitement marquées
// « communes » deviennent visibles entre membres — le reste des données de
// chaque profil reste privé (EF-001).

import (
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

// ── Espaces partagés (EF-007) ─────────────────────────────────────────────────

func (s *Server) handleListSpaces(w http.ResponseWriter, r *http.Request) {
	p := profileFromContext(r.Context())
	spaces, err := s.store.ListSpaces(r.Context(), p.ID)
	if err != nil {
		s.storeErr(w, err, "list spaces")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"spaces": spaces})
}

func (s *Server) handleCreateSpace(w http.ResponseWriter, r *http.Request) {
	p := profileFromContext(r.Context())
	var req struct {
		Name string `json:"name"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "invalid_body", "name est requis")
		return
	}
	space, err := s.store.CreateSpace(r.Context(), p.ID, req.Name)
	if err != nil {
		s.storeErr(w, err, "create space")
		return
	}
	writeJSON(w, http.StatusCreated, space)
}

// requireSpaceMember vérifie l'appartenance et répond 404 sinon (on ne
// révèle pas l'existence d'un espace auquel on n'appartient pas).
func (s *Server) requireSpaceMember(w http.ResponseWriter, r *http.Request, spaceID string) bool {
	p := profileFromContext(r.Context())
	ok, err := s.store.IsSpaceMember(r.Context(), spaceID, p.ID)
	if err != nil {
		s.storeErr(w, err, "space membership")
		return false
	}
	if !ok {
		writeError(w, http.StatusNotFound, "not_found", "espace introuvable")
		return false
	}
	return true
}

// handleSpaceDetail — balance + dernières dépenses communes.
func (s *Server) handleSpaceDetail(w http.ResponseWriter, r *http.Request) {
	spaceID := chi.URLParam(r, "id")
	if !s.requireSpaceMember(w, r, spaceID) {
		return
	}
	members, total, err := s.store.SpaceBalance(r.Context(), spaceID)
	if err != nil {
		s.storeErr(w, err, "space balance")
		return
	}
	transactions, err := s.store.SpaceTransactions(r.Context(), spaceID, 50)
	if err != nil {
		s.storeErr(w, err, "space transactions")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"members":      members,
		"total_cents":  total,
		"transactions": transactions,
	})
}

func (s *Server) handleAddSpaceMember(w http.ResponseWriter, r *http.Request) {
	spaceID := chi.URLParam(r, "id")
	if !s.requireSpaceOwner(w, r, spaceID) {
		return
	}
	var req struct {
		ProfileID string `json:"profile_id"`
	}
	if err := decodeJSON(r, &req); err != nil || req.ProfileID == "" {
		writeError(w, http.StatusBadRequest, "invalid_body", "profile_id est requis")
		return
	}
	// Le profil doit exister (foyer = profils de l'instance).
	if _, err := s.store.GetProfile(r.Context(), req.ProfileID); err != nil {
		s.storeErr(w, err, "add member: profile")
		return
	}
	if err := s.store.AddSpaceMember(r.Context(), spaceID, req.ProfileID); err != nil {
		s.storeErr(w, err, "add member")
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}

func (s *Server) handleRemoveSpaceMember(w http.ResponseWriter, r *http.Request) {
	id, target := chi.URLParam(r, "id"), chi.URLParam(r, "profileID")
	if target == profileFromContext(r.Context()).ID {
		if !s.requireSpaceMember(w, r, id) {
			return
		}
	} else if !s.requireSpaceOwner(w, r, id) {
		return
	}
	owner, e := s.store.IsSpaceOwner(r.Context(), id, target)
	if e != nil {
		s.storeErr(w, e, "space owner")
		return
	}
	if owner {
		writeError(w, 409, "owner_required", "Le propriétaire doit conserver son accès à l’espace")
		return
	}
	if e = s.store.RemoveSpaceMember(r.Context(), id, target); e != nil {
		s.storeErr(w, e, "remove member")
		return
	}
	writeJSON(w, 204, nil)
}

// handleSetTransactionSpace marque/démarque une dépense commune.
func (s *Server) handleSetTransactionSpace(w http.ResponseWriter, r *http.Request) {
	p := profileFromContext(r.Context())
	var req struct {
		SpaceID string `json:"space_id"` // vide = retirer du commun
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}
	var spaceID *string
	if req.SpaceID != "" {
		spaceID = &req.SpaceID
	}
	if err := s.store.SetTransactionSpace(r.Context(), p.ID, chi.URLParam(r, "id"), spaceID); err != nil {
		s.storeErr(w, err, "set transaction space")
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}

// ── Multi-devises (EF-008) ────────────────────────────────────────────────────

var currencyRe = regexp.MustCompile(`^[A-Z]{3}$`)

func (s *Server) handleListFX(w http.ResponseWriter, r *http.Request) {
	rates, unrated, err := s.store.ListFXRates(r.Context(), profileFromContext(r.Context()).ID)
	if err != nil {
		s.storeErr(w, err, "list fx")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"rates":   rates,
		"unrated": unrated, // devises sans taux : calcul bloqué
	})
}

func (s *Server) handleUpsertFX(w http.ResponseWriter, r *http.Request) {
	currency := strings.ToUpper(chi.URLParam(r, "currency"))
	if !currencyRe.MatchString(currency) || currency == "EUR" {
		writeError(w, http.StatusBadRequest, "invalid_currency",
			"devise invalide (code ISO 3 lettres, EUR exclu)")
		return
	}
	var req struct {
		AsOf      string `json:"as_of"`
		RateMicro int64  `json:"rate_micro"` // 1 unité = N micro-euros
	}
	if err := decodeJSON(r, &req); err != nil || req.RateMicro <= 0 {
		writeError(w, http.StatusBadRequest, "invalid_body",
			"rate_micro (> 0, micro-euros par unité) est requis — ex. 1 USD = 0,92 € → 920000")
		return
	}
	if req.AsOf == "" {
		req.AsOf = parisToday().Format(dayLayout)
	}
	if d, e := time.Parse(dayLayout, req.AsOf); e != nil || d.After(parisToday()) {
		writeError(w, 400, "invalid_date", "Date de taux invalide ou future")
		return
	}
	rate, err := s.store.UpsertProfileFX(r.Context(), profileFromContext(r.Context()).ID, currency, req.AsOf, req.RateMicro)
	if err != nil {
		s.storeErr(w, err, "upsert fx")
		return
	}
	writeJSON(w, http.StatusOK, rate)
}

func (s *Server) handleDeleteFX(w http.ResponseWriter, r *http.Request) {
	currency := strings.ToUpper(chi.URLParam(r, "currency"))
	if err := s.store.DeleteProfileFX(r.Context(), profileFromContext(r.Context()).ID, currency); err != nil {
		s.storeErr(w, err, "delete fx")
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}

func (s *Server) requireSpaceOwner(w http.ResponseWriter, r *http.Request, id string) bool {
	ok, err := s.store.IsSpaceOwner(r.Context(), id, profileFromContext(r.Context()).ID)
	if err != nil {
		s.storeErr(w, err, "space owner")
		return false
	}
	if !ok {
		writeError(w, 404, "not_found", "espace indisponible")
		return false
	}
	return true
}
