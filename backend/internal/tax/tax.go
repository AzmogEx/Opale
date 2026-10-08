// Package tax — module fiscal français (indicatif) : barème progressif de
// l'impôt sur le revenu, TMI, et effet d'un versement PER.
//
// AVERTISSEMENT assumé côté produit : estimation indicative sur le barème
// courant, hors décote, réductions, crédits d'impôt et cas particuliers.
// Arithmétique entière (centimes / bps) — ENF-007.
package tax

import "github.com/opale-app/opale/internal/money"

// Bracket — une tranche du barème (bornes annuelles PAR PART, en centimes).
type Bracket struct {
	UpTo    money.Cents `json:"up_to_cents"` // 0 = sans plafond (dernière tranche)
	RateBps int         `json:"rate_bps"`
}

// Barème IR 2026 (revenus 2025), vérifié le 08/10/2026.
// Source : https://www.service-public.gouv.fr/particuliers/vosdroits/F1419
// L’impôt brut ne comprend pas décote, plafonnement familial ou crédits.
var Brackets = []Bracket{
	{UpTo: money.Cents(11_600_00), RateBps: 0},
	{UpTo: money.Cents(29_579_00), RateBps: 1_100},
	{UpTo: money.Cents(84_577_00), RateBps: 3_000},
	{UpTo: money.Cents(181_917_00), RateBps: 4_100},
	{UpTo: 0, RateBps: 4_500},
}

// BracketDetail — la part d'impôt payée dans une tranche.
type BracketDetail struct {
	Bracket
	Taxed money.Cents `json:"taxed_cents"` // assiette imposée dans la tranche
	Tax   money.Cents `json:"tax_cents"`
}

// Estimate — le verdict fiscal.
type Estimate struct {
	// Impôt total du foyer (toutes parts).
	Tax money.Cents `json:"tax_cents"`
	// TMI : taux marginal d'imposition (bps).
	MarginalRateBps int `json:"marginal_rate_bps"`
	// Taux moyen effectif (bps, sur le revenu imposable).
	AverageRateBps int `json:"average_rate_bps"`
	// Détail par tranche (pour une part, multiplié par les parts au total).
	Details []BracketDetail `json:"details"`
	// Revenu net après impôt.
	NetIncome money.Cents `json:"net_income_cents"`
}

// Compute estime l'IR : revenu imposable annuel du foyer et nombre de parts
// (en dixièmes : 10 = 1 part, 25 = 2,5 parts — entier, jamais de float).
func Compute(taxableIncome money.Cents, partsTenths int) Estimate {
	if partsTenths <= 0 {
		partsTenths = 10
	}
	if taxableIncome < 0 {
		taxableIncome = 0
	}

	// Quotient familial : revenu par part.
	perPart := int64(taxableIncome) * 10 / int64(partsTenths)

	est := Estimate{}
	var taxPerPart int64
	var lower int64
	for _, b := range Brackets {
		upper := int64(b.UpTo)
		var taxedInBracket int64
		if upper == 0 || perPart <= upper { // dernière tranche atteinte
			taxedInBracket = perPart - lower
		} else {
			taxedInBracket = upper - lower
		}
		if taxedInBracket < 0 {
			taxedInBracket = 0
		}
		tax := taxedInBracket * int64(b.RateBps) / 10_000
		taxPerPart += tax
		if taxedInBracket > 0 {
			est.MarginalRateBps = b.RateBps
			est.Details = append(est.Details, BracketDetail{
				Bracket: b,
				Taxed:   money.Cents(taxedInBracket),
				Tax:     money.Cents(tax),
			})
		}
		if upper == 0 || perPart <= upper {
			break
		}
		lower = upper
	}

	// Impôt du foyer = impôt par part × parts.
	est.Tax = money.Cents(taxPerPart * int64(partsTenths) / 10)
	if taxableIncome > 0 {
		est.AverageRateBps = int(int64(est.Tax) * 10_000 / int64(taxableIncome))
	}
	est.NetIncome = taxableIncome - est.Tax
	return est
}

// PEREffect — l'économie d'impôt d'un versement PER (déductible du revenu).
type PEREffect struct {
	Contribution money.Cents `json:"contribution_cents"`
	TaxBefore    money.Cents `json:"tax_before_cents"`
	TaxAfter     money.Cents `json:"tax_after_cents"`
	Savings      money.Cents `json:"savings_cents"`
	// Coût réel du versement une fois l'économie déduite.
	RealCost money.Cents `json:"real_cost_cents"`
}

// ComputePEREffect mesure l'effet d'un versement PER : le versement se
// déduit du revenu imposable (plafonds non modélisés — indicatif).
func ComputePEREffect(taxableIncome, contribution money.Cents, partsTenths int) PEREffect {
	before := Compute(taxableIncome, partsTenths)
	after := Compute(taxableIncome-contribution, partsTenths)
	savings := before.Tax - after.Tax
	return PEREffect{
		Contribution: contribution,
		TaxBefore:    before.Tax,
		TaxAfter:     after.Tax,
		Savings:      savings,
		RealCost:     contribution - savings,
	}
}

// Deadline — une échéance fiscale française récurrente (indicative).
type Deadline struct {
	Month int    `json:"month"`
	Day   int    `json:"day"`
	Title string `json:"title"`
}

// Deadlines — les rendez-vous fiscaux de l'année.
var Deadlines = []Deadline{
	{Month: 5, Day: 25, Title: "Déclaration de revenus (date indicative)"},
	{Month: 9, Day: 15, Title: "Solde de l'impôt sur le revenu"},
	{Month: 10, Day: 15, Title: "Taxe foncière"},
	{Month: 12, Day: 15, Title: "CFE (si entreprise)"},
}
