package api

// Export complet des données du profil (EF-006 / ENF-009) : la portabilité
// est une promesse du produit — « tes données t'appartiennent ».
//
// GET /v1/export renvoie un ZIP :
//   export.json    — tout le profil en JSON lisible (montants en centimes)
//   documents/…    — les documents du coffre, DÉCHIFFRÉS (c'est un export :
//                    l'utilisateur récupère ses originaux)

import (
	"net/http"
	"os"
	"strconv"
	"time"
)

func (s *Server) handleExport(w http.ResponseWriter, r *http.Request) {
	p := profileFromContext(r.Context())
	f, err := os.CreateTemp("", "opale-export-*.zip")
	if err != nil {
		s.storeErr(w, err, "export file")
		return
	}
	defer os.Remove(f.Name())
	defer f.Close()
	var decrypt func([]byte) ([]byte, error)
	if s.vault != nil {
		decrypt = s.vault.Decrypt
	}
	if err = s.store.ExportZIP(r.Context(), p.ID, f, decrypt); err != nil {
		s.storeErr(w, err, "export incomplete")
		return
	}
	stat, err := f.Stat()
	if err != nil {
		s.storeErr(w, err, "export stat")
		return
	}
	s.journal(r, &p.ID, "export", "complete format 2")
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="opale-export-`+time.Now().Format("2006-01-02")+`.zip"`)
	w.Header().Set("Content-Length", strconv.FormatInt(stat.Size(), 10))
	w.Header().Set("Cache-Control", "no-store")
	http.ServeContent(w, r, "export.zip", time.Time{}, f)
}

// handleAccessLog — le journal d'accès du profil (ENF-004).
func (s *Server) handleAccessLog(w http.ResponseWriter, r *http.Request) {
	p := profileFromContext(r.Context())
	events, err := s.store.AccessLog(r.Context(), p.ID, 100)
	if err != nil {
		s.storeErr(w, err, "access log")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"events": events})
}

// ── Gestion du profil (Réglages) ──────────────────────────────────────────────

// handleResetData efface tout le contenu du profil (Réglages → Données).
// Garde-fou : le client doit envoyer confirm=<nom du profil>.
func (s *Server) handleResetData(w http.ResponseWriter, r *http.Request) {
	p := profileFromContext(r.Context())
	if r.URL.Query().Get("confirm") != p.Name {
		writeError(w, http.StatusBadRequest, "confirmation_required",
			"Ajoute ?confirm=<nom du profil> pour confirmer la réinitialisation.")
		return
	}
	if err := s.store.ResetProfileData(r.Context(), p.ID); err != nil {
		s.storeErr(w, err, "reset data")
		return
	}
	s.journal(r, &p.ID, "data_reset", "")
	writeJSON(w, http.StatusNoContent, nil)
}

// handleDeleteMe supprime le profil et toutes ses données. Irréversible.
func (s *Server) handleDeleteMe(w http.ResponseWriter, r *http.Request) {
	p := profileFromContext(r.Context())
	if r.URL.Query().Get("confirm") != p.Name {
		writeError(w, http.StatusBadRequest, "confirmation_required",
			"Ajoute ?confirm=<nom du profil> pour confirmer la suppression.")
		return
	}
	// Journalisé AVANT (le cascade emporte le journal du profil).
	s.journal(r, &p.ID, "profile_deleted", "")
	if err := s.store.DeleteProfile(r.Context(), p.ID); err != nil {
		s.storeErr(w, err, "delete profile")
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}
