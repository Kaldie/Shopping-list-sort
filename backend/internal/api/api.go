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

	"github.com/kaldie/shopping-list-sort/backend/internal/auth"
	"github.com/kaldie/shopping-list-sort/backend/internal/layout"
	"github.com/kaldie/shopping-list-sort/backend/internal/receipt"
	"github.com/kaldie/shopping-list-sort/backend/internal/store"
)

type Config struct {
	// GoogleClientID enables Google Sign-In. Empty enables dev login instead.
	GoogleClientID string
	// SecureCookies sets the Secure flag (serve over HTTPS in production).
	SecureCookies bool
}

type Server struct {
	repo     *store.Repository
	cfg      Config
	verifier *auth.Verifier
}

func New(repo *store.Repository, cfg Config) http.Handler {
	s := &Server{repo: repo, cfg: cfg}
	if cfg.GoogleClientID != "" {
		s.verifier = auth.NewVerifier(cfg.GoogleClientID)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/auth/config", s.authConfig)
	mux.HandleFunc("POST /api/auth/google", s.loginGoogle)
	mux.HandleFunc("POST /api/auth/dev", s.loginDev)
	mux.HandleFunc("POST /api/auth/logout", s.logout)
	mux.HandleFunc("GET /api/me", s.authed(s.me))

	mux.HandleFunc("GET /api/stores", s.authed(s.listStores))
	mux.HandleFunc("POST /api/stores", s.authed(s.createStore))
	mux.HandleFunc("GET /api/stores/{id}", s.authed(s.getStore))
	mux.HandleFunc("PUT /api/stores/{id}", s.authed(s.updateStore))
	mux.HandleFunc("DELETE /api/stores/{id}", s.authed(s.deleteStore))
	mux.HandleFunc("POST /api/stores/{id}/fork", s.authed(s.forkStore))
	mux.HandleFunc("DELETE /api/stores/{id}/fork", s.authed(s.unforkStore))
	mux.HandleFunc("GET /api/stores/{id}/layout", s.authed(s.getLayout))
	mux.HandleFunc("POST /api/stores/{id}/sort", s.authed(s.sortList))
	mux.HandleFunc("POST /api/stores/{id}/trips", s.authed(s.recordTrip))
	mux.HandleFunc("POST /api/stores/{id}/receipt", s.authed(s.applyReceipt))
	mux.HandleFunc("POST /api/ocr", s.authed(s.ocr))
	mux.HandleFunc("POST /api/receipt/parse", s.authed(s.parseReceipt))
	s.registerLists(mux)
	return mux
}

type storeReq struct {
	Name        string            `json:"name"`
	Sections    []string          `json:"sections"`
	ItemSection map[string]string `json:"itemSection"`
}

type itemsReq struct {
	Items []string `json:"items"`
}

func (s *Server) listStores(w http.ResponseWriter, r *http.Request, u store.User) {
	writeJSON(w, http.StatusOK, s.repo.List(u.ID))
}

func (s *Server) createStore(w http.ResponseWriter, r *http.Request, u store.User) {
	var req storeReq
	if !readJSON(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		httpError(w, http.StatusBadRequest, "name is required")
		return
	}
	st, err := s.repo.Create(req.Name, u.ID, cleanLines(req.Sections))
	respond(w, http.StatusCreated, st, err)
}

func (s *Server) getStore(w http.ResponseWriter, r *http.Request, u store.User) {
	st, err := s.repo.Get(r.PathValue("id"), u.ID)
	respond(w, http.StatusOK, st, err)
}

func (s *Server) updateStore(w http.ResponseWriter, r *http.Request, u store.User) {
	var req storeReq
	if !readJSON(w, r, &req) {
		return
	}
	var secs []string
	if req.Sections != nil {
		secs = cleanLines(req.Sections)
	}
	st, err := s.repo.Update(r.PathValue("id"), u.ID, req.Name, secs, req.ItemSection)
	respond(w, http.StatusOK, st, err)
}

func (s *Server) deleteStore(w http.ResponseWriter, r *http.Request, u store.User) {
	if err := s.repo.Delete(r.PathValue("id"), u.ID); err != nil {
		respond(w, 0, nil, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) forkStore(w http.ResponseWriter, r *http.Request, u store.User) {
	st, err := s.repo.Fork(r.PathValue("id"), u.ID)
	respond(w, http.StatusOK, st, err)
}

func (s *Server) unforkStore(w http.ResponseWriter, r *http.Request, u store.User) {
	st, err := s.repo.Unfork(r.PathValue("id"), u.ID)
	respond(w, http.StatusOK, st, err)
}

func (s *Server) getLayout(w http.ResponseWriter, r *http.Request, u store.User) {
	st, err := s.repo.Get(r.PathValue("id"), u.ID)
	if err != nil {
		respond(w, 0, nil, err)
		return
	}
	writeJSON(w, http.StatusOK, layout.Generate(st))
}

func (s *Server) sortList(w http.ResponseWriter, r *http.Request, u store.User) {
	var req itemsReq
	if !readJSON(w, r, &req) {
		return
	}
	st, err := s.repo.Get(r.PathValue("id"), u.ID)
	if err != nil {
		respond(w, 0, nil, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"groups": layout.Sort(st, req.Items)})
}

func (s *Server) recordTrip(w http.ResponseWriter, r *http.Request, u store.User) {
	var req itemsReq
	if !readJSON(w, r, &req) {
		return
	}
	st, err := s.repo.RecordTrip(r.PathValue("id"), u.ID, cleanLines(req.Items))
	respond(w, http.StatusOK, st, err)
}

// parseReceipt extracts the ordered item lines from receipt text and, given
// a listId, maps each line to an item on that list by keyword.
func (s *Server) parseReceipt(w http.ResponseWriter, r *http.Request, u store.User) {
	var req struct {
		Text   string `json:"text"`
		ListID string `json:"listId"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	var names []string
	if req.ListID != "" {
		l, err := s.repo.GetList(req.ListID, u.ID)
		if err != nil {
			respond(w, 0, nil, err)
			return
		}
		for _, it := range l.Items {
			names = append(names, it.Name)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"matches": receipt.MatchLines(receipt.Parse(req.Text), names)})
}

type receiptLine struct {
	Item   string `json:"item"`
	Order  bool   `json:"order"`  // use for learning the walking order
	Bought bool   `json:"bought"` // counts as bought for statistics
}

// applyReceipt treats a receipt as bought: it records the walking order as a
// trip and completes the chosen list (or a new one) with the bought items.
func (s *Server) applyReceipt(w http.ResponseWriter, r *http.Request, u store.User) {
	var req struct {
		ListID string        `json:"listId"`
		Lines  []receiptLine `json:"lines"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	storeID := r.PathValue("id")
	if _, err := s.repo.Get(storeID, u.ID); err != nil {
		respond(w, 0, nil, err)
		return
	}
	var order []string
	var bought []store.Item
	for _, l := range req.Lines {
		if strings.TrimSpace(l.Item) == "" || !l.Bought {
			continue
		}
		bought = append(bought, store.Item{Name: l.Item, Qty: 1})
		if l.Order {
			order = append(order, l.Item)
		}
	}
	if len(bought) == 0 {
		httpError(w, http.StatusBadRequest, "no bought items")
		return
	}
	listID := req.ListID
	if listID == "" {
		l, err := s.repo.CreateList(u.ID, "Receipt", nil, "")
		if err != nil {
			respond(w, 0, nil, err)
			return
		}
		listID = l.ID
	}
	l, err := s.repo.CompleteList(listID, u.ID, bought)
	if err != nil {
		respond(w, 0, nil, err)
		return
	}
	st, err := s.repo.Get(storeID, u.ID)
	if len(order) >= 2 {
		st, err = s.repo.RecordTrip(storeID, u.ID, order)
	}
	respond(w, http.StatusOK, map[string]any{"list": l, "store": st}, err)
}

// ocr accepts a multipart "image" (photo or PDF) and returns its text lines.
// Photos need tesseract, PDFs (digital receipts) need pdftotext.
func (s *Server) ocr(w http.ResponseWriter, r *http.Request, _ store.User) {
	r.Body = http.MaxBytesReader(w, r.Body, 20<<20)
	f, hdr, err := r.FormFile("image")
	if err != nil {
		httpError(w, http.StatusBadRequest, "multipart field 'image' required")
		return
	}
	defer f.Close()
	head := make([]byte, 5)
	n, _ := io.ReadFull(f, head)
	isPDF := string(head[:n]) == "%PDF-" || strings.HasSuffix(strings.ToLower(hdr.Filename), ".pdf")
	tool := "tesseract"
	if isPDF {
		tool = "pdftotext"
	}
	bin, err := exec.LookPath(tool)
	if err != nil {
		httpError(w, http.StatusNotImplemented, "unavailable: install "+tool+" on the server")
		return
	}
	tmp, err := os.CreateTemp("", "ocr-*")
	if err != nil {
		httpError(w, http.StatusInternalServerError, err.Error())
		return
	}
	defer os.Remove(tmp.Name())
	_, err = io.Copy(tmp, io.MultiReader(strings.NewReader(string(head[:n])), f))
	tmp.Close()
	if err != nil {
		httpError(w, http.StatusBadRequest, err.Error())
		return
	}
	var cmd *exec.Cmd
	if isPDF {
		cmd = exec.CommandContext(r.Context(), bin, "-layout", tmp.Name(), "-")
	} else {
		lang := r.URL.Query().Get("lang")
		if lang == "" {
			lang = "eng"
		}
		cmd = exec.CommandContext(r.Context(), bin, tmp.Name(), "stdout", "-l", lang)
	}
	out, err := cmd.Output()
	if err != nil {
		httpError(w, http.StatusUnprocessableEntity, "reading failed: "+err.Error())
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
		httpError(w, http.StatusNotFound, "not found")
	case errors.Is(err, store.ErrForbidden):
		httpError(w, http.StatusForbidden, "not allowed")
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
