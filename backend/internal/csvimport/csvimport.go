// Package csvimport lit les exports CSV des banques françaises (EF-021).
//
// Tolérant par construction : délimiteur détecté (';' ',' ou tab), entête
// reconnue par mots-clés (date / libellé / montant / débit / crédit),
// dates aux formats français ou ISO, montants « 1 234,56 € », encodage
// Windows-1252 converti si le fichier n'est pas de l'UTF-8 valide.
package csvimport

import (
	"encoding/csv"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/opale-app/opale/internal/money"
)

// Row — une ligne de relevé prête à insérer.
type Row struct {
	OccurredOn time.Time
	Amount     money.Cents // signé : + crédit, − débit
	RawLabel   string
}

// ErrEmpty est renvoyé quand aucun mouvement n'est trouvé.
var ErrEmpty = errors.New("csvimport: aucun mouvement détecté")

var dateLayouts = []string{
	"02/01/2006", "02/01/06", "2006-01-02", "02-01-2006", "02.01.2006",
}

// Parse lit un export CSV bancaire complet.
func Parse(data string) ([]Row, error) {
	data = ensureUTF8(data)
	data = strings.TrimPrefix(data, "\uFEFF") // BOM UTF-8

	lines := strings.Split(strings.ReplaceAll(data, "\r\n", "\n"), "\n")
	// Certaines banques préfixent le CSV de lignes d'info (solde, IBAN…) :
	// on cherche la première ligne qui ressemble à une entête ou un mouvement.
	start := 0
	for i, l := range lines {
		if looksLikeHeader(l) || looksLikeMovement(l) {
			start = i
			break
		}
	}
	body := strings.Join(lines[start:], "\n")

	delim := detectDelimiter(lines[start])
	r := csv.NewReader(strings.NewReader(body))
	r.Comma = delim
	r.FieldsPerRecord = -1 // lignes irrégulières tolérées
	r.LazyQuotes = true

	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("csvimport: lecture CSV : %w", err)
	}
	if len(records) == 0 {
		return nil, ErrEmpty
	}

	cols, hasHeader := mapColumns(records[0])
	if hasHeader {
		records = records[1:]
	}

	var rows []Row
	for _, rec := range records {
		row, ok := parseRecord(rec, cols)
		if ok {
			rows = append(rows, row)
		}
	}
	if len(rows) == 0 {
		return nil, ErrEmpty
	}
	return rows, nil
}

// columns : index de chaque champ (-1 = absent).
type columns struct {
	date, label, amount, debit, credit int
}

// mapColumns identifie les colonnes depuis l'entête. Sans entête reconnue,
// convention date;libellé;montant.
func mapColumns(header []string) (columns, bool) {
	c := columns{date: -1, label: -1, amount: -1, debit: -1, credit: -1}
	found := false
	for i, h := range header {
		key := normalizeHeader(h)
		switch {
		case c.date == -1 && strings.Contains(key, "date"):
			c.date, found = i, true
		case c.label == -1 && (strings.Contains(key, "libell") ||
			strings.Contains(key, "label") || strings.Contains(key, "description") ||
			strings.Contains(key, "designation") || strings.Contains(key, "nature")):
			c.label, found = i, true
		case c.debit == -1 && strings.Contains(key, "debit"):
			c.debit, found = i, true
		case c.credit == -1 && strings.Contains(key, "credit"):
			c.credit, found = i, true
		case c.amount == -1 && (strings.Contains(key, "montant") || strings.Contains(key, "amount")):
			c.amount, found = i, true
		}
	}
	if !found {
		return columns{date: 0, label: 1, amount: 2, debit: -1, credit: -1}, false
	}
	if c.date == -1 {
		c.date = 0
	}
	if c.label == -1 {
		c.label = 1
	}
	return c, true
}

func parseRecord(rec []string, c columns) (Row, bool) {
	get := func(i int) string {
		if i < 0 || i >= len(rec) {
			return ""
		}
		return strings.TrimSpace(rec[i])
	}

	date, ok := parseDate(get(c.date))
	if !ok {
		return Row{}, false
	}

	label := get(c.label)
	if label == "" {
		return Row{}, false
	}

	var amount money.Cents
	switch {
	case c.amount >= 0 && get(c.amount) != "":
		v, err := parseAmount(get(c.amount))
		if err != nil {
			return Row{}, false
		}
		amount = v
	case c.debit >= 0 || c.credit >= 0:
		// Colonnes séparées : débit compté négatif, crédit positif.
		if d := get(c.debit); d != "" {
			v, err := parseAmount(d)
			if err != nil {
				return Row{}, false
			}
			amount = -money.Cents(abs(int64(v)))
		}
		if cr := get(c.credit); cr != "" {
			v, err := parseAmount(cr)
			if err != nil {
				return Row{}, false
			}
			amount = money.Cents(abs(int64(v)))
		}
	default:
		return Row{}, false
	}
	if amount == 0 {
		return Row{}, false
	}

	return Row{OccurredOn: date, Amount: amount, RawLabel: label}, true
}

func parseDate(s string) (time.Time, bool) {
	for _, layout := range dateLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// parseAmount : « -1 234,56 € » → centimes, sans float (money.Parse).
func parseAmount(s string) (money.Cents, error) {
	s = strings.NewReplacer(
		"€", "", " ", "", " ", "", " ", "", "+", "",
	).Replace(strings.TrimSpace(s))
	return money.Parse(s)
}

func detectDelimiter(line string) rune {
	if strings.Count(line, ";") >= strings.Count(line, ",") &&
		strings.Count(line, ";") >= strings.Count(line, "\t") {
		return ';'
	}
	if strings.Count(line, "\t") > strings.Count(line, ",") {
		return '\t'
	}
	return ','
}

func looksLikeHeader(line string) bool {
	l := normalizeHeader(line)
	return strings.Contains(l, "date") &&
		(strings.Contains(l, "libell") || strings.Contains(l, "montant") ||
			strings.Contains(l, "debit") || strings.Contains(l, "label") ||
			strings.Contains(l, "amount"))
}

func looksLikeMovement(line string) bool {
	for _, layout := range dateLayouts {
		first := line
		if i := strings.IndexAny(line, ";,\t"); i > 0 {
			first = line[:i]
		}
		if _, err := time.Parse(layout, strings.TrimSpace(first)); err == nil {
			return true
		}
	}
	return false
}

func normalizeHeader(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	replacer := strings.NewReplacer("é", "e", "è", "e", "ê", "e", "à", "a", "û", "u")
	return replacer.Replace(s)
}

// ensureUTF8 convertit un contenu Windows-1252/Latin-1 (exports bancaires
// français fréquents) en UTF-8 si nécessaire.
func ensureUTF8(s string) string {
	if utf8.ValidString(s) {
		return s
	}
	out := make([]rune, 0, len(s))
	for _, b := range []byte(s) {
		out = append(out, rune(b))
	}
	return string(out)
}

func abs(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}
