package api

import (
	"github.com/opale-app/opale/internal/money"
	"github.com/opale-app/opale/internal/tax"
	"net/http"
	"strconv"
)

// Only the verified assessment year is accepted. A PER deduction is capped at
// the user's available notice ceiling, never inferred from taxable income.
func (s *Server) handleTaxEstimateVerified(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	income, e := strconv.ParseInt(q.Get("income_cents"), 10, 64)
	if e != nil || income < 0 || income > 1_000_000_000_000 {
		writeError(w, 400, "invalid_income", "Revenu imposable annuel en centimes requis (0–10 milliards €)")
		return
	}
	year := 2026
	if v := q.Get("year"); v != "" {
		year, e = strconv.Atoi(v)
		if e != nil || year != 2026 {
			writeError(w, 422, "unsupported_tax_year", "Seul le barème 2026 sur revenus 2025 est vérifié")
			return
		}
	}
	parts := 10
	if v := q.Get("parts_tenths"); v != "" {
		parts, e = strconv.Atoi(v)
		if e != nil || parts < 10 || parts > 100 {
			writeError(w, 400, "invalid_parts", "parts_tenths entre 10 et 100")
			return
		}
	}
	resp := map[string]any{"estimate": tax.Compute(money.Cents(income), parts), "assessment_year": year, "income_year": 2025, "verified_on": "2026-10-08", "source_url": "https://www.service-public.gouv.fr/particuliers/vosdroits/F1419", "disclaimer": "Impôt brut au barème, hors décote, plafonnement du quotient familial, réductions/crédits et contributions exceptionnelles. Ne constitue pas le montant final à payer."}
	if v := q.Get("per_cents"); v != "" {
		per, e := strconv.ParseInt(v, 10, 64)
		if e != nil || per < 0 || per > 1_000_000_000_000 {
			writeError(w, 400, "invalid_per", "Versement PER invalide")
			return
		}
		if per > 0 {
			ceiling, e := strconv.ParseInt(q.Get("per_ceiling_cents"), 10, 64)
			if e != nil || ceiling < 0 || ceiling > 1_000_000_000_000 {
				writeError(w, 422, "per_ceiling_required", "Renseigne le plafond PER disponible indiqué sur ton avis d’impôt pour les versements 2025")
				return
			}
			deductible := min(per, ceiling, income)
			effect := tax.ComputePEREffect(money.Cents(income), money.Cents(deductible), parts)
			effect.Contribution = money.Cents(per)
			effect.RealCost = money.Cents(per) - effect.Savings
			resp["per_effect"] = effect
			resp["deductible_cents"] = deductible
			resp["per_ceiling_cents"] = ceiling
			resp["per_source_url"] = "https://www.impots.gouv.fr/particulier/questions/une-somme-est-pre-remplie-dans-la-case-plafond-de-deduction-de-la-rubrique"
		}
	}
	writeJSON(w, 200, resp)
}
