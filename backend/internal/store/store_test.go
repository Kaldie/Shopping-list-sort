package store

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestVotingResistsOneNoisyUser(t *testing.T) {
	r, _ := Open("")
	s, _ := r.Create("AH", "alice", []string{"Produce", "Dairy"})
	for i := 0; i < 20; i++ { // troll records the reverse order many times
		r.RecordTrip(s.ID, "troll", []string{"milk", "apples"})
	}
	r.RecordTrip(s.ID, "alice", []string{"apples", "milk"})
	r.RecordTrip(s.ID, "bob", []string{"apples", "milk"})

	v, _ := r.Get(s.ID, "carol")
	if v.Pairs["apples|milk"] != 2 || v.Pairs["milk|apples"] != 1 {
		t.Fatalf("votes: %v", v.Pairs)
	}
}

func TestForkOnEdit(t *testing.T) {
	r, _ := Open("")
	s, _ := r.Create("AH", "alice", []string{"A", "B"})
	r.Update(s.ID, "bob", "", []string{"B", "A"}, nil)
	r.RecordTrip(s.ID, "alice", []string{"x", "y"})

	bob, _ := r.Get(s.ID, "bob")
	if !bob.Forked || bob.Sections[0] != "B" || len(bob.Pairs) != 0 {
		t.Fatalf("bob fork: %+v", bob)
	}
	alice, _ := r.Get(s.ID, "alice")
	if alice.Forked || alice.Sections[0] != "A" {
		t.Fatalf("public changed: %+v", alice)
	}
	r.Unfork(s.ID, "bob")
	if bob, _ = r.Get(s.ID, "bob"); bob.Forked || bob.Sections[0] != "A" {
		t.Fatalf("unfork: %+v", bob)
	}
	if err := r.Delete(s.ID, "bob"); err != ErrForbidden {
		t.Fatalf("non-creator delete: %v", err)
	}
}

func TestListsArePrivate(t *testing.T) {
	r, _ := Open("")
	l, _ := r.CreateList("alice", "wk", []Item{{Name: "milk"}}, "")
	if _, err := r.GetList(l.ID, "bob"); err != ErrNotFound {
		t.Fatalf("bob read alice's list: %v", err)
	}
	if len(r.Lists("bob")) != 0 {
		t.Fatal("bob sees alice's lists")
	}
}

func TestItemSectionOverridesArePersonal(t *testing.T) {
	r, _ := Open("")
	s, _ := r.Create("Shop", "owner", []string{"Fruit", "Dairy"})
	r.Update(s.ID, "owner", "", nil, map[string]string{"milk": "Dairy"})
	got, _ := r.Update(s.ID, "eve", "", nil, map[string]string{"milk": "Fruit", "apple": "Fruit"})
	if got.ItemSection["milk"] != "Fruit" || got.ItemSection["apple"] != "Fruit" {
		t.Fatalf("eve's view: %v", got.ItemSection)
	}
	own, _ := r.Get(s.ID, "owner")
	if own.ItemSection["milk"] != "Dairy" || own.ItemSection["apple"] != "" {
		t.Fatalf("shared map changed by non-creator: %v", own.ItemSection)
	}
}

func TestCleanItems(t *testing.T) {
	got := CleanItems([]Item{
		{Name: "* Orzo"}, {Name: "- Orzo"}, {Name: "- 3 Paprika's"}, {Name: "100 g Feta"},
		{Name: "Kaas ?"}, {Name: "2x Groente", Qty: 2}, {Name: "***"}, {Name: "Pinda's"},
	})
	want := []Item{{"Orzo", 2}, {"Paprikas", 3}, {"100 g Feta", 1}, {"Kaas", 1}, {"Groente", 4}, {"Pindas", 1}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestOpenCleansOldItems(t *testing.T) {
	file := filepath.Join(t.TempDir(), "data.json")
	os.WriteFile(file, []byte(`{"lists":{"l1":{"id":"l1","ownerId":"a","name":"x","items":[{"name":"* Orzo","qty":1},{"name":"- Orzo","qty":1},{"name":"- Kaas ?","qty":1}]}}}`), 0o600)
	r, err := Open(file)
	if err != nil {
		t.Fatal(err)
	}
	l, _ := r.GetList("l1", "a")
	if want := []Item{{"Orzo", 2}, {"Kaas", 1}}; !reflect.DeepEqual(l.Items, want) {
		t.Fatalf("got %v want %v", l.Items, want)
	}
}
