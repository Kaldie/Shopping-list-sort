// Package stats derives buying habits from completed shopping lists and
// turns them into template suggestions.
package stats

import (
	"math"
	"sort"
	"strconv"
	"time"

	"github.com/kaldie/shopping-list-sort/backend/internal/store"
)

// Thresholds for suggestions. Kept simple and explainable.
const (
	addMinFreq     = 0.5  // bought in ≥50% of lists but not on the template
	addMinCount    = 2    // …and at least twice
	removeMaxFreq  = 0.25 // on the template but bought in ≤25% of lists
	removeMinLists = 4    // …only judged after this many lists
)

type ItemStat struct {
	Name       string    `json:"name"`
	Count      int       `json:"count"`     // lists it was bought in
	Frequency  float64   `json:"frequency"` // Count / lists
	TotalQty   int       `json:"totalQty"`
	AvgQty     float64   `json:"avgQty"` // per list it was bought in
	LastBought time.Time `json:"lastBought"`
	InTemplate bool      `json:"inTemplate"`
}

type Suggestion struct {
	Name   string `json:"name"`
	Qty    int    `json:"qty"`
	Reason string `json:"reason"`
}

type Report struct {
	Lists              int          `json:"lists"`
	Items              []ItemStat   `json:"items"`
	AddToTemplate      []Suggestion `json:"addToTemplate"`
	RemoveFromTemplate []Suggestion `json:"removeFromTemplate"`
	AdjustQty          []Suggestion `json:"adjustQty"`
}

// Compute builds a report from completed lists, newest first, limited to the
// last `window` lists (0 = all). If tmpl is non-nil, only lists made from it
// are counted and template suggestions are produced.
func Compute(lists []store.List, tmpl *store.Template, window int) Report {
	var done []store.List
	for _, l := range lists {
		if l.CompletedAt == nil || (tmpl != nil && l.TemplateID != tmpl.ID) {
			continue
		}
		done = append(done, l)
	}
	sort.Slice(done, func(i, j int) bool { return done[i].CompletedAt.After(*done[j].CompletedAt) })
	if window > 0 && len(done) > window {
		done = done[:window]
	}

	inTmpl := map[string]store.Item{}
	if tmpl != nil {
		for _, it := range tmpl.Items {
			inTmpl[store.Normalize(it.Name)] = it
		}
	}

	byKey := map[string]*ItemStat{}
	for _, l := range done {
		for _, it := range l.Items {
			k := store.Normalize(it.Name)
			s, ok := byKey[k]
			if !ok {
				s = &ItemStat{Name: it.Name}
				byKey[k] = s
			}
			s.Count++
			s.TotalQty += it.Qty
			if l.CompletedAt.After(s.LastBought) {
				s.LastBought = *l.CompletedAt
			}
		}
	}

	r := Report{Lists: len(done), Items: []ItemStat{}, AddToTemplate: []Suggestion{}, RemoveFromTemplate: []Suggestion{}, AdjustQty: []Suggestion{}}
	for k, s := range byKey {
		s.Frequency = float64(s.Count) / float64(len(done))
		s.AvgQty = float64(s.TotalQty) / float64(s.Count)
		_, s.InTemplate = inTmpl[k]
		r.Items = append(r.Items, *s)
	}
	sort.Slice(r.Items, func(i, j int) bool {
		if r.Items[i].Count != r.Items[j].Count {
			return r.Items[i].Count > r.Items[j].Count
		}
		return r.Items[i].Name < r.Items[j].Name
	})

	if tmpl == nil || len(done) == 0 {
		return r
	}
	for _, s := range r.Items {
		k := store.Normalize(s.Name)
		qty := int(math.Round(s.AvgQty))
		if !s.InTemplate && s.Frequency >= addMinFreq && s.Count >= addMinCount {
			r.AddToTemplate = append(r.AddToTemplate, Suggestion{s.Name, qty, pct(s.Count, len(done)) + " but not on the template"})
		}
		if t, ok := inTmpl[k]; ok && qty != t.Qty && s.Count >= addMinCount {
			r.AdjustQty = append(r.AdjustQty, Suggestion{t.Name, qty, "template says " + itoa(t.Qty) + ", you usually buy " + itoa(qty)})
		}
	}
	if len(done) >= removeMinLists {
		for k, t := range inTmpl {
			s := byKey[k]
			count := 0
			if s != nil {
				count = s.Count
			}
			if float64(count)/float64(len(done)) <= removeMaxFreq {
				r.RemoveFromTemplate = append(r.RemoveFromTemplate, Suggestion{t.Name, t.Qty, "on the template but " + pct(count, len(done))})
			}
		}
		sort.Slice(r.RemoveFromTemplate, func(i, j int) bool { return r.RemoveFromTemplate[i].Name < r.RemoveFromTemplate[j].Name })
	}
	return r
}

func pct(n, of int) string { return "bought in " + itoa(n) + " of the last " + itoa(of) + " lists" }

func itoa(n int) string { return strconv.Itoa(n) }
