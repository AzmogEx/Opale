package api

import (
	"github.com/go-chi/chi/v5"
	"github.com/opale-app/opale/internal/engine"
	"github.com/opale-app/opale/internal/money"
	"github.com/opale-app/opale/internal/store"
	"net/http"
	"sort"
	"strings"
	"time"
	_ "time/tzdata"
)

func parisToday() time.Time {
	loc, _ := time.LoadLocation("Europe/Paris")
	now := time.Now().In(loc)
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
}
func (s *Server) handleCalendar(w http.ResponseWriter, r *http.Request) {
	p := profileFromContext(r.Context())
	from, until := parisToday(), parisToday().AddDate(0, 3, 0)
	for k, d := range map[string]*time.Time{"from": &from, "until": &until} {
		if v := r.URL.Query().Get(k); v != "" {
			parsed, e := time.Parse(dayLayout, v)
			if e != nil {
				writeError(w, 400, "invalid_date", "Dates attendues YYYY-MM-DD")
				return
			}
			*d = parsed
		}
	}
	if until.Before(from) || until.After(from.AddDate(2, 0, 0)) {
		writeError(w, 400, "invalid_horizon", "Horizon compris entre 0 et 2 ans")
		return
	}
	rules, e := s.store.ListCalendarRules(r.Context(), p.ID)
	if e != nil {
		s.storeErr(w, e, "calendar")
		return
	}
	occ, e := s.store.CalendarOccurrences(r.Context(), p.ID, from, until)
	if e != nil {
		s.storeErr(w, e, "calendar occurrences")
		return
	}
	sort.Slice(occ, func(i, j int) bool { return occ[i].Date < occ[j].Date })
	writeJSON(w, 200, map[string]any{"rules": rules, "occurrences": occ, "from": from.Format(dayLayout), "until": until.Format(dayLayout)})
}
func (s *Server) handleSaveCalendar(w http.ResponseWriter, r *http.Request) {
	var c store.CalendarRule
	if e := decodeJSON(r, &c); e != nil {
		writeError(w, 400, "invalid_body", "Échéance invalide")
		return
	}
	c.ID = chi.URLParam(r, "id")
	c.Label = strings.TrimSpace(c.Label)
	d, e := time.Parse(dayLayout, c.Date)
	valid := map[string]bool{"once": true, "weekly": true, "monthly": true, "quarterly": true, "yearly": true}
	if e != nil || d.Year() < 1900 || d.Year() > time.Now().Year()+100 || c.AssetID == "" || len(c.Label) == 0 || len(c.Label) > 200 || c.Amount == 0 || !valid[c.Frequency] {
		writeError(w, 400, "invalid_calendar", "Compte, libellé, montant non nul, date et périodicité valides requis")
		return
	}
	if c.EndDate != nil && *c.EndDate == "" {
		c.EndDate = nil
	}
	if c.EndDate != nil {
		end, e := time.Parse(dayLayout, *c.EndDate)
		if e != nil || end.Before(d) {
			writeError(w, 400, "invalid_end", "Fin antérieure au début ou invalide")
			return
		}
	}
	created, e := s.store.SaveCalendarRule(r.Context(), profileFromContext(r.Context()).ID, c)
	if e != nil {
		s.storeErr(w, e, "save calendar")
		return
	}
	writeJSON(w, 200, created)
}
func (s *Server) handleDeleteCalendar(w http.ResponseWriter, r *http.Request) {
	if e := s.store.DeleteCalendarRule(r.Context(), profileFromContext(r.Context()).ID, chi.URLParam(r, "id")); e != nil {
		s.storeErr(w, e, "delete calendar")
		return
	}
	writeJSON(w, 204, nil)
}
func (s *Server) handleCalendarOccurrence(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Date          string  `json:"date"`
		Status        string  `json:"status"`
		Amount        *int64  `json:"amount_cents"`
		TransactionID *string `json:"transaction_id"`
	}
	e := decodeJSON(r, &req)
	_, dateErr := time.Parse(dayLayout, req.Date)
	if e != nil || dateErr != nil || (req.Status != "planned" && req.Status != "realized" && req.Status != "excluded") || (req.Status == "realized" && (req.TransactionID == nil || *req.TransactionID == "")) {
		writeError(w, 400, "invalid_occurrence", "Date, état et transaction réalisée valides requis")
		return
	}
	if req.Status != "realized" {
		req.TransactionID = nil
	}
	e = s.store.SetCalendarOccurrence(r.Context(), profileFromContext(r.Context()).ID, chi.URLParam(r, "id"), req.Date, req.Status, req.Amount, req.TransactionID)
	if e != nil {
		s.storeErr(w, e, "calendar occurrence")
		return
	}
	writeJSON(w, 200, map[string]bool{"saved": true})
}
func (s *Server) handleRecurringExclusion(w http.ResponseWriter, r *http.Request) {
	var req struct {
		MerchantKey string `json:"merchant_key"`
		Excluded    bool   `json:"excluded"`
	}
	if e := decodeJSON(r, &req); e != nil || strings.TrimSpace(req.MerchantKey) == "" {
		writeError(w, 400, "invalid_body", "merchant_key requis")
		return
	}
	if e := s.store.ExcludeRecurring(r.Context(), profileFromContext(r.Context()).ID, req.MerchantKey, req.Excluded); e != nil {
		s.storeErr(w, e, "exclude recurring")
		return
	}
	writeJSON(w, 200, map[string]bool{"saved": true})
}

// addCalendarFlows augments any cash projection with explicitly planned flows.
// Realized and excluded occurrences never count again as predicted cash.
func (s *Server) addCalendarFlows(r *http.Request, profile string, today time.Time, proj *engine.CashProjection) error {
	occ, e := s.store.CalendarOccurrences(r.Context(), profile, today, proj.Until)
	if e != nil {
		return e
	}
	for _, o := range occ {
		if o.Status != "planned" {
			continue
		}
		d, _ := time.Parse(dayLayout, o.Date)
		proj.Upcoming = append(proj.Upcoming, engine.UpcomingFlow{Date: d, Label: o.Label, Amount: o.EUR})
		v, e := money.Add(proj.EndCash, o.EUR)
		if e != nil {
			return e
		}
		proj.EndCash = v
	}
	sort.Slice(proj.Upcoming, func(i, j int) bool { return proj.Upcoming[i].Date.Before(proj.Upcoming[j].Date) })
	return nil
}
