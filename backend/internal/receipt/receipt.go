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
	// Digital receipts (Jumbo) put the price on its own line below the name.
	priceOnly = regexp.MustCompile(`^-?€?\s*\d+[.,]\d{2}$`)
	// "2 X 0,84" between a name and its line total.
	qtyPrice = regexp.MustCompile(`^\d+\s*[xX*]\s*€?\s*\d+[.,]\d{2}$`)
	percent  = regexp.MustCompile(`\b\d+\s*%\s*`)
	// "4st", "1kc", "500g" pack sizes at the end of a name.
	packSize = regexp.MustCompile(`(?i)\s+\d+[.,]?\d*\s*(st|stk|kc|kg|g|ml|cl|l)$`)
	brand    = regexp.MustCompile(`(?i)^(jumbo|ah|plus|lidl|aldi)\s+`)
	skipWord = []string{
		"totaal", "total", "subtotaal", "subtotal", "btw", "vat", "tax", "pin", "betaal", "betaling",
		"contant", "cash", "wisselgeld", "change", "korting", "discount", "bonus", "spaar", "zegel",
		"kassa", "bon", "receipt", "datum", "date", "tijd", "time", "bedankt", "thank", "terminal",
		"transactie", "transaction", "klant", "customer", "artikelen", "items", "statiegeld", "deposit",
		"maestro", "visa", "mastercard", "card", "actie",
	}
)

// Parse returns item names in receipt order, without prices, quantities,
// totals, payment and header/footer lines. Duplicates keep the first spot.
func Parse(text string) []string {
	lines := joinPriceLines(strings.Split(text, "\n"))
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
		l = percent.ReplaceAllString(l, "")
		l = strings.Trim(l, " .:-*")
		l = packSize.ReplaceAllString(l, "")
		if b := brand.ReplaceAllString(l, ""); letters.MatchString(b) {
			l = b
		}
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

// joinPriceLines glues a price-only line onto the item name above it and drops
// "2 X 0,84" quantity lines, so name/price pairs look like single-line items.
func joinPriceLines(lines []string) []string {
	var out []string
	for _, line := range lines {
		l := strings.TrimSpace(line)
		switch {
		case qtyPrice.MatchString(l):
		case priceOnly.MatchString(l) && len(out) > 0 && out[len(out)-1] != "" && !price.MatchString(out[len(out)-1]):
			out[len(out)-1] += " " + l
		default:
			out = append(out, l)
		}
	}
	return out
}
