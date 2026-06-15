package api

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/opale-app/opale/internal/money"
	"github.com/opale-app/opale/internal/store"
)

// storeErr mappe une erreur du store vers une réponse HTTP.
func (s *Server) storeErr(w http.ResponseWriter, err error, action string) {
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not_found", "ressource introuvable")
		return
	}
	s.log.Error(action, "err", err)
	writeError(w, http.StatusInternalServerError, "internal", "erreur interne")
}

type assetRequest struct {
	Name     string `json:"name"`
	Kind     string `json:"kind"`
	Currency string `json:"currency"`
	Note     string `json:"note"`
}

func (s *Server) handleCreateAsset(w http.ResponseWriter, r *http.Request) {
	var req assetRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "corps JSON invalide")
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "invalid_name", "le nom est requis")
		return
	}
	if !store.AssetKinds[req.Kind] {
		writeError(w, http.StatusBadRequest, "invalid_kind", "type d'actif inconnu")
		return
	}
	currency := normalizeCurrency(req.Currency)
	if currency == "" {
		writeError(w, http.StatusBadRequest, "invalid_currency", "devise invalide (code à 3 lettres)")
		return
	}

	p := profileFromContext(r.Context())
	asset, err := s.store.CreateAsset(r.Context(), p.ID, req.Name, req.Kind, currency, req.Note)
	if err != nil {
		s.storeErr(w, err, "create asset")
		return
	}
	writeJSON(w, http.StatusCreated, asset)
}

func (s *Server) handleListAssets(w http.ResponseWriter, r *http.Request) {
	p := profileFromContext(r.Context())
	assets, err := s.store.ListAssets(r.Context(), p.ID)
	if err != nil {
		s.storeErr(w, err, "list assets")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"assets": assets})
}

func (s *Server) handleGetAsset(w http.ResponseWriter, r *http.Request) {
	p := profileFromContext(r.Context())
	asset, err := s.store.GetAsset(r.Context(), p.ID, chi.URLParam(r, "id"))
	if err != nil {
		s.storeErr(w, err, "get asset")
		return
	}
	writeJSON(w, http.StatusOK, asset)
}

type updateAssetRequest struct {
	Name     string `json:"name"`
	Note     string `json:"note"`
	Archived bool   `json:"archived"`
}

func (s *Server) handleUpdateAsset(w http.ResponseWriter, r *http.Request) {
	var req updateAssetRequest
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
	asset, err := s.store.UpdateAsset(r.Context(), p.ID, chi.URLParam(r, "id"), req.Name, req.Note, req.Archived)
	if err != nil {
		s.storeErr(w, err, "update asset")
		return
	}
	writeJSON(w, http.StatusOK, asset)
}

func (s *Server) handleDeleteAsset(w http.ResponseWriter, r *http.Request) {
	p := profileFromContext(r.Context())
	if err := s.store.DeleteAsset(r.Context(), p.ID, chi.URLParam(r, "id")); err != nil {
		s.storeErr(w, err, "delete asset")
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}

// ─── Valorisations d'un actif (EF-032) ───────────────────────────────────────

type valuationRequest struct {
	ValueCents int64  `json:"value_cents"`
	AsOf       string `json:"as_of"` // format AAAA-MM-JJ ; défaut = aujourd'hui
	Note       string `json:"note"`
}

func (s *Server) handleAddAssetValuation(w http.ResponseWriter, r *http.Request) {
	req, asOf, ok := s.parseValuation(w, r)
	if !ok {
		return
	}
	p := profileFromContext(r.Context())
	v, err := s.store.AddAssetValuation(r.Context(), p.ID, chi.URLParam(r, "id"), money.Cents(req.ValueCents), asOf, req.Note)
	if err != nil {
		s.storeErr(w, err, "add asset valuation")
		return
	}
	writeJSON(w, http.StatusCreated, v)
}

func (s *Server) handleListAssetValuations(w http.ResponseWriter, r *http.Request) {
	p := profileFromContext(r.Context())
	vals, err := s.store.ListAssetValuations(r.Context(), p.ID, chi.URLParam(r, "id"))
	if err != nil {
		s.storeErr(w, err, "list asset valuations")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"valuations": vals})
}

// parseValuation valide un corps de valorisation partagé actifs/passifs.
func (s *Server) parseValuation(w http.ResponseWriter, r *http.Request) (valuationRequest, time.Time, bool) {
	var req valuationRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "corps JSON invalide")
		return req, time.Time{}, false
	}
	if req.ValueCents < 0 {
		writeError(w, http.StatusBadRequest, "invalid_value", "la valeur doit être positive (centimes)")
		return req, time.Time{}, false
	}
	asOf := time.Now().UTC().Truncate(24 * time.Hour)
	if req.AsOf != "" {
		t, err := time.Parse("2006-01-02", req.AsOf)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_date", "as_of doit être au format AAAA-MM-JJ")
			return req, time.Time{}, false
		}
		asOf = t
	}
	return req, asOf, true
}

// normalizeCurrency renvoie un code devise ISO à 3 lettres en majuscules, ou ""
// si invalide. Vide → EUR par défaut.
func normalizeCurrency(c string) string {
	c = strings.ToUpper(strings.TrimSpace(c))
	if c == "" {
		return "EUR"
	}
	if len(c) != 3 {
		return ""
	}
	for _, r := range c {
		if r < 'A' || r > 'Z' {
			return ""
		}
	}
	return c
}
