package api

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/opale-app/opale/internal/categorize"
	"github.com/opale-app/opale/internal/csvimport"
	"github.com/opale-app/opale/internal/money"
	"github.com/opale-app/opale/internal/store"
)

const dayLayout = "2006-01-02"

// ── Catégories ────────────────────────────────────────────────────────────────

func (s *Server) handleListCategories(w http.ResponseWriter, r *http.Request) {
	p := profileFromContext(r.Context())
	cats, err := s.store.ListCategories(r.Context(), p.ID)
	if err != nil {
		s.storeErr(w, err, "list categories")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"categories": cats})
}

// categoriesByName construit l'index nom → id des catégories du profil.
func (s *Server) categoriesByName(r *http.Request, profileID string) (map[string]string, error) {
	cats, err := s.store.ListCategories(r.Context(), profileID)
	if err != nil {
		return nil, err
	}
	byName := make(map[string]string, len(cats))
	for _, c := range cats {
		byName[c.Name] = c.ID
	}
	return byName, nil
}

// suggestCategoryID applique le catégoriseur (règles du profil puis mots-clés)
// et traduit le nom proposé en id. nil si aucune suggestion.
func (s *Server) suggestCategoryID(
	r *http.Request, profileID, rawLabel string, amount money.Cents,
	rules map[string]string, byName map[string]string,
) *string {
	name := categorize.SuggestCategory(rules, rawLabel, int64(amount))
	if name == "" {
		return nil
	}
	if id, ok := byName[name]; ok {
		return &id
	}
	return nil
}

// ── Transactions (EF-020) ─────────────────────────────────────────────────────

func (s *Server) handleListTransactions(w http.ResponseWriter, r *http.Request) {
	p := profileFromContext(r.Context())
	q := r.URL.Query()

	f := store.TransactionFilter{
		Query:      strings.TrimSpace(q.Get("q")),
		CategoryID: q.Get("category_id"),
		AssetID:    q.Get("asset_id"),
	}
	if raw := q.Get("from"); raw != "" {
		t, err := time.Parse(dayLayout, raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_from", "from doit être yyyy-MM-dd")
			return
		}
		f.From = &t
	}
	if raw := q.Get("to"); raw != "" {
		t, err := time.Parse(dayLayout, raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_to", "to doit être yyyy-MM-dd")
			return
		}
		f.To = &t
	}
	if raw := q.Get("limit"); raw != "" {
		f.Limit, _ = strconv.Atoi(raw)
	}
	if raw := q.Get("offset"); raw != "" {
		f.Offset, _ = strconv.Atoi(raw)
	}

	txs, err := s.store.ListTransactions(r.Context(), p.ID, f)
	if err != nil {
		s.storeErr(w, err, "list transactions")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"transactions": txs})
}

type transactionRequest struct {
	AssetID    string `json:"asset_id"`
	Amount     int64  `json:"amount_cents"`
	OccurredOn string `json:"occurred_on"` // yyyy-MM-dd
	Label      string `json:"label"`
	CategoryID string `json:"category_id"`
	Note       string `json:"note"`
}

func (s *Server) handleCreateTransaction(w http.ResponseWriter, r *http.Request) {
	p := profileFromContext(r.Context())

	var req transactionRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}
	req.Label = strings.TrimSpace(req.Label)
	if req.AssetID == "" || req.Label == "" || req.Amount == 0 {
		writeError(w, http.StatusBadRequest, "invalid_body",
			"asset_id, label et amount_cents (≠ 0) sont requis")
		return
	}
	occurredOn, err := time.Parse(dayLayout, req.OccurredOn)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_date", "occurred_on doit être yyyy-MM-dd")
		return
	}

	n := store.NewTransaction{
		AssetID:     req.AssetID,
		Amount:      money.Cents(req.Amount),
		OccurredOn:  occurredOn,
		Label:       req.Label,
		RawLabel:    req.Label,
		MerchantKey: categorize.MerchantKey(req.Label),
		Note:        req.Note,
	}
	if req.CategoryID != "" {
		n.CategoryID = &req.CategoryID
	} else {
		// Catégorisation automatique (EF-022).
		rules, err := s.store.MerchantRuleNames(r.Context(), p.ID)
		if err == nil {
			if byName, err2 := s.categoriesByName(r, p.ID); err2 == nil {
				n.CategoryID = s.suggestCategoryID(r, p.ID, req.Label, n.Amount, rules, byName)
			}
		}
	}

	tx, err := s.store.CreateTransaction(r.Context(), p.ID, n)
	if err != nil {
		s.storeErr(w, err, "create transaction")
		return
	}
	writeJSON(w, http.StatusCreated, tx)
}

type transactionPatchRequest struct {
	Label          *string `json:"label"`
	Note           *string `json:"note"`
	CategoryID     *string `json:"category_id"` // "" = décatégoriser
	OccurredOn     *string `json:"occurred_on"`
	Amount         *int64  `json:"amount_cents"`
	ApplyToSimilar bool    `json:"apply_to_similar"` // même marchand → même catégorie
}

