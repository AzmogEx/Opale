// Package categorize nettoie les libellés bancaires et propose une catégorie
// (EF-022/EF-023).
//
// P3 : moteur À RÈGLES, déterministe et instantané —
//  1. règles du profil (apprises des corrections de l'utilisateur), prioritaires ;
//  2. règles par mots-clés (enseignes françaises courantes) ;
//  3. sinon : aucune catégorie (l'app affiche « À catégoriser »).
//
// P5 branchera la cascade IA (EIA-001/002) au-dessus : le niveau local/homelab
// ne sera consulté QUE si aucune règle ne matche — les règles restent la
// mémoire rapide et privée du système.
package categorize

import (
	"regexp"
	"strings"
)

// reNoise : fragments techniques des libellés bancaires français.
var reNoise = regexp.MustCompile(
	`(?i)\b(CB|CARTE|PAIEMENT|ACHAT|VIR(EMENT)?|SEPA|INST(ANTANE)?|PRLV|PRELEVEMENT|` +
		`RETRAIT|DAB|SANS CONTACT|FACTURE|ECH(EANCE)?|EMIS( LE)?|LE)\b`,
)

// reDate : dates au fil du libellé (12/06, 12/06/26, 06/2026, 12.06.2026…).
var reDate = regexp.MustCompile(`\b\d{1,2}[./-]\d{1,4}([./-]\d{2,4})?\b`)

// reDigits : jetons purement numériques (n° de carte, références, codes
// postaux, jours) — quelle que soit leur longueur.
var reDigits = regexp.MustCompile(`\b[0-9*]+\b`)

// reSpaces : espaces multiples après nettoyage.
var reSpaces = regexp.MustCompile(`\s+`)

// MerchantKey normalise un libellé bancaire brut en « clé marchand » stable :
// majuscules, sans bruit technique, sans dates ni numéros.
//
//	"CB CARREFOUR MARKET 12/06 PARIS 75" → "CARREFOUR MARKET PARIS"
//
// C'est la clé des règles d'apprentissage (merchant_rules).
func MerchantKey(rawLabel string) string {
	s := strings.ToUpper(strings.TrimSpace(rawLabel))
	s = reDate.ReplaceAllString(s, " ")
	s = reNoise.ReplaceAllString(s, " ")
	s = reDigits.ReplaceAllString(s, " ")
	s = strings.Map(func(r rune) rune {
		switch r {
		case '*', '/', '\\', '-', '_', '.', ',', ':', ';', '#', '(', ')':
			return ' '
		}
		return r
	}, s)
	return reSpaces.ReplaceAllString(strings.TrimSpace(s), " ")
}

// CleanLabel produit un libellé lisible pour l'affichage (EF-023) :
// la clé marchand en capitalisation de titre.
//
//	"CB CARREFOUR MARKET 12/06 PARIS 75" → "Carrefour Market Paris"
func CleanLabel(rawLabel string) string {
	key := MerchantKey(rawLabel)
	if key == "" {
		return strings.TrimSpace(rawLabel)
	}
	words := strings.Split(strings.ToLower(key), " ")
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}

// keywordRule : mot-clé (contenu dans la clé marchand) → nom de catégorie.
type keywordRule struct {
	keyword  string
	category string
}

