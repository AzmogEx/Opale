package categorize

import "testing"

func TestMerchantKey(t *testing.T) {
	cases := map[string]string{
		"CB CARREFOUR MARKET 12/06 PARIS 75":     "CARREFOUR MARKET PARIS",
		"PRLV SEPA NETFLIX SARL 06/2026":         "NETFLIX SARL",
		"VIR SEPA SALAIRE ACME CORP":             "SALAIRE ACME CORP",
		"PAIEMENT CB 1234**** SNCF CONNECT":      "SNCF CONNECT",
		"RETRAIT DAB 15/06 PARIS":                "PARIS",
		"  cb   Uber   *Eats 99  ":               "UBER EATS",
	}
	for raw, want := range cases {
		if got := MerchantKey(raw); got != want {
			t.Errorf("MerchantKey(%q) = %q, attendu %q", raw, got, want)
		}
	}
}

func TestCleanLabel(t *testing.T) {
	if got := CleanLabel("CB CARREFOUR MARKET 12/06 PARIS 75"); got != "Carrefour Market Paris" {
		t.Errorf("CleanLabel = %q", got)
	}
	// Libellé entièrement « bruit » : on garde le brut plutôt que du vide.
	if got := CleanLabel("12/06 1234"); got == "" {
		t.Error("CleanLabel ne doit jamais renvoyer une chaîne vide")
	}
}

func TestSuggestCategoryKeywords(t *testing.T) {
	cases := []struct {
		raw    string
		amount int64
		want   string
	}{
		{"CB CARREFOUR MARKET PARIS", -4520, "Courses"},
		{"PRLV NETFLIX SARL", -1399, "Abonnements"},
		{"CB SNCF CONNECT", -8900, "Transport"},
		{"CB UBER EATS PARIS", -2350, "Restaurants"},
		{"CB UBER TRIP", -1250, "Transport"},
		{"PRLV EDF CLIENTS", -6540, "Logement"},
		{"VIR SEPA SALAIRE ACME", 340000, "Revenus"},
		{"CB PHARMACIE LAFAYETTE", -1230, "Santé"},
		{"PRLV DGFIP IMPOT REVENUS", -42000, "Impôts"},
		{"CB AMAZON EU SARL", -5999, "Shopping"},
		{"CB ZZZZZ INCONNU", -100, ""},
	}
	for _, c := range cases {
		if got := SuggestCategory(nil, c.raw, c.amount); got != c.want {
			t.Errorf("SuggestCategory(%q) = %q, attendu %q", c.raw, got, c.want)
		}
	}
}

func TestSuggestCategoryProfileRulesPriority(t *testing.T) {
	// L'utilisateur a corrigé : pour lui, Amazon = Loisirs. Sa règle prime.
	rules := map[string]string{"AMAZON EU SARL": "Loisirs"}
	if got := SuggestCategory(rules, "CB AMAZON EU SARL", -5999); got != "Loisirs" {
		t.Errorf("la règle du profil doit primer, obtenu %q", got)
	}
}

func TestSuggestCategoryVirHeuristic(t *testing.T) {
	if got := SuggestCategory(nil, "VIR M DUPONT JEAN", 50000); got != "Revenus" {
		t.Errorf("virement entrant → Revenus, obtenu %q", got)
	}
	if got := SuggestCategory(nil, "VIR M DUPONT JEAN", -50000); got != "Virements" {
		t.Errorf("virement sortant → Virements, obtenu %q", got)
	}
}
