package csvimport

// Import OFX (EF-070) — le format d'export bancaire « Money/Quicken ».
// OFX 1.x est du SGML sans balises fermantes ; OFX 2.x du XML. Ce parseur
// tolérant lit les deux : il ne s'intéresse qu'aux blocs <STMTTRN> et aux
// champs TRNAMT / DTPOSTED / NAME / MEMO.

import (
	"fmt"
	"strings"
	"time"

	"github.com/opale-app/opale/internal/money"
)

// IsOFX détecte un contenu OFX (en-tête 1.x ou balise 2.x).
func IsOFX(content string) bool {
	head := strings.ToUpper(content)
	if len(head) > 512 {
		head = head[:512]
	}
	return strings.Contains(head, "OFXHEADER") || strings.Contains(head, "<OFX>")
}

// ParseOFX extrait les mouvements d'un relevé OFX.
func ParseOFX(content string) ([]Row, error) { return ParseOFXCurrency(content, 2) }
func ParseOFXCurrency(content string, exponent int) ([]Row, error) {
	if len(content) > 5<<20 {
		return nil, fmt.Errorf("ofx: fichier supérieur à 5 Mio")
	}
	content = ensureUTF8(content)
	upper := strings.ToUpper(content)

	var rows []Row
	cursor := 0
	for {
		start := strings.Index(upper[cursor:], "<STMTTRN>")
		if start < 0 {
			break
		}
		start += cursor
		end := strings.Index(upper[start:], "</STMTTRN>")
		blockEnd := len(content)
		if end >= 0 {
			blockEnd = start + end
		} else {
			// OFX 1.x sans fermeture : le bloc va jusqu'au prochain <STMTTRN>
			// ou à la fin.
			if next := strings.Index(upper[start+9:], "<STMTTRN>"); next >= 0 {
				blockEnd = start + 9 + next
			}
		}
		block := content[start:blockEnd]
		cursor = blockEnd

		amountRaw := ofxField(block, "TRNAMT")
		dateRaw := ofxField(block, "DTPOSTED")
		label := ofxField(block, "NAME")
		if label == "" {
			label = ofxField(block, "MEMO")
		}
		if amountRaw == "" || dateRaw == "" {
			return nil, fmt.Errorf("ofx: mouvement %d incomplet, aucun import effectué", len(rows)+1)
		}

		amount, err := money.ParseMinor(amountRaw, exponent)
		if err != nil {
			return nil, fmt.Errorf("ofx: montant illisible %q : %w", amountRaw, err)
		}
		// DTPOSTED : YYYYMMDD, parfois suivi de l'heure/fuseau.
		if len(dateRaw) < 8 {
			return nil, fmt.Errorf("ofx: date illisible %q", dateRaw)
		}
		day, err := time.Parse("20060102", dateRaw[:8])
		if err != nil {
			return nil, fmt.Errorf("ofx: date illisible %q : %w", dateRaw, err)
		}
		if label == "" {
			label = "Mouvement bancaire"
		}

		rows = append(rows, Row{SourceID: ofxField(block, "FITID"), Amount: amount, OccurredOn: day, RawLabel: label})
	}

	if len(rows) > 10000 {
		return nil, fmt.Errorf("ofx: maximum 10000 mouvements, aucun import effectué")
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("ofx: aucun mouvement <STMTTRN> trouvé")
	}
	return rows, nil
}

// ofxField lit la valeur d'un champ : « <TAG>valeur » jusqu'à la prochaine
// balise ou fin de ligne (SGML 1.x et XML 2.x confondus).
func ofxField(block, tag string) string {
	upper := strings.ToUpper(block)
	idx := strings.Index(upper, "<"+tag+">")
	if idx < 0 {
		return ""
	}
	rest := block[idx+len(tag)+2:]
	if cut := strings.IndexAny(rest, "<\r\n"); cut >= 0 {
		rest = rest[:cut]
	}
	return strings.TrimSpace(rest)
}