// Règles par mots-clés — enseignes et motifs français courants.
// L'ordre compte : première correspondance gagnante.
var keywordRules = []keywordRule{
	// Revenus (sur libellé, le signe est vérifié par l'appelant)
	{"SALAIRE", "Revenus"}, {"PAIE ", "Revenus"}, {"CAF", "Revenus"},
	{"POLE EMPLOI", "Revenus"}, {"FRANCE TRAVAIL", "Revenus"},

	// Courses
	{"CARREFOUR", "Courses"}, {"LECLERC", "Courses"}, {"AUCHAN", "Courses"},
	{"LIDL", "Courses"}, {"ALDI", "Courses"}, {"INTERMARCHE", "Courses"},
	{"MONOPRIX", "Courses"}, {"FRANPRIX", "Courses"}, {"CASINO", "Courses"},
	{"SUPER U", "Courses"}, {"PICARD", "Courses"}, {"BIOCOOP", "Courses"},
	{"GRAND FRAIS", "Courses"},

	// Restaurants / livraison
	{"MCDONALD", "Restaurants"}, {"MCDO", "Restaurants"}, {"BURGER KING", "Restaurants"},
	{"KFC", "Restaurants"}, {"SUBWAY", "Restaurants"}, {"DELIVEROO", "Restaurants"},
	{"UBER EATS", "Restaurants"}, {"RESTAURANT", "Restaurants"}, {"BRASSERIE", "Restaurants"},
	{"BOULANGERIE", "Restaurants"}, {"PIZZ", "Restaurants"}, {"SUSHI", "Restaurants"},

	// Transport
	{"SNCF", "Transport"}, {"RATP", "Transport"}, {"UBER", "Transport"},
	{"BLABLACAR", "Transport"}, {"TOTALENERGIES", "Transport"}, {"TOTAL ", "Transport"},
	{"ESSO", "Transport"}, {"SHELL", "Transport"}, {"BP ", "Transport"},
	{"AUTOROUTE", "Transport"}, {"VINCI", "Transport"}, {"PARKING", "Transport"},
	{"NAVIGO", "Transport"}, {"TISSEO", "Transport"}, {"TCL", "Transport"},

	// Logement / énergie / télécom
	{"EDF", "Logement"}, {"ENGIE", "Logement"}, {"VEOLIA", "Logement"},
	{"LOYER", "Logement"}, {"FONCIA", "Logement"}, {"NEXITY", "Logement"},
	{"ORANGE", "Logement"}, {"SFR", "Logement"}, {"BOUYGUES TEL", "Logement"},
	{"FREE MOBILE", "Logement"}, {"FREE HAUTDEBIT", "Logement"}, {"FREE ", "Logement"},
	{"SOSH", "Logement"}, {"RED BY", "Logement"},

	// Abonnements numériques
	{"NETFLIX", "Abonnements"}, {"SPOTIFY", "Abonnements"}, {"DISNEY", "Abonnements"},
	{"CANAL", "Abonnements"}, {"AMAZON PRIME", "Abonnements"}, {"YOUTUBE PREMIUM", "Abonnements"},
	{"APPLE COM BILL", "Abonnements"}, {"ICLOUD", "Abonnements"}, {"DEEZER", "Abonnements"},
	{"BASIC FIT", "Abonnements"}, {"FITNESS PARK", "Abonnements"},

	// Santé
	{"PHARMACIE", "Santé"}, {"DOCTEUR", "Santé"}, {"DR ", "Santé"},
	{"CPAM", "Santé"}, {"MUTUELLE", "Santé"}, {"LABORATOIRE", "Santé"},
	{"DENTAIRE", "Santé"}, {"OPTIC", "Santé"},

	// Loisirs
	{"CINEMA", "Loisirs"}, {"UGC", "Loisirs"}, {"PATHE", "Loisirs"},
	{"GAUMONT", "Loisirs"}, {"STEAM", "Loisirs"}, {"PLAYSTATION", "Loisirs"},
	{"NINTENDO", "Loisirs"}, {"XBOX", "Loisirs"}, {"FNAC SPECTACLE", "Loisirs"},

	// Shopping
	{"AMAZON", "Shopping"}, {"FNAC", "Shopping"}, {"DARTY", "Shopping"},
	{"BOULANGER", "Shopping"}, {"ZALANDO", "Shopping"}, {"SHEIN", "Shopping"},
	{"ZARA", "Shopping"}, {"H M ", "Shopping"}, {"DECATHLON", "Shopping"},
	{"IKEA", "Shopping"}, {"LEROY MERLIN", "Shopping"}, {"ACTION", "Shopping"},

	// Voyages
	{"AIRBNB", "Voyages"}, {"BOOKING", "Voyages"}, {"HOTEL", "Voyages"},
	{"AIR FRANCE", "Voyages"}, {"EASYJET", "Voyages"}, {"RYANAIR", "Voyages"},
	{"TRANSAVIA", "Voyages"},

	// Impôts
	{"DGFIP", "Impôts"}, {"IMPOT", "Impôts"}, {"TRESOR PUBLIC", "Impôts"},
	{"URSSAF", "Impôts"},
}

// SuggestCategory propose un nom de catégorie pour un libellé brut, ou ""
// si aucune règle ne matche.
//
// profileRules : règles apprises du profil (clé marchand → nom de catégorie),
// prioritaires sur les mots-clés. amountCents permet les heuristiques de signe
// (un crédit « VIR M DUPONT » n'est pas un abonnement).
func SuggestCategory(profileRules map[string]string, rawLabel string, amountCents int64) string {
	key := MerchantKey(rawLabel)
	if key == "" {
		return ""
	}

	// 1. Les corrections de l'utilisateur priment (EF-022, apprentissage).
	if cat, ok := profileRules[key]; ok {
		return cat
	}

	// 2. Règles par mots-clés.
	padded := " " + key + " "
	for _, r := range keywordRules {
		if strings.Contains(padded, " "+r.keyword) || strings.Contains(key, r.keyword) {
			return r.category
		}
	}

	// 3. Heuristiques génériques : un virement entrant = Revenus probables,
	// un virement sortant = Virements.
	if strings.Contains(strings.ToUpper(rawLabel), "VIR") {
		if amountCents > 0 {
			return "Revenus"
		}
		return "Virements"
	}

	return ""
}
