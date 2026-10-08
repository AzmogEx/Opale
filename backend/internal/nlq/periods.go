package nlq

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

type period struct {
	from, to time.Time
	label    string
}
type monthMention struct {
	name     string
	position int
}
type yearMention struct{ year, position int }

// Splitting letters and numbers also handles compact input such as mars2024
// or en2024, while keeping accented month names intact.
var periodTokens = regexp.MustCompile(`[\p{L}]+|[0-9]+`)

func explicitPeriods(text string, now time.Time) ([]period, bool) {
	named := []monthMention{}
	years := []yearMention{}
	for i, word := range periodTokens.FindAllString(text, -1) {
		if _, ok := months[word]; ok {
			named = append(named, monthMention{word, i})
			continue
		}
		if len(word) == 4 {
			if y, e := strconv.Atoi(word); e == nil && y >= 1900 && y <= 9999 {
				years = append(years, yearMention{y, i})
			}
		}
	}
	if len(named) > 2 || len(years) > 2 {
		return nil, false
	}
	if len(named)+len(years) == 0 {
		return nil, true
	}
	// Mixing a named period with a relative period is not resolved by this
	// grammar. Never silently discard the second part of the user's request.
	if strings.Contains(text, "ce mois") || strings.Contains(text, "mois dernier") || strings.Contains(text, "cette année") || strings.Contains(text, "cette annee") {
		return nil, false
	}
	yearPeriod := func(y int) period {
		return period{time.Date(y, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(y, 12, 31, 0, 0, 0, 0, time.UTC), "en " + strconv.Itoa(y)}
	}
	monthPeriod := func(name string, y int) period {
		from := time.Date(y, months[name], 1, 0, 0, 0, 0, time.UTC)
		return period{from, from.AddDate(0, 1, -1), "en " + name + " " + strconv.Itoa(y)}
	}
	out := []period{}
	if len(named) == 0 {
		for _, y := range years {
			out = append(out, yearPeriod(y.year))
		}
		return out, true
	}
	if len(named) == 1 && len(years) == 2 {
		// "mars 2024 et 2025" repeats the named month, with distinct years.
		for _, y := range years {
			out = append(out, monthPeriod(named[0].name, y.year))
		}
		return out, true
	}
	if len(named) == 2 && len(years) == 2 {
		// Two years listed entirely before/after both months are ambiguous (for
		// example "mars et avril 2024 et 2025" could mean four periods).
		if years[0].position > named[1].position || years[1].position < named[0].position {
			return nil, false
		}
	}
	for i, m := range named {
		switch len(years) {
		case 0:
			f, t, label := monthRange(m.name, now)
			out = append(out, period{f, t, label})
		case 1:
			// A single explicit year is shared: "mars et avril 2024".
			out = append(out, monthPeriod(m.name, years[0].year))
		case 2:
			out = append(out, monthPeriod(m.name, years[i].year))
		}
	}
	return out, true
}
