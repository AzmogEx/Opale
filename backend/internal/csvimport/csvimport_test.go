package csvimport

import (
	"testing"

	"github.com/opale-app/opale/internal/money"
)

func TestParseFrenchBankFormat(t *testing.T) {
	// Format type Crédit Agricole / BNP : point-virgule, dates FR, virgule décimale.
	csv := `Date;Libellé;Montant
15/06/2026;CB CARREFOUR MARKET PARIS;-45,20
14/06/2026;PRLV NETFLIX;-13,99
01/06/2026;VIR SEPA SALAIRE ACME;3 400,00`

	rows, err := Parse(csv)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 {
		t.Fatalf("attendu 3 lignes, obtenu %d", len(rows))
	}
	if rows[0].Amount != money.Cents(-4520) {
		t.Errorf("montant[0] = %d, attendu -4520", rows[0].Amount)
	}
	if rows[2].Amount != money.Cents(340000) {
		t.Errorf("montant[2] = %d, attendu 340000 (espace de milliers)", rows[2].Amount)
	}
	if rows[0].OccurredOn.Format("2006-01-02") != "2026-06-15" {
		t.Errorf("date[0] = %s", rows[0].OccurredOn)
	}
}

func TestParseDebitCreditColumns(t *testing.T) {
	// Format type Banque Populaire : colonnes débit / crédit séparées.
	csv := `Date;Libellé;Débit;Crédit
15/06/2026;CB LECLERC;45,20;
01/06/2026;VIR SALAIRE;;3400,00`

	rows, err := Parse(csv)
	if err != nil {
		t.Fatal(err)
	}
	if rows[0].Amount != money.Cents(-4520) {
		t.Errorf("débit doit être négatif, obtenu %d", rows[0].Amount)
	}
	if rows[1].Amount != money.Cents(340000) {
		t.Errorf("crédit doit être positif, obtenu %d", rows[1].Amount)
	}
}

func TestParseCommaDelimiterISODates(t *testing.T) {
	// Format type export néo-banque : virgule, dates ISO, point décimal.
	csv := `date,label,amount
2026-06-15,CARREFOUR PARIS,-45.20
2026-06-01,SALAIRE,3400.00`

	rows, err := Parse(csv)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || rows[0].Amount != money.Cents(-4520) {
		t.Fatalf("parse virgule/ISO : %+v", rows)
	}
}

func TestParseSkipsPreamble(t *testing.T) {
	// Certaines banques préfixent d'infos de compte.
	csv := `Compte;12345678901
Solde au 15/06/2026;1 234,56

Date;Libellé;Montant
15/06/2026;CB LIDL;-23,45`

	rows, err := Parse(csv)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Amount != money.Cents(-2345) {
		t.Fatalf("le préambule doit être ignoré : %+v", rows)
	}
}

func TestParseNoHeader(t *testing.T) {
	// Sans entête : convention date;libellé;montant.
	csv := `15/06/2026;CB AUCHAN;-12,00`
	rows, err := Parse(csv)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Amount != money.Cents(-1200) {
		t.Fatalf("sans entête : %+v", rows)
	}
}

func TestParseWindows1252(t *testing.T) {
	// « Libellé » encodé en Windows-1252 (é = 0xE9).
	raw := "Date;Libell\xe9;Montant\n15/06/2026;CB CIN\xc9MA UGC;-11,50"
	rows, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("latin-1 : attendu 1 ligne, obtenu %d", len(rows))
	}
	if rows[0].RawLabel != "CB CINÉMA UGC" {
		t.Errorf("libellé transcodé = %q", rows[0].RawLabel)
	}
}

func TestParseEmpty(t *testing.T) {
	if _, err := Parse("rien du tout"); err == nil {
		t.Fatal("erreur attendue sur contenu vide")
	}
}
