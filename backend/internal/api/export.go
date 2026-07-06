package api

// Export complet des données du profil (EF-006 / ENF-009) : la portabilité
// est une promesse du produit — « tes données t'appartiennent ».
//
// GET /v1/export renvoie un ZIP :
//   export.json    — tout le profil en JSON lisible (montants en centimes)
//   documents/…    — les documents du coffre, DÉCHIFFRÉS (c'est un export :
//                    l'utilisateur récupère ses originaux)

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/opale-app/opale/internal/store"
)

func (s *Server) handleExport(w http.ResponseWriter, r *http.Request) {
	p := profileFromContext(r.Context())
	ctx := r.Context()

	// ── Collecte (échec = pas d'export partiel silencieux) ────────────────
	assets, err := s.store.ListAssets(ctx, p.ID)
	if err != nil {
		s.storeErr(w, err, "export: assets")
		return
	}
	liabilities, err := s.store.ListLiabilities(ctx, p.ID)
	if err != nil {
		s.storeErr(w, err, "export: liabilities")
		return
	}

	type valuedAsset struct {
		store.Asset
		Valuations []store.Valuation `json:"valuations"`
	}
	exportAssets := make([]valuedAsset, 0, len(assets))
	for _, a := range assets {
		vals, err := s.store.ListAssetValuations(ctx, p.ID, a.ID)
		if err != nil {
			s.storeErr(w, err, "export: valuations")
			return
		}
		exportAssets = append(exportAssets, valuedAsset{Asset: a, Valuations: vals})
	}
	type valuedLiability struct {
		store.Liability
		Valuations []store.Valuation `json:"valuations"`
	}
	exportLiabilities := make([]valuedLiability, 0, len(liabilities))
	for _, l := range liabilities {
		vals, err := s.store.ListLiabilityValuations(ctx, p.ID, l.ID)
		if err != nil {
			s.storeErr(w, err, "export: valuations")
			return
		}
		exportLiabilities = append(exportLiabilities, valuedLiability{Liability: l, Valuations: vals})
	}

	transactions, err := s.store.ListTransactions(ctx, p.ID, store.TransactionFilter{})
	if err != nil {
		s.storeErr(w, err, "export: transactions")
		return
	}
	categories, err := s.store.ListCategories(ctx, p.ID)
	if err != nil {
		s.storeErr(w, err, "export: categories")
		return
	}
	goals, err := s.store.ListGoals(ctx, p.ID)
	if err != nil {
		s.storeErr(w, err, "export: goals")
		return
	}
	contacts, err := s.store.ListContacts(ctx, p.ID)
	if err != nil {
		s.storeErr(w, err, "export: contacts")
		return
	}
	documents, err := s.store.ListDocuments(ctx, p.ID)
	if err != nil {
		s.storeErr(w, err, "export: documents")
		return
	}
	properties, err := s.store.ListProperties(ctx, p.ID)
	if err != nil {
		s.storeErr(w, err, "export: properties")
		return
	}
	objects, err := s.store.ListObjects(ctx, p.ID)
	if err != nil {
		s.storeErr(w, err, "export: objects")
		return
	}
	companies, err := s.store.ListCompanies(ctx, p.ID)
	if err != nil {
		s.storeErr(w, err, "export: companies")
		return
	}

	payload := map[string]any{
		"format":       "opale-export/1",
		"exported_at":  time.Now().Format(time.RFC3339),
		"profile":      map[string]any{"id": p.ID, "name": p.Name},
		"assets":       exportAssets,
		"liabilities":  exportLiabilities,
		"transactions": transactions,
		"categories":   categories,
		"goals":        goals,
		"contacts":     contacts,
		"documents":    documents, // métadonnées ; contenus dans documents/
		"real_estate":  properties,
		"objects":      objects,
		"companies":    companies,
		"note":         "Montants en centimes d'euro (entiers). Documents déchiffrés dans documents/.",
	}

	s.journal(r, &p.ID, "export", fmt.Sprintf("%d actifs, %d mouvements, %d documents",
		len(exportAssets), len(transactions), len(documents)))

	// ── ZIP en flux ────────────────────────────────────────────────────────
	filename := "opale-export-" + time.Now().Format("2006-01-02") + ".zip"
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="`+filename+`"`)
	w.WriteHeader(http.StatusOK)

	zw := zip.NewWriter(w)
	defer zw.Close()

	f, err := zw.Create("export.json")
	if err != nil {
		return
	}
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(payload); err != nil {
		return
	}

	// Documents déchiffrés (seulement si le coffre est configuré).
	if s.vault != nil {
		for i, d := range documents {
			_, sealed, err := s.store.DocumentContent(ctx, p.ID, d.ID)
			if err != nil {
				continue
			}
			plain, err := s.vault.Decrypt(sealed)
			if err != nil {
				continue // document illisible : signalé par ailleurs, on n'interrompt pas l'export
			}
			df, err := zw.Create("documents/" + strconv.Itoa(i+1) + "-" + d.Name)
			if err != nil {
				return
			}
			if _, err := df.Write(plain); err != nil {
				return
			}
		}
	}
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
