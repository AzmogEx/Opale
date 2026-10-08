package engine

import "github.com/opale-app/opale/internal/money"

type ComparisonCheckpoint struct {
	Months int         `json:"months"`
	A      Alternative `json:"a"`
	B      Alternative `json:"b"`
	Delta  money.Cents `json:"delta_b_minus_a_cents"`
}

type ComparisonScenario struct {
	Name                string      `json:"name"`
	InvestmentReturnBps int         `json:"investment_return_bps"`
	A                   Alternative `json:"a"`
	B                   Alternative `json:"b"`
	Delta               money.Cents `json:"delta_b_minus_a_cents"`
}

type ComparisonRecommendation struct {
	Preferred string `json:"preferred"` // a | b | none
	Message   string `json:"message"`
}

// preference gives feasibility precedence over a terminal wealth advantage.
// No alternative with a funding deficit can be recommended.
func comparisonPreference(a, b Alternative, delta money.Cents) string {
	if !a.Feasible && !b.Feasible {
		return "none"
	}
	if !a.Feasible {
		return "b"
	}
	if !b.Feasible {
		return "a"
	}
	if delta > 0 {
		return "b"
	}
	if delta < 0 {
		return "a"
	}
	return "none"
}

func recommendComparison(r ComparisonResult) (ComparisonRecommendation, []string) {
	risks := []string{"Rendements hypothétiques et constants : ces scénarios ne couvrent pas une baisse ponctuelle, le risque de liquidité ni tous les coûts fiscaux et personnels."}
	if !r.A.Feasible {
		risks = append(risks, r.A.Label+" : les ressources déclarées ne financent pas toutes les échéances à l’horizon demandé.")
	}
	if !r.B.Feasible {
		risks = append(risks, r.B.Label+" : les ressources déclarées ne financent pas toutes les échéances à l’horizon demandé.")
	}
	preferred := comparisonPreference(r.A, r.B, r.Delta)
	stable := true
	for _, s := range r.Scenarios {
		if comparisonPreference(s.A, s.B, s.Delta) != preferred {
			stable = false
		}
	}
	if !stable {
		risks = append(risks, "L’alternative préférable ou sa faisabilité change selon le rendement testé ; la conclusion est sensible à cette hypothèse.")
	}
	for _, p := range r.Timeline {
		if p.Months > r.HorizonMonths && (!p.A.Feasible || !p.B.Feasible) {
			risks = append(risks, "Au-delà de l’horizon demandé, au moins une alternative manque de financement à un jalon de 5 ou 10 ans ; vérifier les détails avant un engagement long.")
			break
		}
	}
	if !r.A.Feasible && !r.B.Feasible {
		return ComparisonRecommendation{"none", "Aucune alternative n’est finançable avec les ressources déclarées. Revoir l’apport, les coûts ou le budget avant de décider."}, risks
	}
	if !stable {
		return ComparisonRecommendation{"none", "Aucune préférence robuste aux trois rendements testés. Comparer les scénarios et la réserve de liquidités avant de décider."}, risks
	}
	if preferred == "none" {
		return ComparisonRecommendation{"none", "Aucun avantage patrimonial calculé ne départage les alternatives aux rendements testés. La liquidité, les coûts non saisis et les préférences personnelles restent déterminants."}, risks
	}
	label := r.A.Label
	if preferred == "b" {
		label = r.B.Label
	}
	return ComparisonRecommendation{preferred, "Sous les seules hypothèses saisies et à l’horizon demandé, « " + label + " » est à privilégier parmi les alternatives finançables dans les trois scénarios testés. Cette conclusion reste conditionnelle aux rendements, aux coûts et au budget déclarés."}, risks
}
