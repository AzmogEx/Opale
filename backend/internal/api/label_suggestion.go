package api

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/opale-app/opale/internal/ai"
)

func (s *Server) handleLabelSuggestion(w http.ResponseWriter, r *http.Request) {
	p := profileFromContext(r.Context())
	tx, err := s.store.GetTransaction(r.Context(), p.ID, chi.URLParam(r, "id"))
	if err != nil {
		s.storeErr(w, err, "label suggestion transaction")
		return
	}
	// Send exactly the current label shown in the preview, never the hidden
	// imported raw label, which can contain additional banking references.
	suggestion, err := s.ai.SuggestLabel(r.Context(), tx.Label)
	out := map[string]any{"available": err == nil, "original_label": tx.Label, "source": "homelab", "state": "available"}
	if err != nil {
		out["state"] = "unavailable"
		out["reason"] = "Le homelab privé est indisponible. Tu peux corriger le libellé manuellement. Aucun envoi cloud."
		if errors.Is(err, ai.ErrInvalidLabel) {
			out["state"] = "invalid_response"
			out["reason"] = "La proposition n’a pas pu être validée. Le libellé est inchangé ; corrige-le manuellement."
		}
	} else {
		out["suggested_label"] = suggestion
	}
	writeJSON(w, http.StatusOK, out)
}
