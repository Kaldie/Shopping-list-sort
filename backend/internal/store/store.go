// Package store holds the domain model and a JSON-file backed repository.
package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

var ErrNotFound = errors.New("not found")

type Store struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Sections []string `json:"sections"` // explicit order (photo/manual), may be empty
	// ItemSection maps a normalized item name to a section name.
	ItemSection map[string]string `json:"itemSection"`
	// Pairs counts trips where item a was picked before item b, keyed "a|b".
	Pairs map[string]int `json:"pairs"`
	Trips int            `json:"trips"`
}

type Trip struct {
	StoreID string    `json:"storeId"`
	Items   []string  `json:"items"`
	At      time.Time `json:"at"`
}

type data struct {
	Stores    map[string]*Store    `json:"stores"`
	Trips     []Trip               `json:"trips"`
	Lists     map[string]*List     `json:"lists"`
	Templates map[string]*Template `json:"templates"`
	NextID    int                  `json:"nextId"`
}

// Repository is a goroutine-safe JSON file store.
type Repository struct {
	mu   sync.Mutex
	path string
	d    data
}

func Open(path string) (*Repository, error) {
	r := &Repository{path: path}
	if path != "" {
		b, err := os.ReadFile(path)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		if err == nil {
			if err := json.Unmarshal(b, &r.d); err != nil {
				return nil, fmt.Errorf("parse %s: %w", path, err)
			}
		}
	}
	if r.d.Stores == nil {
		r.d.Stores = map[string]*Store{}
	}
	if r.d.Lists == nil {
		r.d.Lists = map[string]*List{}
	}
	if r.d.Templates == nil {
		r.d.Templates = map[string]*Template{}
	}
	return r, nil
}

func (r *Repository) save() error {
	if r.path == "" {
		return nil
	}
	b, err := json.MarshalIndent(r.d, "", "  ")
	if err != nil {
		return err
	}
	tmp := r.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, r.path)
}

func Normalize(s string) string { return strings.ToLower(strings.TrimSpace(s)) }

func clone(s *Store) Store {
	c := *s
	c.Sections = append([]string(nil), s.Sections...)
	c.ItemSection = make(map[string]string, len(s.ItemSection))
	for k, v := range s.ItemSection {
		c.ItemSection[k] = v
	}
	c.Pairs = make(map[string]int, len(s.Pairs))
	for k, v := range s.Pairs {
		c.Pairs[k] = v
	}
	return c
}

func (r *Repository) List() []Store {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]Store, 0, len(r.d.Stores))
	for _, s := range r.d.Stores {
		out = append(out, clone(s))
	}
	return out
}

func (r *Repository) Get(id string) (Store, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.d.Stores[id]
	if !ok {
		return Store{}, ErrNotFound
	}
	return clone(s), nil
}

func (r *Repository) Create(name string, sections []string) (Store, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s := &Store{
		ID:          r.newID("s"),
		Name:        name,
		Sections:    sections,
		ItemSection: map[string]string{},
		Pairs:       map[string]int{},
	}
	r.d.Stores[s.ID] = s
	return clone(s), r.save()
}

// Update replaces name, section order and item→section assignments.
func (r *Repository) Update(id, name string, sections []string, itemSection map[string]string) (Store, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.d.Stores[id]
	if !ok {
		return Store{}, ErrNotFound
	}
	if name != "" {
		s.Name = name
	}
	if sections != nil {
		s.Sections = sections
	}
	for k, v := range itemSection {
		if v == "" {
			delete(s.ItemSection, Normalize(k))
		} else {
			s.ItemSection[Normalize(k)] = v
		}
	}
	return clone(s), r.save()
}

func (r *Repository) Delete(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.d.Stores[id]; !ok {
		return ErrNotFound
	}
	delete(r.d.Stores, id)
	return r.save()
}

// RecordTrip stores items in the order they were picked and updates the
// pairwise precedence counts.
func (r *Repository) RecordTrip(id string, items []string) (Store, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.d.Stores[id]
	if !ok {
		return Store{}, ErrNotFound
	}
	seen := map[string]bool{}
	var norm []string
	for _, it := range items {
		n := Normalize(it)
		if n != "" && !seen[n] {
			seen[n] = true
			norm = append(norm, n)
		}
	}
	for i := range norm {
		for j := i + 1; j < len(norm); j++ {
			s.Pairs[PairKey(norm[i], norm[j])]++
		}
	}
	s.Trips++
	r.d.Trips = append(r.d.Trips, Trip{StoreID: id, Items: norm, At: time.Now().UTC()})
	return clone(s), r.save()
}

func PairKey(a, b string) string { return a + "|" + b }
