// Package layout turns observed shopping trips and explicit section orders
// into a supermarket layout, and sorts shopping lists against it.
package layout

import (
	"sort"
	"strings"

	"github.com/kaldie/shopping-list-sort/backend/internal/store"
)

// Layout is the generated view of a store.
type Layout struct {
	Sections []SectionPos `json:"sections"`
	Items    []string     `json:"items"` // learned item walking order
}

type SectionPos struct {
	Name     string   `json:"name"`
	Position float64  `json:"position"` // 0 = entrance, 1 = checkout
	Items    []string `json:"items"`
	Source   string   `json:"source"` // "explicit", "learned" or "both"
}

type Group struct {
	Section string   `json:"section"`
	Items   []string `json:"items"`
}

const unsorted = "Unsorted"

// RankItems orders items by a Copeland score over observed pairwise
// precedences: each pair contributes the net fraction of trips in which one
// item was picked before the other. Robust to the odd out-of-order trip.
func RankItems(items []string, pairs map[string]int) []string {
	score := make(map[string]float64, len(items))
	for i, a := range items {
		for _, b := range items[i+1:] {
			ab, ba := pairs[store.PairKey(a, b)], pairs[store.PairKey(b, a)]
			if ab+ba == 0 {
				continue
			}
			d := float64(ab-ba) / float64(ab+ba)
			score[a] += d
			score[b] -= d
		}
	}
	out := append([]string(nil), items...)
	sort.SliceStable(out, func(i, j int) bool {
		if score[out[i]] != score[out[j]] {
			return score[out[i]] > score[out[j]]
		}
		return out[i] < out[j]
	})
	return out
}

// KnownItems returns every item that appears in at least one trip.
func KnownItems(s store.Store) []string {
	set := map[string]bool{}
	for k := range s.Pairs {
		a, b, _ := strings.Cut(k, "|")
		set[a], set[b] = true, true
	}
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// SectionOf returns the section for an item: learned assignment first, then
// the keyword dictionary, else "".
func SectionOf(s store.Store, item string) string {
	n := store.Normalize(item)
	if sec, ok := s.ItemSection[n]; ok {
		return sec
	}
	return Guess(n)
}

// Generate builds the layout: learned item order, and section positions that
// blend the explicit order (prior) with the mean position of learned items.
func Generate(s store.Store) Layout {
	items := RankItems(KnownItems(s), s.Pairs)
	itemPos := positions(items)

	type acc struct {
		sum      float64
		n        int
		explicit float64
		hasExp   bool
		items    []string
	}
	secs := map[string]*acc{}
	get := func(name string) *acc {
		key := strings.ToLower(name)
		if a, ok := secs[key]; ok {
			return a
		}
		a := &acc{}
		secs[key] = a
		return a
	}
	names := map[string]string{}
	for i, name := range s.Sections {
		a := get(name)
		names[strings.ToLower(name)] = name
		a.hasExp = true
		a.explicit = norm(i, len(s.Sections))
	}
	for _, it := range items {
		sec := SectionOf(s, it)
		if sec == "" {
			continue
		}
		if _, ok := names[strings.ToLower(sec)]; !ok {
			names[strings.ToLower(sec)] = sec
		}
		a := get(sec)
		a.sum += itemPos[it]
		a.n++
		a.items = append(a.items, it)
	}

	l := Layout{Items: items}
	for key, a := range secs {
		sp := SectionPos{Name: names[key], Items: a.items}
		switch {
		case a.hasExp && a.n > 0:
			sp.Position, sp.Source = (a.explicit+a.sum/float64(a.n))/2, "both"
		case a.hasExp:
			sp.Position, sp.Source = a.explicit, "explicit"
		default:
			sp.Position, sp.Source = a.sum/float64(a.n), "learned"
		}
		if sp.Items == nil {
			sp.Items = []string{}
		}
		l.Sections = append(l.Sections, sp)
	}
	sort.SliceStable(l.Sections, func(i, j int) bool {
		if l.Sections[i].Position != l.Sections[j].Position {
			return l.Sections[i].Position < l.Sections[j].Position
		}
		return l.Sections[i].Name < l.Sections[j].Name
	})
	if l.Sections == nil {
		l.Sections = []SectionPos{}
	}
	return l
}

// Sort orders a shopping list for the store and groups it by section.
// Items seen on trips use their learned position; otherwise the section's
// position; unknown items go last under "Unsorted".
func Sort(s store.Store, list []string) []Group {
	l := Generate(s)
	itemPos := positions(l.Items)
	secPos := map[string]float64{}
	for _, sp := range l.Sections {
		secPos[strings.ToLower(sp.Name)] = sp.Position
	}

	type entry struct {
		label, key, section string
		pos                 float64
	}
	var entries []entry
	seen := map[string]bool{}
	for _, raw := range list {
		label := strings.TrimSpace(raw)
		key := store.Normalize(label)
		if key == "" || seen[key] {
			continue
		}
		seen[key] = true
		e := entry{label: label, key: key, section: SectionOf(s, key), pos: 2}
		ip, learned := itemPos[key]
		if sp, ok := secPos[strings.ToLower(e.section)]; ok {
			// Section decides the coarse order; learned order breaks ties inside it.
			e.pos = sp
			if learned {
				e.pos += ip * 1e-3
			} else {
				e.pos += 2e-3
			}
		} else if learned {
			e.pos = ip
		}
		if e.section == "" {
			e.section = unsorted
		}
		entries = append(entries, e)
	}
	sort.SliceStable(entries, func(i, j int) bool { return entries[i].pos < entries[j].pos })

	groups := []Group{}
	for _, e := range entries {
		if n := len(groups); n > 0 && strings.EqualFold(groups[n-1].Section, e.section) {
			groups[n-1].Items = append(groups[n-1].Items, e.label)
			continue
		}
		groups = append(groups, Group{Section: e.section, Items: []string{e.label}})
	}
	return groups
}

func positions(items []string) map[string]float64 {
	m := make(map[string]float64, len(items))
	for i, it := range items {
		m[it] = norm(i, len(items))
	}
	return m
}

func norm(i, n int) float64 {
	if n <= 1 {
		return 0.5
	}
	return float64(i) / float64(n-1)
}
