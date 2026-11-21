package layout

import "strings"

// keywords is the fallback dictionary used to place items that have never been
// seen on a trip. Dutch first, some English; extend freely.
var keywords = map[string][]string{
	"Groente & fruit": {"apple", "appel", "banana", "banaan", "banan", "tomato", "tomaat", "tomat", "lettuce", "sla", "onion", "ui", "potato", "aardappel", "carrot", "wortel", "cucumber", "komkommer", "pepper", "paprika", "fruit", "groente", "lemon", "citroen", "garlic", "knoflook", "spinach", "spinazie", "avocado", "grape", "druif", "berry", "bes", "courgette", "champignon", "broccoli", "bloemkool", "prei", "peer", "sinaasappel", "mandarijn", "aubergine", "rucola", "peen"},
	"Brood & gebak":   {"bread", "brood", "baguette", "croissant", "roll", "bolletje", "broodje", "bagel", "cake", "koek", "ontbijtkoek", "krentenbol", "bollen", "afbak", "wrap", "tosti"},
	"Vlees & vis":     {"chicken", "kip", "beef", "rund", "pork", "varken", "gehakt", "mince", "sausage", "worst", "bacon", "spek", "fish", "vis", "salmon", "zalm", "ham", "vlees", "biefstuk", "schnitzel", "filet", "tonijn"},
	"Zuivel":          {"milk", "melk", "cheese", "kaas", "yogurt", "yoghurt", "butter", "boter", "cream", "room", "egg", "ei", "eieren", "kwark", "vla", "feta", "mozzarella", "margarine", "karnemelk", "slagroom", "skyr"},
	"Houdbaar":        {"pasta", "rice", "rijst", "flour", "bloem", "sugar", "suiker", "oil", "olie", "sauce", "saus", "soup", "soep", "beans", "bonen", "cereal", "muesli", "peanut", "pinda", "pindakaas", "jam", "hagelslag", "spice", "kruiden", "salt", "zout", "coffee", "koffie", "tea", "thee", "orzo", "couscous", "cracker", "beschuit", "appelmoes", "moes", "chocopasta", "noedels", "wraps", "mayonaise", "ketchup", "tonijnblik", "mais", "linzen", "kikkererwten"},
	"Diepvries":       {"frozen", "diepvries", "ice cream", "ijs", "pizza", "fries", "friet"},
	"Dranken":         {"water", "juice", "sap", "soda", "cola", "beer", "bier", "wine", "wijn", "lemonade", "limonade", "tintelfris", "frisdrank", "sinaasappelsap", "appelsap", "ranja", "siroop"},
	"Snacks":          {"chips", "crisps", "chocolate", "chocola", "cookie", "candy", "snoep", "nuts", "noten", "pinda's", "koekjes", "drop", "popcorn", "borrelnoten"},
	"Huishouden":      {"soap", "zeep", "detergent", "wasmiddel", "toilet", "paper", "papier", "shampoo", "toothpaste", "tandpasta", "sponge", "spons", "bag", "zak", "vuilniszak", "keukenrol", "afwasmiddel"},
}

// english maps a section name to the English one older stores may use, so a
// store built with "Dairy" keeps matching after the dictionary went Dutch.
var english = map[string]string{
	"Groente & fruit": "Produce",
	"Brood & gebak":   "Bakery",
	"Vlees & vis":     "Meat & Fish",
	"Zuivel":          "Dairy",
	"Houdbaar":        "Pantry",
	"Diepvries":       "Frozen",
	"Dranken":         "Drinks",
	"Snacks":          "Snacks",
	"Huishouden":      "Household",
}

// Guess returns a section name for an item based on keywords, or "".
// Whole-word matches win over word endings, which win over substrings; Dutch
// compounds put the meaning last ("sinaasappelsap" is a drink, not a fruit).
func Guess(item string) string {
	item = strings.ToLower(strings.ReplaceAll(item, "'", ""))
	words := strings.Fields(item)
	best, bestScore := "", 0
	for sec, kws := range keywords {
		for _, kw := range kws {
			kw = strings.ReplaceAll(kw, "'", "")
			score := 0
			for _, w := range words {
				switch {
				case w == kw || w == kw+"s" || w == kw+"en" || w == kw+"je" || w == kw+"jes":
					score = max(score, 1000+len(kw))
				case len(kw) >= 3 && (strings.HasSuffix(w, kw) || strings.HasSuffix(strings.TrimSuffix(strings.TrimSuffix(w, "en"), "s"), kw)):
					score = max(score, 500+len(kw))
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
