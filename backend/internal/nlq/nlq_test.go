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

func TestParseComparisonTwoMonths(t *testing.T) {
	q := Parse("Compare mes dépenses en courses en mars et avril", cats, merchants, now)
	if !q.Confident() || !q.IsComparison() {
		t.Fatal("comparaison non reconnue")
	}
	if q.From.Month() != 3 || q.CompareFrom.Month() != 4 {
		t.Fatalf("périodes %v / %v, attendu mars puis avril", q.From, q.CompareFrom)
	}
	if q.CategoryName != "Courses" {
		t.Fatalf("catégorie %q, attendu Courses", q.CategoryName)
	}
}

func TestParseComparisonVs(t *testing.T) {
	q := Parse("combien en transport en janvier vs février ?", cats, merchants, now)
	if !q.IsComparison() {
		t.Fatal("comparaison vs non reconnue")
	}
	if q.From.Month() != 1 || q.CompareFrom.Month() != 2 {
		t.Fatalf("périodes %v / %v, attendu janvier puis février", q.From, q.CompareFrom)
	}
	if q.From.Year() != 2026 || q.CompareFrom.Year() != 2026 {
		t.Fatal("mois passés : année courante attendue")
	}
}

func TestSingleMonthStillWorks(t *testing.T) {
	// Un seul mois cité → pas de comparaison.
	q := Parse("combien en mars ?", cats, merchants, now)
	if q.IsComparison() {
		t.Fatal("un seul mois ne doit pas déclencher de comparaison")
	}
	if q.From.Month() != 3 {
		t.Fatalf("période %v, attendu mars", q.From)
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
func TestExplicitLeapYear(t *testing.T) {
	q := Parse("combien en février 2024", nil, nil, time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC))
	if q.From.Year() != 2024 || q.To.Day() != 29 {
		t.Fatal(q)
	}
}

func TestExplicitIndependentPeriods(t *testing.T) {
	for _, tc := range []struct{ question, from, to, compareFrom, compareTo string }{
		{"Compare mars2024 et mars2025", "2024-03-01", "2024-03-31", "2025-03-01", "2025-03-31"},
		{"Compare mes courses en décembre 2024 et février 2025", "2024-12-01", "2024-12-31", "2025-02-01", "2025-02-28"},
		{"Compare février 2024 et février 2025", "2024-02-01", "2024-02-29", "2025-02-01", "2025-02-28"},
		{"Compare février 2000 et février 2100", "2000-02-01", "2000-02-29", "2100-02-01", "2100-02-28"},
		{"Combien en2024", "2024-01-01", "2024-12-31", "", ""},
		{"Compare mes courses en 2024 et 2025", "2024-01-01", "2024-12-31", "2025-01-01", "2025-12-31"},
		{"Compare mars de 2024 et avril de 2025", "2024-03-01", "2024-03-31", "2025-04-01", "2025-04-30"},
		{"Compare mars et avril 2024", "2024-03-01", "2024-03-31", "2024-04-01", "2024-04-30"},
		{"Compare février 2024 et 2025", "2024-02-01", "2024-02-29", "2025-02-01", "2025-02-28"},
	} {
		t.Run(tc.question, func(t *testing.T) {
			q := Parse(tc.question, cats, merchants, now)
			if !q.Confident() || q.From.Format("2006-01-02") != tc.from || q.To.Format("2006-01-02") != tc.to {
				t.Fatalf("first period %+v", q)
			}
			if tc.compareFrom == "" {
				if q.IsComparison() {
					t.Fatalf("unexpected comparison %+v", q)
				}
				return
			}
			if !q.IsComparison() || q.CompareFrom.Format("2006-01-02") != tc.compareFrom || q.CompareTo.Format("2006-01-02") != tc.compareTo {
				t.Fatalf("second period %+v", q)
			}
		})
	}
}

func TestAmbiguousOrExcessPeriodsNeverConfident(t *testing.T) {
	for _, question := range []string{
		"Compare mes courses en mars 2024, mars 2025 et mars 2026",
		"Compare chez carrefour en 2023 2024 et 2025",
		"Combien de courses en janvier, février et mars",
		"Compare mars et avril 2024 et 2025",
		"Compare en mars 2024 et ce mois",
	} {
		q := Parse(question, cats, merchants, now)
		if q.Confident() || q.IsComparison() {
			t.Fatalf("silently truncated %q: %+v", question, q)
		}
	}
}
