package csvimport

import (
	"testing"

	"github.com/opale-app/opale/internal/money"
)

// OFX 1.x — SGML sans balises fermantes, tel que l'exportent les banques FR.
const ofx1 = `OFXHEADER:100
DATA:OFXSGML
VERSION:102

<OFX>
<BANKMSGSRSV1><STMTTRNRS><STMTRS><BANKTRANLIST>
<STMTTRN>
<TRNTYPE>DEBIT
<DTPOSTED>20260703
<TRNAMT>-42.90
<FITID>0001
<NAME>CARTE 03/07 CARREFOUR LYON
<STMTTRN>
<TRNTYPE>CREDIT
<DTPOSTED>20260701120000
<TRNAMT>2600.00
<FITID>0002
<MEMO>VIR SALAIRE JUILLET
</BANKTRANLIST></STMTRS></STMTTRNRS></BANKMSGSRSV1>
</OFX>`

// OFX 2.x — XML avec balises fermantes.
const ofx2 = `<?xml version="1.0"?>
<OFX><BANKMSGSRSV1><STMTTRNRS><STMTRS><BANKTRANLIST>
<STMTTRN><TRNTYPE>DEBIT</TRNTYPE><DTPOSTED>20260705</DTPOSTED><TRNAMT>-9.99</TRNAMT><NAME>NETFLIX.COM</NAME></STMTTRN>
</BANKTRANLIST></STMTRS></STMTTRNRS></BANKMSGSRSV1></OFX>`

func TestIsOFX(t *testing.T) {
	if !IsOFX(ofx1) || !IsOFX(ofx2) {
		t.Fatal("OFX non détecté")
	}
	if IsOFX("Date;Libellé;Montant\n01/07/2026;TEST;-1,00") {
		t.Fatal("un CSV ne doit pas être détecté comme OFX")
	}
}

func TestParseOFX1SGML(t *testing.T) {
	rows, err := ParseOFX(ofx1)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("2 mouvements attendus, obtenu %d", len(rows))
	}
	if rows[0].Amount != money.Cents(-4_290) {
		t.Fatalf("montant %d, attendu -4290", rows[0].Amount)
	}
	if rows[0].RawLabel != "CARTE 03/07 CARREFOUR LYON" {
		t.Fatalf("libellé %q", rows[0].RawLabel)
	}
	// DTPOSTED avec heure : seuls les 8 premiers caractères comptent.
	if rows[1].OccurredOn.Day() != 1 || rows[1].OccurredOn.Month() != 7 {
		t.Fatalf("date %v, attendu 1er juillet", rows[1].OccurredOn)
	}
	if rows[1].RawLabel != "VIR SALAIRE JUILLET" {
		t.Fatalf("le MEMO doit servir de libellé, obtenu %q", rows[1].RawLabel)
	}
}

func TestParseOFX2XML(t *testing.T) {
	rows, err := ParseOFX(ofx2)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Amount != money.Cents(-999) {
		t.Fatalf("mouvement Netflix attendu, obtenu %+v", rows)
	}
}
