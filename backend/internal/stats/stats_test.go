package stats

import (
	"testing"

	"github.com/kaldie/shopping-list-sort/backend/internal/store"
)

func TestTemplateSuggestions(t *testing.T) {
	repo, _ := store.Open("")
	tmpl, _ := repo.CreateTemplate("Weekly", []store.Item{{Name: "Milk", Qty: 1}, {Name: "Bread"}, {Name: "Caviar"}})
	weeks := [][]store.Item{
		{{Name: "milk", Qty: 2}, {Name: "bread"}, {Name: "bananas"}},
		{{Name: "milk", Qty: 2}, {Name: "bread"}, {Name: "bananas"}},
		{{Name: "milk", Qty: 3}, {Name: "bread"}, {Name: "beer"}},
		{{Name: "milk", Qty: 2}, {Name: "bread"}, {Name: "bananas"}},
	}
	for _, items := range weeks {
		l, _ := repo.ListFromTemplate(tmpl.ID, "")
		if _, err := repo.CompleteList(l.ID, items); err != nil {
			t.Fatal(err)
		}
	}
	repo.CreateList("open, not completed", []store.Item{{Name: "bananas"}}, tmpl.ID)

	r := Compute(repo.Lists(), &tmpl, 0)
	if r.Lists != 4 {
		t.Fatalf("lists=%d", r.Lists)
	}
	if len(r.AddToTemplate) != 1 || r.AddToTemplate[0].Name != "bananas" {
		t.Errorf("add: %+v", r.AddToTemplate)
	}
	if len(r.RemoveFromTemplate) != 1 || r.RemoveFromTemplate[0].Name != "Caviar" {
		t.Errorf("remove: %+v", r.RemoveFromTemplate)
	}
	if len(r.AdjustQty) != 1 || r.AdjustQty[0].Name != "Milk" || r.AdjustQty[0].Qty != 2 {
		t.Errorf("adjust: %+v", r.AdjustQty)
	}
	if r.Items[0].Count != 4 || r.Items[0].Frequency != 1 {
		t.Errorf("top item: %+v", r.Items[0])
	}
}
