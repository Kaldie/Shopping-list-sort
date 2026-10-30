package store

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

type User struct {
	ID      string `json:"id"` // e.g. "google:<sub>"
	Email   string `json:"email"`
	Name    string `json:"name"`
	Picture string `json:"picture,omitempty"`
}

type Session struct {
	UserID  string    `json:"userId"`
	Expires time.Time `json:"expires"`
}

const sessionTTL = 30 * 24 * time.Hour

// UpsertUser stores the profile and starts a session, returning its token.
func (r *Repository) UpsertUser(u User) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token := hex.EncodeToString(b)
	r.mu.Lock()
	defer r.mu.Unlock()
	r.d.Users[u.ID] = &u
	now := time.Now()
	for k, s := range r.d.Sessions {
		if now.After(s.Expires) {
			delete(r.d.Sessions, k)
		}
	}
	r.d.Sessions[token] = &Session{UserID: u.ID, Expires: now.Add(sessionTTL)}
	return token, r.save()
}

// SessionUser returns the user for a valid session token.
func (r *Repository) SessionUser(token string) (User, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.d.Sessions[token]
	if !ok || time.Now().After(s.Expires) {
		return User{}, false
	}
	u, ok := r.d.Users[s.UserID]
	if !ok {
		return User{}, false
	}
	return *u, true
}

func (r *Repository) EndSession(token string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.d.Sessions, token)
	return r.save()
}
