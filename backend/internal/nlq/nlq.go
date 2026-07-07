// Package nlq — recherche en langage naturel sur les transactions (EF-050).
//
// « Combien en courses en mars ? » → le PARSEUR (déterministe, testé)
// extrait période / catégorie / marchand / sens, puis l'API fouille les
// transactions et répond avec les chiffres EXACTS du moteur. Aucun LLM
// n'invente de montant (EIA-040) — la cascade IA ne sert qu'aux questions
// qui ne sont pas des questions de données.
package nlq

import (
	"strings"
	"time"
)

// Query — ce que le parseur a compris de la question.
type Query struct {
	// Période résolue (mois calendaire, ou zéro si non détectée).
	From, To time.Time
	// PeriodLabel : « en mars 2026 », « ce mois-ci »… pour la réponse.
	PeriodLabel string
	// CategoryName : catégorie reconnue (nom exact du référentiel).
	CategoryName string
	// MerchantQuery : fragment de marchand reconnu (« chez carrefour »).
	MerchantQuery string
	// Income : question sur les revenus plutôt que les dépenses.
	Income bool
}

// Confident dit si la question est bien une question de données :
// il faut au moins une période OU une cible (catégorie/marchand).
func (q Query) Confident() bool {
	return !q.From.IsZero() || q.CategoryName != "" || q.MerchantQuery != ""
}

var months = map[string]time.Month{
	"janvier": 1, "fevrier": 2, "février": 2, "mars": 3, "avril": 4,
	"mai": 5, "juin": 6, "juillet": 7, "aout": 8, "août": 8,
	"septembre": 9, "octobre": 10, "novembre": 11, "decembre": 12, "décembre": 12,
}

// triggers : sans l'un de ces mots, ce n'est pas une question de données.
var triggers = []string{
	"combien", "dépensé", "depense", "dépense", "cout", "coût", "coûté",
	"gagné", "gagne", "reçu", "recu", "total",
}

// normalize : minuscules + apostrophes/ponctuation neutralisées.
func normalize(s string) string {
	s = strings.ToLower(s)
	replacer := strings.NewReplacer("'", " ", "’", " ", "?", " ", "!", " ",
		",", " ", ".", " ", "-", " ")
	return " " + replacer.Replace(s) + " "
}

// Parse analyse la question. categories = noms du référentiel du profil ;
// merchants = libellés nettoyés connus (pour « chez X »).
// now sert de référence pour « ce mois-ci » / « le mois dernier ».
func Parse(question string, categories []string, merchants []string, now time.Time) Query {
	q := Query{}
	text := normalize(question)

	// Une question de données contient un déclencheur (« combien »…).
	triggered := false
	for _, t := range triggers {
		if strings.Contains(text, t) {
			triggered = true
			break
		}
	}
	if !triggered {
		return Query{}
	}

	// ── Sens : revenus ou dépenses (défaut : dépenses) ────────────────────
	for _, w := range []string{"gagné", "gagne", "reçu", "recu", "revenu"} {
		if strings.Contains(text, w) {
			q.Income = true
			break
		}
	}

	// ── Période ───────────────────────────────────────────────────────────
	switch {
	case strings.Contains(text, "ce mois"):
		q.From = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		q.To = q.From.AddDate(0, 1, -1)
		q.PeriodLabel = "ce mois-ci"
	case strings.Contains(text, "mois dernier"):
		q.From = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, -1, 0)
		q.To = q.From.AddDate(0, 1, -1)
		q.PeriodLabel = "le mois dernier"
	case strings.Contains(text, "cette annee") || strings.Contains(text, "cette année"):
		q.From = time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
		q.To = time.Date(now.Year(), 12, 31, 0, 0, 0, 0, time.UTC)
		q.PeriodLabel = "cette année"
	default:
		for name, m := range months {
			if strings.Contains(text, " "+name+" ") ||
				strings.Contains(text, " "+name) && strings.HasSuffix(strings.TrimSpace(text), name) {
				year := now.Year()
				// Un mois futur sans année explicite = l'an passé ? Non :
				// on prend le dernier mois écoulé portant ce nom.
				if m > now.Month() {
					year--
				}
				q.From = time.Date(year, m, 1, 0, 0, 0, 0, time.UTC)
				q.To = q.From.AddDate(0, 1, -1)
				q.PeriodLabel = "en " + name + " " + q.From.Format("2006")
				break
			}
		}
	}

	// ── Catégorie : le nom du référentiel présent dans la question ────────
	for _, c := range categories {
		if strings.Contains(text, " "+strings.ToLower(c)+" ") {
			q.CategoryName = c
			break
		}
	}

	// ── Marchand : « chez X » ou un libellé connu cité tel quel ──────────
	if idx := strings.Index(text, " chez "); idx >= 0 {
		rest := strings.Fields(text[idx+6:])
		if len(rest) > 0 {
			q.MerchantQuery = rest[0]
		}
	}
	if q.MerchantQuery == "" {
		for _, m := range merchants {
			lower := strings.ToLower(m)
			if len(lower) >= 4 && strings.Contains(text, lower) {
				q.MerchantQuery = m
				break
			}
		}
	}

	return q
}
