package nlq

import (
	"testing"
	"time"
)

var now = time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC)
var cats = []string{"Courses", "Loisirs", "Transport", "Santé"}
var merchants = []string{"Carrefour Lyon", "Netflix", "Pharmacie Lafayette"}

func TestParseCategoryAndMonth(t *testing.T) {
	q := Parse("Combien en courses en mars ?", cats, merchants, now)
	if !q.Confident() {
		t.Fatal("question de données non reconnue")
	}
	if q.CategoryName != "Courses" {
		t.Fatalf("catégorie %q, attendu Courses", q.CategoryName)
	}
	if q.From.Month() != 3 || q.From.Year() != 2026 {
		t.Fatalf("période %v, attendu mars 2026", q.From)
	}
	if q.Income {
		t.Fatal("dépenses attendues par défaut")
	}
}

func TestParseFutureMonthMeansLastYear(t *testing.T) {
	// En juillet 2026, « en décembre » = décembre 2025.
	q := Parse("combien j'ai dépensé en décembre", cats, merchants, now)
	if q.From.Year() != 2025 || q.From.Month() != 12 {
		t.Fatalf("attendu décembre 2025, obtenu %v", q.From)
	}
}

func TestParseThisMonthAndIncome(t *testing.T) {
	q := Parse("Combien j'ai gagné ce mois-ci ?", cats, merchants, now)
	if !q.Income {
		t.Fatal("revenus attendus")
	}
	if q.From.Month() != 7 || q.From.Year() != 2026 {
		t.Fatalf("période %v, attendu juillet 2026", q.From)
	}
}

func TestParseMerchant(t *testing.T) {
	q := Parse("Combien chez carrefour cette année ?", cats, merchants, now)
	if q.MerchantQuery != "carrefour" {
		t.Fatalf("marchand %q, attendu carrefour", q.MerchantQuery)
	}
	if q.From.Month() != 1 || q.To.Month() != 12 {
		t.Fatal("période « cette année » attendue")
	}
}

func TestParseKnownMerchantLabel(t *testing.T) {
	q := Parse("combien m'a coûté netflix ce mois", cats, merchants, now)
	if q.MerchantQuery != "Netflix" {
		t.Fatalf("marchand %q, attendu Netflix", q.MerchantQuery)
	}
}

func TestNotADataQuestion(t *testing.T) {
	// Pas de déclencheur → la cascade IA garde la main.
	q := Parse("Comment va mon épargne ?", cats, merchants, now)
	if q.Confident() {
		t.Fatal("ne doit pas être traitée comme une question de données")
	}
	// Déclencheur mais ni période ni cible → pas confiant non plus.
	q = Parse("combien vaut une maison ?", cats, merchants, now)
	if q.Confident() {
		t.Fatal("sans période ni cible, la cascade doit garder la main")
	}
}
