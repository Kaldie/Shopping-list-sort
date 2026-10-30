// Package api exposes the store, layout and sorting over HTTP/JSON.
package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/kaldie/shopping-list-sort/backend/internal/layout"
	"github.com/kaldie/shopping-list-sort/backend/internal/store"
)

type Server struct{ repo *store.Repository }

func New(repo *store.Repository) http.Handler {
	s := &Server{repo: repo}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/stores", s.listStores)
	mux.HandleFunc("POST /api/stores", s.createStore)
	mux.HandleFunc("GET /api/stores/{id}", s.getStore)
	mux.HandleFunc("PUT /api/stores/{id}", s.updateStore)
	mux.HandleFunc("DELETE /api/stores/{id}", s.deleteStore)
	mux.HandleFunc("GET /api/stores/{id}/layout", s.getLayout)
	mux.HandleFunc("POST /api/stores/{id}/sort", s.sortList)
	mux.HandleFunc("POST /api/stores/{id}/trips", s.recordTrip)
	mux.HandleFunc("POST /api/ocr", s.ocr)
	return cors(mux)
}

type storeReq struct {
	Name        string            `json:"name"`
	Sections    []string          `json:"sections"`
	ItemSection map[string]string `json:"itemSection"`
}

type itemsReq struct {
	Items []string `json:"items"`
}

func (s *Server) listStores(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.repo.List())
}

func (s *Server) createStore(w http.ResponseWriter, r *http.Request) {
	var req storeReq
	if !readJSON(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		httpError(w, http.StatusBadRequest, "name is required")
		return
	}
	st, err := s.repo.Create(req.Name, cleanLines(req.Sections))
	respond(w, http.StatusCreated, st, err)
}

func (s *Server) getStore(w http.ResponseWriter, r *http.Request) {
	st, err := s.repo.Get(r.PathValue("id"))
	respond(w, http.StatusOK, st, err)
}

func (s *Server) updateStore(w http.ResponseWriter, r *http.Request) {
	var req storeReq
	if !readJSON(w, r, &req) {
		return
	}
	var secs []string
	if req.Sections != nil {
		secs = cleanLines(req.Sections)
	}
	st, err := s.repo.Update(r.PathValue("id"), req.Name, secs, req.ItemSection)
	respond(w, http.StatusOK, st, err)
}

func (s *Server) deleteStore(w http.ResponseWriter, r *http.Request) {
	if err := s.repo.Delete(r.PathValue("id")); err != nil {
		respond(w, 0, nil, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) getLayout(w http.ResponseWriter, r *http.Request) {
	st, err := s.repo.Get(r.PathValue("id"))
	if err != nil {
		respond(w, 0, nil, err)
		return
	}
	writeJSON(w, http.StatusOK, layout.Generate(st))
}

func (s *Server) sortList(w http.ResponseWriter, r *http.Request) {
	var req itemsReq
	if !readJSON(w, r, &req) {
		return
	}
	st, err := s.repo.Get(r.PathValue("id"))
	if err != nil {
		respond(w, 0, nil, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"groups": layout.Sort(st, req.Items)})
}

func (s *Server) recordTrip(w http.ResponseWriter, r *http.Request) {
	var req itemsReq
	if !readJSON(w, r, &req) {
		return
	}
	st, err := s.repo.RecordTrip(r.PathValue("id"), cleanLines(req.Items))
	respond(w, http.StatusOK, st, err)
}

// ocr accepts a multipart "image" and returns the recognised text lines.
// Requires the tesseract binary; returns 501 otherwise.
func (s *Server) ocr(w http.ResponseWriter, r *http.Request) {
	bin, err := exec.LookPath("tesseract")
	if err != nil {
		httpError(w, http.StatusNotImplemented, "OCR unavailable: install tesseract-ocr on the server")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 20<<20)
	f, _, err := r.FormFile("image")
	if err != nil {
		httpError(w, http.StatusBadRequest, "multipart field 'image' required")
		return
	}
	defer f.Close()
	tmp, err := os.CreateTemp("", "ocr-*")
	if err != nil {
		httpError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer os.Remove(tmp.Name())
	if _, err := io.Copy(tmp, f); err != nil {
		tmp.Close()
		httpError(w, http.StatusBadRequest, err.Error())
		return
	}
	tmp.Close()
	lang := r.URL.Query().Get("lang")
	if lang == "" {
		lang = "eng"
	}
	out, err := exec.CommandContext(r.Context(), bin, tmp.Name(), "stdout", "-l", lang).Output()
	if err != nil {
		httpError(w, http.StatusUnprocessableEntity, "OCR failed: "+err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"lines": cleanLines(strings.Split(string(out), "\n"))})
}

func cleanLines(in []string) []string {
	out := []string{}
	for _, l := range in {
		l = strings.TrimSpace(strings.TrimLeft(strings.TrimSpace(l), "-*•☐□[] "))
		if l != "" {
			out = append(out, l)
		}
	}
	return out
}

func readJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(v); err != nil {
		httpError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return false
	}
	return true
}

func respond(w http.ResponseWriter, status int, v any, err error) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		httpError(w, http.StatusNotFound, "store not found")
	case err != nil:
		httpError(w, http.StatusInternalServerError, err.Error())
	default:
		writeJSON(w, status, v)
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func httpError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func cors(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		h.ServeHTTP(w, r)
	})
}
