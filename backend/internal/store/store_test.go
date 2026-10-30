package store

import "testing"

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
