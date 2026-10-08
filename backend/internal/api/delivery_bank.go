package api

import (
	"context"
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/opale-app/opale/internal/bank"
	"github.com/opale-app/opale/internal/categorize"
	"github.com/opale-app/opale/internal/money"
	"github.com/opale-app/opale/internal/store"
	"net/http"
	"time"
)

type bankSyncResult struct {
	LinkID      string `json:"link_id"`
	Institution string `json:"institution"`
	Status      string `json:"status"`
	Imported    int    `json:"imported"`
	Duplicates  int    `json:"duplicates"`
	Error       string `json:"error,omitempty"`
}

// RunBankSync is safe to call on several server instances: persisted per-link
// leases prevent concurrent imports, and the due date survives server restarts.
func (s *Server) RunBankSync(ctx context.Context) {
	if s.bank == nil {
		return
	}
	profiles, e := s.store.ListProfiles(ctx)
	if e != nil {
		return
	}
	for _, p := range profiles {
		if ctx.Err() != nil {
			return
		}
		links, e := s.store.ListBankLinks(ctx, p.ID)
		if e != nil {
			continue
		}
		for _, link := range links {
			if ctx.Err() != nil {
				return
			}
			s.syncBankLink(ctx, p.ID, link)
		}
	}
}
func bankFailure(err error) (string, string, time.Duration) {
	var p *bank.APIError
	if errors.As(err, &p) {
		switch p.Status {
		case 401, 403:
			return "renew_consent", "Accès refusé ou expiré : vérifie les secrets fournisseur puis renouvelle le consentement.", 24 * time.Hour
		case 429:
			return "error", "Limite bancaire atteinte ; nouvelle tentative planifiée.", max(p.RetryAfter, 6*time.Hour)
		case 409:
			return "error", "Compte en cours de traitement ou suspendu chez le fournisseur.", 6 * time.Hour
		}
	}
	return "error", "Synchronisation indisponible ; vérifie le compte associé et réessaie après le délai indiqué.", 6 * time.Hour
}
func (s *Server) syncBankLink(ctx context.Context, profile string, link store.BankLink) (res bankSyncResult) {
	res = bankSyncResult{LinkID: link.ID, Institution: link.InstitutionName}
	claimed, e := s.store.ClaimBankLink(ctx, profile, link.ID)
	if e != nil {
		res.Status = "error"
		res.Error = "Planification indisponible"
		return
	}
	if !claimed {
		res.Status = "cooldown"
		return
	}
	retry := 12 * time.Hour
	defer func() {
		cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		if e := s.store.FinishBankLink(cleanup, profile, link.ID, res.Status, res.Error, retry); e != nil {
			s.log.Warn("bank sync state unavailable")
		}
	}()
	ctx, cancel := context.WithTimeout(ctx, 4*time.Minute)
	defer cancel()
	requisition, e := s.bank.GetRequisition(ctx, link.RequisitionID)
	if e != nil {
		res.Status, res.Error, retry = bankFailure(e)
		return
	}
	if requisition.Status == "EX" || requisition.Status == "RJ" {
		res.Status = "renew_consent"
		res.Error = "Consentement expiré ou refusé : reconnecte la banque."
		retry = 24 * time.Hour
		return
	}
	if len(requisition.Accounts) == 0 || requisition.Status != "LN" {
		res.Status = "pending_consent"
		retry = 15 * time.Minute
		return
	}
	if e = s.store.DiscoverBankAccounts(ctx, profile, link.ID, requisition.Accounts); e != nil {
		res.Status, res.Error, retry = bankFailure(e)
		return
	}
	accounts, e := s.store.ListBankAccounts(ctx, profile)
	if e != nil {
		res.Status, res.Error, retry = bankFailure(e)
		return
	}
	cats, e := s.store.ListCategories(ctx, profile)
	if e != nil {
		res.Status, res.Error, retry = bankFailure(e)
		return
	}
	byName := map[string]string{}
	for _, c := range cats {
		byName[c.Name] = c.ID
	}
	rules, e := s.store.MerchantRuleNames(ctx, profile)
	if e != nil {
		res.Status, res.Error, retry = bankFailure(e)
		return
	}
	res.Status = "synced"
	for _, a := range accounts {
		if a.LinkID != link.ID || a.Status == "closed" {
			continue
		}
		if a.Currency == "" {
			details, e := s.bank.AccountDetails(ctx, a.ProviderAccountID)
			if e != nil {
				res.Status, res.Error, retry = bankFailure(e)
				_ = s.store.MarkBankAccountError(ctx, profile, a.ID, res.Error)
				continue
			}
			if details.Currency == "" || details.Status == "blocked" || details.Status == "deleted" {
				res.Status, res.Error = "error", "Compte indisponible ou devise absente chez le fournisseur."
				_ = s.store.MarkBankAccountError(ctx, profile, a.ID, res.Error)
				continue
			}
			if e = s.store.SaveBankAccountDetails(ctx, profile, a.ID, details.Currency, details.Name); e != nil {
				res.Status, res.Error, retry = bankFailure(e)
				continue
			}
			a.Currency, a.Name = details.Currency, details.Name
		}
		if a.AssetID == nil {
			if res.Status == "synced" {
				res.Status = "needs_mapping"
			}
			continue
		}
		asset, e := s.store.GetAsset(ctx, profile, *a.AssetID)
		if e != nil {
			res.Status, res.Error, retry = bankFailure(e)
			continue
		}
		if a.Currency != asset.Currency {
			res.Status, res.Error = "error", "La devise du compte bancaire diffère de celle de l’actif associé."
			_ = s.store.MarkBankAccountError(ctx, profile, a.ID, res.Error)
			continue
		}
		a.Currency = asset.Currency
		exponent, e := s.store.CurrencyExponent(ctx, asset.Currency)
		if e != nil {
			res.Status, res.Error, retry = bankFailure(e)
			continue
		}
		movements, e := s.bank.Transactions(ctx, a.ProviderAccountID)
		if e != nil {
			res.Status, res.Error, retry = bankFailure(e)
			_ = s.store.MarkBankAccountError(ctx, profile, a.ID, res.Error)
			continue
		}
		booked, pending := []store.NewTransaction{}, []store.NewTransaction{}
		invalid := false
		for _, m := range movements {
			if m.Currency != asset.Currency {
				invalid = true
				break
			}
			amount, e := bank.ParseMinor(m.RawAmount, exponent)
			if e != nil {
				invalid = true
				break
			}
			row := store.NewTransaction{AssetID: *a.AssetID, Amount: amount, OccurredOn: m.OccurredOn, Label: categorize.CleanLabel(m.RawLabel), RawLabel: m.RawLabel, MerchantKey: categorize.MerchantKey(m.RawLabel), SourceID: m.SourceID, BankStatus: m.Status}
			if id, ok := byName[categorize.SuggestCategory(rules, m.RawLabel, int64(amount))]; ok {
				row.CategoryID = &id
			}
			if m.Status == "pending" {
				pending = append(pending, row)
			} else {
				booked = append(booked, row)
			}
		}
		if invalid {
			res.Status = "error"
			res.Error = "Devise ou précision incompatible avec le compte associé ; aucun mouvement de ce compte importé."
			_ = s.store.MarkBankAccountError(ctx, profile, a.ID, res.Error)
			continue
		}
		balances, e := s.bank.Balances(ctx, a.ProviderAccountID)
		if e != nil {
			res.Status, res.Error, retry = bankFailure(e)
			_ = s.store.MarkBankAccountError(ctx, profile, a.ID, res.Error)
			continue
		}
		var selected *bank.Balance
		var balanceValue money.Cents
		for _, b := range balances {
			if b.Currency != asset.Currency || b.CreditIncluded || b.Date == "" {
				continue
			}
			day, e := time.Parse(dayLayout, b.Date)
			if e != nil || day.After(parisToday()) {
				continue
			}
			if b.Kind != "closingBooked" && b.Kind != "interimBooked" {
				continue
			}
			if selected == nil || b.Date > selected.Date || (b.Date == selected.Date && b.Kind == "closingBooked") {
				copy := b
				selected = &copy
			}
		}
		if selected != nil {
			balanceValue, e = bank.ParseMinor(selected.RawAmount, exponent)
			if e != nil {
				res.Status, res.Error, retry = bankFailure(e)
				continue
			}
		}
		imported, e := s.store.ApplyBankAccount(ctx, profile, a, booked, pending, selected, balanceValue)
		if e != nil {
			res.Status, res.Error, retry = bankFailure(e)
			_ = s.store.MarkBankAccountError(ctx, profile, a.ID, res.Error)
			continue
		}
		res.Imported += imported.Imported
		res.Duplicates += imported.Duplicates
	}
	if res.Status == "error" {
		retry = max(retry, time.Duration(1<<min(link.Attempts, 3))*time.Hour)
	}
	if res.Status == "needs_mapping" {
		retry = 12 * time.Hour
	}
	return
}
func (s *Server) handleBankAccounts(w http.ResponseWriter, r *http.Request) {
	accounts, e := s.store.ListBankAccounts(r.Context(), profileFromContext(r.Context()).ID)
	if e != nil {
		s.storeErr(w, e, "bank accounts")
		return
	}
	pending, e := s.store.ListBankPending(r.Context(), profileFromContext(r.Context()).ID)
	if e != nil {
		s.storeErr(w, e, "bank pending")
		return
	}
	writeJSON(w, 200, map[string]any{"accounts": accounts, "pending": pending})
}
func (s *Server) handleMapBankAccount(w http.ResponseWriter, r *http.Request) {
	var req struct {
		AssetID string `json:"asset_id"`
	}
	if e := decodeJSON(r, &req); e != nil {
		writeError(w, 400, "invalid_body", "asset_id requis (vide pour dissocier)")
		return
	}
	var asset *string
	if req.AssetID != "" {
		asset = &req.AssetID
	}
	p := profileFromContext(r.Context())
	if e := s.store.MapBankAccount(r.Context(), p.ID, chi.URLParam(r, "id"), asset); e != nil {
		if errors.Is(e, store.ErrInvalid) {
			writeError(w, 422, "bank_mapping_locked", "Ce compte bancaire a déjà un historique importé. Réassocie l’actif d’origine pour éviter de dupliquer les mouvements ; la dissociation reste possible.")
			return
		}
		s.storeErr(w, e, "bank map")
		return
	}
	s.journal(r, &p.ID, "bank_account_mapped", chi.URLParam(r, "id"))
	writeJSON(w, 200, map[string]bool{"saved": true})
}
func (s *Server) handleRenewBank(w http.ResponseWriter, r *http.Request) {
	if !s.requireBank(w) {
		return
	}
	p := profileFromContext(r.Context())
	id := chi.URLParam(r, "id")
	link, e := s.store.BankLink(r.Context(), p.ID, id)
	if e != nil {
		s.storeErr(w, e, "renew bank")
		return
	}
	var req struct {
		Redirect string `json:"redirect"`
	}
	if e := decodeJSON(r, &req); e != nil {
		writeError(w, 400, "invalid_body", "redirect invalide")
		return
	}
	if req.Redirect == "" {
		req.Redirect = "opale://bank-linked"
	}
	next, e := s.bank.CreateRequisition(r.Context(), link.InstitutionID, req.Redirect)
	if e != nil {
		writeError(w, 502, "bank_error", "Création du consentement indisponible")
		return
	}
	if e = s.bank.DeleteRequisition(r.Context(), link.RequisitionID); e != nil {
		var pErr *bank.APIError
		if !errors.As(e, &pErr) || pErr.Status != 404 {
			_ = s.bank.DeleteRequisition(r.Context(), next.ID)
			writeError(w, 502, "revoke_failed", "Ancien consentement non révoqué ; renouvellement annulé")
			return
		}
	}
	if e = s.store.RenewBankLink(r.Context(), p.ID, id, next.ID); e != nil {
		s.storeErr(w, e, "renew bank link")
		return
	}
	s.journal(r, &p.ID, "bank_consent_renewed", id)
	writeJSON(w, 200, map[string]any{"consent_link": next.Link})
}
