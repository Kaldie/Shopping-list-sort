// Package receipt turns raw receipt text (OCR output or pasted) into the
// ordered list of purchased items. Self-scanner receipts list items in the
// order they were scanned, i.e. the order they were picked in the store.
package receipt

import (
	"regexp"
	"strings"
)

var (
	price    = regexp.MustCompile(`\s*-?€?\s*\d+[.,]\d{2}\s*[A-Za-z]?\s*$`)
	qtyLead  = regexp.MustCompile(`^\s*\d+\s*[xX*]?\s+`)
	weight   = regexp.MustCompile(`^\s*[\d.,]+\s*(kg|g|st|stk)\b`)
	letters  = regexp.MustCompile(`\p{L}{2,}`)
	skipWord = []string{
		"totaal", "total", "subtotaal", "subtotal", "btw", "vat", "tax", "pin", "betaal", "betaling",
		"contant", "cash", "wisselgeld", "change", "korting", "discount", "bonus", "spaar", "zegel",
		"kassa", "bon", "receipt", "datum", "date", "tijd", "time", "bedankt", "thank", "terminal",
		"transactie", "transaction", "klant", "customer", "artikelen", "items", "statiegeld", "deposit",
		"maestro", "visa", "mastercard", "card",
	}
)

// Parse returns item names in receipt order, without prices, quantities,
// totals, payment and header/footer lines. Duplicates keep the first spot.
func Parse(text string) []string {
	lines := strings.Split(text, "\n")
	// Item lines carry a price; headers (store name, address) don't. Plain
	// pasted lists without any prices keep every line.
	needPrice := false
	for _, l := range lines {
		if price.MatchString(l) {
			needPrice = true
			break
		}
	}
	var out []string
	seen := map[string]bool{}
	for _, line := range lines {
		l := strings.TrimSpace(line)
		if l == "" || weight.MatchString(l) || isSkip(l) || (needPrice && !price.MatchString(l)) {
			continue
		}
		l = price.ReplaceAllString(l, "")
		l = qtyLead.ReplaceAllString(l, "")
		l = strings.Trim(l, " .:-*")
		if !letters.MatchString(l) {
			continue
		}
		key := strings.ToLower(l)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, l)
	}
	return out
}

func isSkip(l string) bool {
	low := strings.ToLower(l)
	for _, w := range strings.FieldsFunc(low, func(r rune) bool { return r == ' ' || r == ':' || r == '/' }) {
		for _, s := range skipWord {
			if w == s {
				return true
			}
		}
	}
	return false
}
