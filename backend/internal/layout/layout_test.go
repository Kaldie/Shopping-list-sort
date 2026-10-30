package layout

import (
	"reflect"
	"testing"

	"github.com/kaldie/shopping-list-sort/backend/internal/store"
)

func newStore(t *testing.T, sections []string, trips ...[]string) store.Store {
	t.Helper()
	r, _ := store.Open("")
	s, _ := r.Create("test", sections)
	for _, tr := range trips {
		var err error
		if s, err = r.RecordTrip(s.ID, tr); err != nil {
			t.Fatal(err)
		}
	}
	return s
}

func TestRankItemsMajorityWins(t *testing.T) {
	s := newStore(t, nil,
		[]string{"bread", "milk", "beer"},
		[]string{"bread", "milk", "beer"},
		[]string{"milk", "bread", "beer"}, // noisy trip
	)
	got := RankItems([]string{"beer", "milk", "bread"}, s.Pairs)
	want := []string{"bread", "milk", "beer"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestSortUsesLearnedThenSectionThenUnknown(t *testing.T) {
	s := newStore(t, []string{"Produce", "Bakery", "Dairy", "Drinks"},
		[]string{"apples", "bread", "milk"},
	)
	groups := Sort(s, []string{"Zorblax", "cola", "Milk", "cheese", "Apples"})
	var order []string
	for _, g := range groups {
		order = append(order, g.Items...)
	}
	want := []string{"Apples", "Milk", "cheese", "cola", "Zorblax"}
	if !reflect.DeepEqual(order, want) {
		t.Fatalf("got %v want %v", order, want)
	}
	if last := groups[len(groups)-1]; last.Section != unsorted {
		t.Fatalf("unknown item should be unsorted, got %q", last.Section)
	}
}

func TestGenerateLearnsSectionOrderWithoutExplicitLayout(t *testing.T) {
	s := newStore(t, nil, []string{"beer", "cheese", "bread"})
	l := Generate(s)
	var names []string
	for _, sp := range l.Sections {
		names = append(names, sp.Name)
	}
	want := []string{"Drinks", "Dairy", "Bakery"}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("got %v want %v", names, want)
	}
}

func TestGuess(t *testing.T) {
	cases := map[string]string{"halfvolle melk": "Dairy", "bananas": "Produce", "toilet paper": "Household", "xyz": ""}
	for in, want := range cases {
		if got := Guess(in); got != want {
			t.Errorf("Guess(%q)=%q want %q", in, got, want)
		}
	}
}
