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

var (
	ErrNotFound  = errors.New("not found")
	ErrForbidden = errors.New("forbidden")
)

// Store is a supermarket. Its layout is public: everyone's trips vote on the
// item order. A user whose route differs can fork the section order; their
// view then uses only their own trips.
type Store struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	CreatedBy string   `json:"createdBy"`
	Sections  []string `json:"sections"` // public explicit order, may be empty
	// ItemSection maps a normalized item name to a section name.
	ItemSection map[string]string `json:"itemSection"`
	// UserPairs[user]["a|b"] counts that user's trips where a was picked before b.
	UserPairs map[string]map[string]int `json:"userPairs"`
	// Forks[user] is a user's personal section order.
	Forks map[string][]string `json:"forks"`
	Trips int                 `json:"trips"`

	// Pairs is the effective precedence data for the viewing user. It is
	// computed by View and never stored (older files used it for raw counts).
	Pairs  map[string]int `json:"pairs,omitempty"`
	Forked bool           `json:"forked"`
}

type Trip struct {
	StoreID string    `json:"storeId"`
	UserID  string    `json:"userId"`
	Items   []string  `json:"items"`
	At      time.Time `json:"at"`
}

type data struct {
	Stores    map[string]*Store    `json:"stores"`
	Trips     []Trip               `json:"trips"`
	Lists     map[string]*List     `json:"lists"`
	Templates map[string]*Template `json:"templates"`
	Users     map[string]*User     `json:"users"`
	Sessions  map[string]*Session  `json:"sessions"`
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
	initMaps(&r.d)
	for _, s := range r.d.Stores {
		initStore(s)
		if len(s.Pairs) > 0 { // migrate pre-account data
			s.UserPairs["legacy"] = s.Pairs
			s.Pairs = nil
		}
	}
	return r, nil
}

func initMaps(d *data) {
	if d.Stores == nil {
		d.Stores = map[string]*Store{}
	}
	if d.Lists == nil {
		d.Lists = map[string]*List{}
	}
	if d.Templates == nil {
		d.Templates = map[string]*Template{}
	}
	if d.Users == nil {
		d.Users = map[string]*User{}
	}
	if d.Sessions == nil {
		d.Sessions = map[string]*Session{}
	}
}

func initStore(s *Store) {
	if s.ItemSection == nil {
		s.ItemSection = map[string]string{}
	}
	if s.UserPairs == nil {
		s.UserPairs = map[string]map[string]int{}
	}
	if s.Forks == nil {
		s.Forks = map[string][]string{}
	}
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
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, r.path)
}

func Normalize(s string) string { return strings.ToLower(strings.TrimSpace(s)) }

func PairKey(a, b string) string { return a + "|" + b }

// view resolves the store as seen by user: their fork (own trips only) or the
// public layout, where each user casts one vote per item pair. One user
// entering nonsense is one vote, however many trips they record.
func view(s *Store, user string) Store {
	c := Store{
		ID: s.ID, Name: s.Name, CreatedBy: s.CreatedBy, Trips: s.Trips,
		ItemSection: make(map[string]string, len(s.ItemSection)),
		Pairs:       map[string]int{},
	}
	for k, v := range s.ItemSection {
		c.ItemSection[k] = v
	}
	if fork, ok := s.Forks[user]; ok {
		c.Forked = true
		c.Sections = append([]string{}, fork...)
		for k, v := range s.UserPairs[user] {
			c.Pairs[k] = v
		}
		return c
	}
	c.Sections = append([]string{}, s.Sections...)
	for _, pairs := range s.UserPairs {
		for k, ab := range pairs {
			a, b, _ := strings.Cut(k, "|")
			if ab > pairs[PairKey(b, a)] {
				c.Pairs[k]++
			}
		}
	}
	return c
}

func (r *Repository) List(user string) []Store {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]Store, 0, len(r.d.Stores))
	for _, s := range r.d.Stores {
		out = append(out, view(s, user))
	}
	return out
}

func (r *Repository) Get(id, user string) (Store, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.d.Stores[id]
	if !ok {
		return Store{}, ErrNotFound
	}
	return view(s, user), nil
}

func (r *Repository) Create(name, user string, sections []string) (Store, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s := &Store{ID: r.newID("s"), Name: name, CreatedBy: user, Sections: sections}
	initStore(s)
	r.d.Stores[s.ID] = s
	return view(s, user), r.save()
}

// Update changes the section order and item→section assignments. The
// creator edits the public order; anyone else gets (or edits) a personal fork.
func (r *Repository) Update(id, user, name string, sections []string, itemSection map[string]string) (Store, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.d.Stores[id]
	if !ok {
		return Store{}, ErrNotFound
	}
	if name != "" && user == s.CreatedBy {
		s.Name = name
	}
	if sections != nil {
		_, forked := s.Forks[user]
		if user == s.CreatedBy && !forked {
			s.Sections = sections
		} else {
			s.Forks[user] = sections
		}
	}
	for k, v := range itemSection {
		if v == "" {
			delete(s.ItemSection, Normalize(k))
		} else {
			s.ItemSection[Normalize(k)] = v
		}
	}
	return view(s, user), r.save()
}

// Fork gives the user a personal copy of the current public section order;
// Unfork drops it and returns them to the shared layout.
func (r *Repository) Fork(id, user string) (Store, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.d.Stores[id]
	if !ok {
		return Store{}, ErrNotFound
	}
	if _, ok := s.Forks[user]; !ok {
		s.Forks[user] = append([]string{}, s.Sections...)
	}
	return view(s, user), r.save()
}

func (r *Repository) Unfork(id, user string) (Store, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.d.Stores[id]
	if !ok {
		return Store{}, ErrNotFound
	}
	delete(s.Forks, user)
	return view(s, user), r.save()
}

// Delete is only allowed for the creator while nobody else has used the store.
func (r *Repository) Delete(id, user string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.d.Stores[id]
	if !ok {
		return ErrNotFound
	}
	if s.CreatedBy != user {
		return ErrForbidden
	}
	for u := range s.UserPairs {
		if u != user {
			return ErrForbidden
		}
	}
	delete(r.d.Stores, id)
	return r.save()
}

// RecordTrip stores items in the order they were picked and updates the
// user's pairwise precedence counts.
func (r *Repository) RecordTrip(id, user string, items []string) (Store, error) {
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
	pairs := s.UserPairs[user]
	if pairs == nil {
		pairs = map[string]int{}
		s.UserPairs[user] = pairs
	}
	for i := range norm {
		for j := i + 1; j < len(norm); j++ {
			pairs[PairKey(norm[i], norm[j])]++
		}
	}
	s.Trips++
	r.d.Trips = append(r.d.Trips, Trip{StoreID: id, UserID: user, Items: norm, At: time.Now().UTC()})
	return view(s, user), r.save()
}