func (s *Server) handleUpdateTransaction(w http.ResponseWriter, r *http.Request) {
	p := profileFromContext(r.Context())
	id := chi.URLParam(r, "id")

	var req transactionPatchRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}

	patch := store.TransactionPatch{Label: req.Label, Note: req.Note, CategoryID: req.CategoryID}
	if req.OccurredOn != nil {
		t, err := time.Parse(dayLayout, *req.OccurredOn)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid_date", "occurred_on doit être yyyy-MM-dd")
			return
		}
		patch.OccurredOn = &t
	}
	if req.Amount != nil {
		c := money.Cents(*req.Amount)
		patch.Amount = &c
	}

	tx, err := s.store.UpdateTransaction(r.Context(), p.ID, id, patch)
	if err != nil {
		s.storeErr(w, err, "update transaction")
		return
	}

	// Correction de catégorie → apprentissage (EF-022) : la règle marchand du
	// profil est mise à jour, et primera sur les mots-clés pour la suite.
	if req.CategoryID != nil && *req.CategoryID != "" && tx.MerchantKey != "" {
		if err := s.store.UpsertMerchantRule(r.Context(), p.ID, tx.MerchantKey, *req.CategoryID); err != nil {
			s.log.Error("upsert merchant rule", "err", err)
		}
		if req.ApplyToSimilar {
			if _, err := s.store.ApplyCategoryToMerchant(r.Context(), p.ID, tx.MerchantKey, *req.CategoryID); err != nil {
				s.log.Error("apply category to merchant", "err", err)
			}
		}
	}

	writeJSON(w, http.StatusOK, tx)
}

func (s *Server) handleDeleteTransaction(w http.ResponseWriter, r *http.Request) {
	p := profileFromContext(r.Context())
	if err := s.store.DeleteTransaction(r.Context(), p.ID, chi.URLParam(r, "id")); err != nil {
		s.storeErr(w, err, "delete transaction")
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}

// ── Résumé mensuel (EF-020) ───────────────────────────────────────────────────

func (s *Server) handleMonthSummary(w http.ResponseWriter, r *http.Request) {
	p := profileFromContext(r.Context())
	now := time.Now()

	year, month := now.Year(), int(now.Month())
	if raw := r.URL.Query().Get("year"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil {
			year = n
		}
	}
	if raw := r.URL.Query().Get("month"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n >= 1 && n <= 12 {
			month = n
		}
	}

	sum, err := s.store.ComputeMonthSummary(r.Context(), p.ID, year, time.Month(month))
	if err != nil {
		s.storeErr(w, err, "month summary")
		return
	}
	writeJSON(w, http.StatusOK, sum)
}

// ── Import CSV (EF-021) ───────────────────────────────────────────────────────

type importRequest struct {
	AssetID string `json:"asset_id"`
	CSV     string `json:"csv"`
}

func (s *Server) handleImportCSV(w http.ResponseWriter, r *http.Request) {
	p := profileFromContext(r.Context())

	var req importRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}
	if req.AssetID == "" || strings.TrimSpace(req.CSV) == "" {
		writeError(w, http.StatusBadRequest, "invalid_body", "asset_id et csv sont requis")
		return
	}

	// CSV ou OFX : détection automatique (EF-070).
	var rows []csvimport.Row
	var parseErr error
	if csvimport.IsOFX(req.CSV) {
		rows, parseErr = csvimport.ParseOFX(req.CSV)
	} else {
		rows, parseErr = csvimport.Parse(req.CSV)
	}
	if parseErr != nil {
		writeError(w, http.StatusBadRequest, "invalid_csv", parseErr.Error())
		return
	}

	// Préparation : nettoyage des libellés (EF-023) + catégorisation (EF-022).
	rules, err := s.store.MerchantRuleNames(r.Context(), p.ID)
	if err != nil {
		s.storeErr(w, err, "merchant rules")
		return
	}
	byName, err := s.categoriesByName(r, p.ID)
	if err != nil {
		s.storeErr(w, err, "categories")
		return
	}

	prepared := make([]store.NewTransaction, 0, len(rows))
	for _, row := range rows {
		prepared = append(prepared, store.NewTransaction{
			AssetID:     req.AssetID,
			Amount:      row.Amount,
			OccurredOn:  row.OccurredOn,
			Label:       categorize.CleanLabel(row.RawLabel),
			RawLabel:    row.RawLabel,
			MerchantKey: categorize.MerchantKey(row.RawLabel),
			CategoryID:  s.suggestCategoryID(r, p.ID, row.RawLabel, row.Amount, rules, byName),
		})
	}

	res, err := s.store.ImportTransactions(r.Context(), p.ID, prepared)
	if err != nil {
		s.storeErr(w, err, "import transactions")
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// ── Split multi-catégories (EF-024) ───────────────────────────────────────────

type splitRequest struct {
	Parts []struct {
		Amount     int64  `json:"amount_cents"`
		CategoryID string `json:"category_id"`
		Label      string `json:"label"`
	} `json:"parts"`
}

// handleSplitTransaction scinde un mouvement en plusieurs parts catégorisées.
// La somme des parts doit égaler le montant d'origine, au centime.
func (s *Server) handleSplitTransaction(w http.ResponseWriter, r *http.Request) {
	p := profileFromContext(r.Context())
	var req splitRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", err.Error())
		return
	}
	if len(req.Parts) < 2 {
		writeError(w, http.StatusBadRequest, "invalid_body", "au moins 2 parts sont requises")
		return
	}

	parts := make([]store.SplitPart, 0, len(req.Parts))
	for _, part := range req.Parts {
		sp := store.SplitPart{Amount: money.Cents(part.Amount), Label: part.Label}
		if part.CategoryID != "" {
			id := part.CategoryID
			sp.CategoryID = &id
		}
		parts = append(parts, sp)
	}

	created, err := s.store.SplitTransaction(r.Context(), p.ID, chi.URLParam(r, "id"), parts)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "mouvement introuvable")
			return
		}
		writeError(w, http.StatusBadRequest, "invalid_split", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"transactions": created})
}
