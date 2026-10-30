package store

import (
	"fmt"
	"strings"
	"time"
)

type Item struct {
	Name string `json:"name"`
	Qty  int    `json:"qty"`
}

// List is one week's shopping list. Completed lists are the history that
// statistics are computed from.
type List struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Items       []Item     `json:"items"`
	TemplateID  string     `json:"templateId,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
}

// Template is a reusable base list ("weekly groceries").
type Template struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Items []Item `json:"items"`
}

// CleanItems trims names, defaults qty to 1 and merges duplicates.
func CleanItems(in []Item) []Item {
	out := []Item{}
	idx := map[string]int{}
	for _, it := range in {
		it.Name = strings.TrimSpace(it.Name)
		if it.Name == "" {
			continue
		}
		if it.Qty < 1 {
			it.Qty = 1
		}
		if i, ok := idx[Normalize(it.Name)]; ok {
			out[i].Qty += it.Qty
			continue
		}
		idx[Normalize(it.Name)] = len(out)
		out = append(out, it)
	}
	return out
}

func (r *Repository) newID(prefix string) string {
	r.d.NextID++
	return fmt.Sprintf("%s%d", prefix, r.d.NextID)
}

func cloneList(l *List) List {
	c := *l
	c.Items = append([]Item{}, l.Items...)
	return c
}

func cloneTemplate(t *Template) Template {
	c := *t
	c.Items = append([]Item{}, t.Items...)
	return c
}

// --- Lists ---

func (r *Repository) Lists() []List {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]List, 0, len(r.d.Lists))
	for _, l := range r.d.Lists {
		out = append(out, cloneList(l))
	}
	return out
}

func (r *Repository) GetList(id string) (List, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	l, ok := r.d.Lists[id]
	if !ok {
		return List{}, ErrNotFound
	}
	return cloneList(l), nil
}

func (r *Repository) CreateList(name string, items []Item, templateID string) (List, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.createListLocked(name, items, templateID)
}

func (r *Repository) createListLocked(name string, items []Item, templateID string) (List, error) {
	if name == "" {
		name = "List " + time.Now().Format("2 Jan")
	}
	l := &List{ID: r.newID("l"), Name: name, Items: CleanItems(items), TemplateID: templateID, CreatedAt: time.Now().UTC()}
	r.d.Lists[l.ID] = l
	return cloneList(l), r.save()
}

func (r *Repository) UpdateList(id, name string, items []Item) (List, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	l, ok := r.d.Lists[id]
	if !ok {
		return List{}, ErrNotFound
	}
	if name != "" {
		l.Name = name
	}
	if items != nil {
		l.Items = CleanItems(items)
	}
	return cloneList(l), r.save()
}

func (r *Repository) DeleteList(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.d.Lists[id]; !ok {
		return ErrNotFound
	}
	delete(r.d.Lists, id)
	return r.save()
}

// CompleteList marks a list as bought. If bought is non-nil it replaces the
// items with what was actually bought, so statistics reflect reality.
func (r *Repository) CompleteList(id string, bought []Item) (List, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	l, ok := r.d.Lists[id]
	if !ok {
		return List{}, ErrNotFound
	}
	if bought != nil {
		l.Items = CleanItems(bought)
	}
	now := time.Now().UTC()
	l.CompletedAt = &now
	return cloneList(l), r.save()
}

// --- Templates ---

func (r *Repository) Templates() []Template {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]Template, 0, len(r.d.Templates))
	for _, t := range r.d.Templates {
		out = append(out, cloneTemplate(t))
	}
	return out
}

func (r *Repository) GetTemplate(id string) (Template, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.d.Templates[id]
	if !ok {
		return Template{}, ErrNotFound
	}
	return cloneTemplate(t), nil
}

func (r *Repository) CreateTemplate(name string, items []Item) (Template, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	t := &Template{ID: r.newID("t"), Name: name, Items: CleanItems(items)}
	r.d.Templates[t.ID] = t
	return cloneTemplate(t), r.save()
}

func (r *Repository) UpdateTemplate(id, name string, items []Item) (Template, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.d.Templates[id]
	if !ok {
		return Template{}, ErrNotFound
	}
	if name != "" {
		t.Name = name
	}
	if items != nil {
		t.Items = CleanItems(items)
	}
	return cloneTemplate(t), r.save()
}

func (r *Repository) DeleteTemplate(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.d.Templates[id]; !ok {
		return ErrNotFound
	}
	delete(r.d.Templates, id)
	return r.save()
}

// ListFromTemplate starts a new list as a copy of the template.
func (r *Repository) ListFromTemplate(templateID, name string) (List, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.d.Templates[templateID]
	if !ok {
		return List{}, ErrNotFound
	}
	if name == "" {
		name = t.Name + " · " + time.Now().Format("2 Jan")
	}
	return r.createListLocked(name, t.Items, t.ID)
}
