package api

import (
	"github.com/go-chi/chi/v5"
	"github.com/opale-app/opale/internal/store"
	"net/http"
	"regexp"
	"time"
)

var requestUUID = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

func (s *Server) handleTransfer(w http.ResponseWriter, r *http.Request) {
	var n store.TransferInput
	if e := decodeJSON(r, &n); e != nil {
		writeError(w, 400, "invalid_body", e.Error())
		return
	}
	d, e := time.Parse(dayLayout, n.Date)
	if e != nil || d.After(parisToday()) || !requestUUID.MatchString(n.RequestID) || n.FromID == n.ToID || n.FromAmount <= 0 || n.ToAmount <= 0 || n.Fee < 0 {
		writeError(w, 400, "invalid_transfer", "Deux comptes distincts, montants positifs, frais positifs ou nuls, date passée et identifiant unique requis")
		return
	}
	rows, e := s.store.Transfer(r.Context(), profileFromContext(r.Context()).ID, n)
	if e != nil {
		s.storeErr(w, e, "transfer")
		return
	}
	writeJSON(w, 201, map[string]any{"transfer_id": n.RequestID, "transactions": rows})
}
func (s *Server) handleDeleteTransfer(w http.ResponseWriter, r *http.Request) {
	if e := s.store.DeleteTransfer(r.Context(), profileFromContext(r.Context()).ID, chi.URLParam(r, "id")); e != nil {
		s.storeErr(w, e, "delete transfer")
		return
	}
	writeJSON(w, 204, nil)
}
