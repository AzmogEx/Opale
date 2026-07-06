package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/opale-app/opale/internal/auth"
)

type createProfileRequest struct {
	Name           string `json:"name"`
	PIN            string `json:"pin"`
	PrivacyDefault string `json:"privacy_default"`
}

// handleCreateProfile crée un profil (EF-001).
func (s *Server) handleCreateProfile(w http.ResponseWriter, r *http.Request) {
	var req createProfileRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "corps JSON invalide")
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "invalid_name", "le nom est requis")
		return
	}
	if len(req.PIN) < 4 {
		writeError(w, http.StatusBadRequest, "invalid_pin", "le code doit comporter au moins 4 caractères")
		return
	}
	if req.PrivacyDefault == "" {
		req.PrivacyDefault = "N1"
	}
	if req.PrivacyDefault != "N1" && req.PrivacyDefault != "N2" && req.PrivacyDefault != "N3" {
		writeError(w, http.StatusBadRequest, "invalid_privacy", "privacy_default doit être N1, N2 ou N3")
		return
	}

	hash, err := auth.HashPIN(req.PIN)
	if err != nil {
		s.log.Error("hash pin", "err", err)
		writeError(w, http.StatusInternalServerError, "internal", "erreur interne")
		return
	}

	profile, err := s.store.CreateProfile(r.Context(), req.Name, hash, req.PrivacyDefault)
	if err != nil {
		s.log.Error("create profile", "err", err)
		writeError(w, http.StatusInternalServerError, "internal", "création du profil impossible")
		return
	}
	writeJSON(w, http.StatusCreated, profile)
}

// handleListProfiles liste les profils (écran de sélection, sans secret).
func (s *Server) handleListProfiles(w http.ResponseWriter, r *http.Request) {
	profiles, err := s.store.ListProfiles(r.Context())
	if err != nil {
		s.log.Error("list profiles", "err", err)
		writeError(w, http.StatusInternalServerError, "internal", "erreur interne")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"profiles": profiles})
}

type loginRequest struct {
	ProfileID string `json:"profile_id"`
	PIN       string `json:"pin"`
}

// handleLogin authentifie un profil par son code et ouvre une session (EF-002).
// Anti brute-force : 5 échecs verrouillent le profil 15 minutes (audit).
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "corps JSON invalide")
		return
	}

	if remaining, locked := s.logins.locked(req.ProfileID); locked {
		s.journal(r, &req.ProfileID, "login_locked", "")
		writeLocked(w, remaining)
		return
	}

	hash, err := s.store.ProfilePINHash(r.Context(), req.ProfileID)
	// Message volontairement générique pour ne pas révéler l'existence d'un profil.
	if err != nil || !auth.CheckPIN(hash, req.PIN) {
		s.logins.fail(req.ProfileID)
		if err == nil { // profil existant : l'échec le concerne, on le journalise
			s.journal(r, &req.ProfileID, "login_failed", "")
		}
		writeError(w, http.StatusUnauthorized, "invalid_credentials", "profil ou code incorrect")
		return
	}
	s.logins.reset(req.ProfileID)
	s.journal(r, &req.ProfileID, "login_ok", "")

	token, err := auth.NewToken()
	if err != nil {
		s.log.Error("new token", "err", err)
		writeError(w, http.StatusInternalServerError, "internal", "erreur interne")
		return
	}
	expiresAt := time.Now().Add(s.cfg.SessionTTL)
	if err := s.store.CreateSession(r.Context(), req.ProfileID, auth.HashToken(token), expiresAt); err != nil {
		s.log.Error("create session", "err", err)
		writeError(w, http.StatusInternalServerError, "internal", "erreur interne")
		return
	}

	profile, _ := s.store.GetProfile(r.Context(), req.ProfileID)
	writeJSON(w, http.StatusOK, map[string]any{
		"token":      token,
		"expires_at": expiresAt,
		"profile":    profile,
	})
}

// handleLogout révoque la session courante.
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get("X-Session-Token")
	if token == "" {
		if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
			token = strings.TrimPrefix(h, "Bearer ")
		}
	}
	if token != "" {
		_ = s.store.DeleteSession(r.Context(), auth.HashToken(token))
	}
	writeJSON(w, http.StatusNoContent, nil)
}

// handleMe renvoie le profil authentifié.
func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, profileFromContext(r.Context()))
}
