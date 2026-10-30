package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/kaldie/shopping-list-sort/backend/internal/store"
)

const sessionCookie = "sls_session"

type authedHandler func(http.ResponseWriter, *http.Request, store.User)

// authed rejects requests without a valid session cookie.
func (s *Server) authed(h authedHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(sessionCookie)
		if err != nil {
			httpError(w, http.StatusUnauthorized, "sign in required")
			return
		}
		u, ok := s.repo.SessionUser(c.Value)
		if !ok {
			httpError(w, http.StatusUnauthorized, "session expired")
			return
		}
		h(w, r, u)
	}
}

func (s *Server) authConfig(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"googleClientId": s.cfg.GoogleClientID,
		"devLogin":       s.verifier == nil,
	})
}

func (s *Server) me(w http.ResponseWriter, _ *http.Request, u store.User) {
	writeJSON(w, http.StatusOK, u)
}

// loginGoogle exchanges a Google ID token (from Google Identity Services in
// the browser) for a session cookie.
func (s *Server) loginGoogle(w http.ResponseWriter, r *http.Request) {
	if s.verifier == nil {
		httpError(w, http.StatusNotImplemented, "Google Sign-In not configured (set GOOGLE_CLIENT_ID)")
		return
	}
	var req struct {
		Credential string `json:"credential"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	c, err := s.verifier.Verify(r.Context(), req.Credential)
	if err != nil {
		httpError(w, http.StatusUnauthorized, "invalid Google token: "+err.Error())
		return
	}
	s.startSession(w, store.User{ID: "google:" + c.Sub, Email: c.Email, Name: c.Name, Picture: c.Picture})
}

// loginDev signs in by name without verification. Only available when
// Google Sign-In is not configured, i.e. local development.
func (s *Server) loginDev(w http.ResponseWriter, r *http.Request) {
	if s.verifier != nil {
		httpError(w, http.StatusNotFound, "not found")
		return
	}
	var req struct {
		Name string `json:"name"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = "dev"
	}
	s.startSession(w, store.User{ID: "dev:" + strings.ToLower(name), Name: name})
}

func (s *Server) startSession(w http.ResponseWriter, u store.User) {
	token, err := s.repo.UpsertUser(u)
	if err != nil {
		httpError(w, http.StatusInternalServerError, err.Error())
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookie, Value: token, Path: "/", HttpOnly: true,
		Secure: s.cfg.SecureCookies, SameSite: http.SameSiteLaxMode,
		Expires: time.Now().Add(30 * 24 * time.Hour),
	})
	writeJSON(w, http.StatusOK, u)
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookie); err == nil {
		_ = s.repo.EndSession(c.Value)
	}
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: "", Path: "/", MaxAge: -1})
	w.WriteHeader(http.StatusNoContent)
}
