package receipt

import (
	"strings"
	"unicode"
)

// Match is one receipt line mapped to an item on the user's list.
type Match struct {
	Line    string `json:"line"`
	Item    string `json:"item"`    // list item name, or the line itself when unmatched
	Matched bool   `json:"matched"` // false: not on the list
}

// MatchLines maps receipt lines to list items by keyword: an item matches a
// line when one of its words appears as a whole word in the line
// ("melk" ↔ "AH HALFV MELK 1L"). Whole words keep "melk" out of
// "MELKCHOCOLADE" and "ei" out of "PREI". Most matched characters wins.
func MatchLines(lines, items []string) []Match {
	out := make([]Match, 0, len(lines))
	for _, line := range lines {
		lw := wordSet(line)
		best, bestScore := "", 0
		for _, it := range items {
			score := 0
			for _, w := range words(it) {
				if lw[w] || lw[w+"s"] || lw[w+"en"] {
					score += len(w)
				}
			}
			if score > bestScore || (score == bestScore && score > 0 && len(it) < len(best)) {
				best, bestScore = it, score
			}
		}
		if bestScore > 0 {
			out = append(out, Match{Line: line, Item: best, Matched: true})
		} else {
			out = append(out, Match{Line: line, Item: line})
		}
	}
	return out
}

func words(s string) []string {
	return strings.FieldsFunc(strings.ToLower(s), func(r rune) bool { return !unicode.IsLetter(r) })
}

func wordSet(s string) map[string]bool {
	m := map[string]bool{}
	for _, w := range words(s) {
		m[w] = true
	}
	return m
}
