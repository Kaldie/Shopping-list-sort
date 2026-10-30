package layout

import "strings"

// keywords is the fallback dictionary used to place items that have never been
// seen on a trip. English and Dutch keywords; extend freely.
var keywords = map[string][]string{
	"Produce":     {"apple", "appel", "banana", "banaan", "tomato", "tomaat", "lettuce", "sla", "onion", "ui", "potato", "aardappel", "carrot", "wortel", "cucumber", "komkommer", "pepper", "paprika", "fruit", "groente", "lemon", "citroen", "garlic", "knoflook", "spinach", "spinazie", "avocado", "grape", "druif", "berry", "bes"},
	"Bakery":      {"bread", "brood", "baguette", "croissant", "roll", "bolletje", "bagel", "cake", "koek"},
	"Meat & Fish": {"chicken", "kip", "beef", "rund", "pork", "varken", "gehakt", "mince", "sausage", "worst", "bacon", "spek", "fish", "vis", "salmon", "zalm", "ham"},
	"Dairy":       {"milk", "melk", "cheese", "kaas", "yogurt", "yoghurt", "butter", "boter", "cream", "room", "egg", "ei", "eieren", "kwark", "vla"},
	"Pantry":      {"pasta", "rice", "rijst", "flour", "bloem", "sugar", "suiker", "oil", "olie", "sauce", "saus", "soup", "soep", "beans", "bonen", "cereal", "muesli", "peanut", "pindakaas", "jam", "hagelslag", "spice", "kruiden", "salt", "zout", "coffee", "koffie", "tea", "thee"},
	"Frozen":      {"frozen", "diepvries", "ice cream", "ijs", "pizza", "fries", "friet"},
	"Drinks":      {"water", "juice", "sap", "soda", "cola", "beer", "bier", "wine", "wijn", "lemonade", "limonade"},
	"Snacks":      {"chips", "crisps", "chocolate", "chocola", "cookie", "candy", "snoep", "nuts", "noten"},
	"Household":   {"soap", "zeep", "detergent", "wasmiddel", "toilet", "paper", "papier", "shampoo", "toothpaste", "tandpasta", "sponge", "spons", "bag", "zak"},
}

// Guess returns a section name for an item based on keywords, or "".
// Whole-word matches win over substring matches; longer keywords win ties.
func Guess(item string) string {
	item = strings.ToLower(item)
	words := strings.Fields(item)
	best, bestScore := "", 0
	for sec, kws := range keywords {
		for _, kw := range kws {
			score := 0
			for _, w := range words {
				if w == kw || w == kw+"s" || w == kw+"en" {
					score = 1000 + len(kw)
				}
			}
			if score == 0 && len(kw) >= 4 && strings.Contains(item, kw) {
				score = len(kw)
			}
			if score > bestScore || (score == bestScore && score > 0 && sec < best) {
				best, bestScore = sec, score
			}
		}
	}
	return best
}
