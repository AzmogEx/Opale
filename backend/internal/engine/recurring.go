package engine

import (
	"sort"
	"time"

	"github.com/opale-app/opale/internal/money"
)

// TxObs — observation d'un mouvement pour la détection de récurrence (EF-026).
type TxObs struct {
	MerchantKey string
	Label       string
	OccurredOn  time.Time
	Amount      money.Cents // signé
}

// RecurringFlow — un flux récurrent détecté (abonnement, salaire, loyer…).
type RecurringFlow struct {
	MerchantKey  string      `json:"merchant_key"`
	Label        string      `json:"label"`
	Amount       money.Cents `json:"amount_cents"`   // montant type (médiane), signé
	IntervalDays int         `json:"interval_days"`  // intervalle médian observé
	Periodicity  string      `json:"periodicity"`    // weekly | monthly | quarterly | yearly
	LastDate     time.Time   `json:"last_date"`
	NextDate     time.Time   `json:"next_date"`
	Occurrences  int         `json:"occurrences"`
	Active       bool        `json:"active"` // false si une échéance a été manquée
}

// Fenêtres de périodicité (jours entre deux occurrences).
var periodicities = []struct {
	name     string
	min, max int
}{
	{"weekly", 5, 9},
	{"monthly", 25, 36},
	{"quarterly", 80, 100},
	{"yearly", 340, 390},
}

// DetectRecurring identifie les flux récurrents parmi des mouvements (EF-026).
//
// Critères, volontairement simples et déterministes :
//   - ≥ 2 occurrences d'un même marchand (clé normalisée) et même signe ;
//   - intervalles réguliers (tous dans la même fenêtre de périodicité) ;
//   - montants stables (chacun à ±25 % de la médiane).
//
// Un flux dont la prochaine échéance est dépassée de plus d'un demi-intervalle
// est marqué inactif (résilié probable).
func DetectRecurring(txs []TxObs, today time.Time) []RecurringFlow {
	byMerchant := map[string][]TxObs{}
	for _, t := range txs {
		if t.MerchantKey == "" || t.Amount == 0 {
			continue
		}
		byMerchant[t.MerchantKey] = append(byMerchant[t.MerchantKey], t)
	}

	var flows []RecurringFlow
	for key, group := range byMerchant {
		if len(group) < 2 {
			continue
		}
		sort.Slice(group, func(i, j int) bool {
			return group[i].OccurredOn.Before(group[j].OccurredOn)
		})

		// Même signe partout (un marchand mixte n'est pas un flux régulier).
		positive := group[0].Amount > 0
		sameSign := true
		for _, t := range group {
			if (t.Amount > 0) != positive {
				sameSign = false
				break
			}
		}
		if !sameSign {
			continue
		}

		// Intervalles entre occurrences consécutives.
		intervals := make([]int, 0, len(group)-1)
		for i := 1; i < len(group); i++ {
			d := int(group[i].OccurredOn.Sub(group[i-1].OccurredOn).Hours() / 24)
			if d <= 0 {
				d = 1
			}
			intervals = append(intervals, d)
		}
		med := medianInt(intervals)

		// Tous les intervalles dans la même fenêtre de périodicité ?
		period := ""
		for _, p := range periodicities {
			if med < p.min || med > p.max {
				continue
			}
			ok := true
			for _, iv := range intervals {
				if iv < p.min || iv > p.max {
					ok = false
					break
				}
			}
			if ok {
				period = p.name
			}
			break
		}
		if period == "" {
			continue
		}

		// Montants stables : chacun à ±25 % de la médiane (comparaison entière).
		amounts := make([]int64, len(group))
		for i, t := range group {
			amounts[i] = absInt64(int64(t.Amount))
		}
		medAmount := medianInt64(amounts)
		stable := true
		for _, a := range amounts {
			// |a - med| * 4 > med  ⇔  écart > 25 %
			if diff := a - medAmount; diff*4 > medAmount || -diff*4 > medAmount {
				stable = false
				break
			}
		}
		if !stable || medAmount == 0 {
			continue
		}

		last := group[len(group)-1]
		next := last.OccurredOn.AddDate(0, 0, med)
		// Inactif si l'échéance est dépassée de plus d'un demi-intervalle.
		active := today.Before(next.AddDate(0, 0, med/2))

		signed := medAmount
		if !positive {
			signed = -signed
		}
		flows = append(flows, RecurringFlow{
			MerchantKey:  key,
			Label:        last.Label,
			Amount:       money.Cents(signed),
			IntervalDays: med,
			Periodicity:  period,
			LastDate:     last.OccurredOn,
			NextDate:     next,
			Occurrences:  len(group),
			Active:       active,
		})
	}

	sort.Slice(flows, func(i, j int) bool {
		return absInt64(int64(flows[i].Amount)) > absInt64(int64(flows[j].Amount))
	})
	return flows
}

func medianInt(v []int) int {
	s := append([]int(nil), v...)
	sort.Ints(s)
	return s[len(s)/2]
}

func medianInt64(v []int64) int64 {
	s := append([]int64(nil), v...)
	sort.Slice(s, func(i, j int) bool { return s[i] < s[j] })
	return s[len(s)/2]
}

func absInt64(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}
