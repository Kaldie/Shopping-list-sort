package api

import (
	"net/http"
	"sort"
	"strconv"

	"github.com/kaldie/shopping-list-sort/backend/internal/stats"
	"github.com/kaldie/shopping-list-sort/backend/internal/store"
)

func (s *Server) registerLists(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/lists", s.authed(s.listLists))
	mux.HandleFunc("POST /api/lists", s.authed(s.createList))
	mux.HandleFunc("GET /api/lists/{id}", s.authed(s.getList))
	mux.HandleFunc("PUT /api/lists/{id}", s.authed(s.updateList))
	mux.HandleFunc("DELETE /api/lists/{id}", s.authed(s.deleteList))
	mux.HandleFunc("POST /api/lists/{id}/complete", s.authed(s.completeList))
	mux.HandleFunc("POST /api/lists/{id}/save-as-template", s.authed(s.saveAsTemplate))

	mux.HandleFunc("GET /api/templates", s.authed(s.listTemplates))
	mux.HandleFunc("POST /api/templates", s.authed(s.createTemplate))
	mux.HandleFunc("GET /api/templates/{id}", s.authed(s.getTemplate))
	mux.HandleFunc("PUT /api/templates/{id}", s.authed(s.updateTemplate))
	mux.HandleFunc("DELETE /api/templates/{id}", s.authed(s.deleteTemplate))
	mux.HandleFunc("POST /api/templates/{id}/new-list", s.authed(s.newListFromTemplate))

	mux.HandleFunc("GET /api/stats", s.authed(s.getStats))
}

type listReq struct {
	Name  string       `json:"name"`
	Items []store.Item `json:"items"`
}

func (s *Server) listLists(w http.ResponseWriter, r *http.Request, u store.User) {
	ls := s.repo.Lists(u.ID)
	sort.Slice(ls, func(i, j int) bool { return ls[i].CreatedAt.After(ls[j].CreatedAt) })
	writeJSON(w, http.StatusOK, ls)
}

func (s *Server) createList(w http.ResponseWriter, r *http.Request, u store.User) {
	var req listReq
	if !readJSON(w, r, &req) {
		return
	}
	l, err := s.repo.CreateList(u.ID, req.Name, req.Items, "")
	respond(w, http.StatusCreated, l, err)
}

func (s *Server) getList(w http.ResponseWriter, r *http.Request, u store.User) {
	l, err := s.repo.GetList(r.PathValue("id"), u.ID)
	respond(w, http.StatusOK, l, err)
}

func (s *Server) updateList(w http.ResponseWriter, r *http.Request, u store.User) {
	var req listReq
	if !readJSON(w, r, &req) {
		return
	}
	l, err := s.repo.UpdateList(r.PathValue("id"), u.ID, req.Name, req.Items)
	respond(w, http.StatusOK, l, err)
}

func (s *Server) deleteList(w http.ResponseWriter, r *http.Request, u store.User) {
	if err := s.repo.DeleteList(r.PathValue("id"), u.ID); err != nil {
		respond(w, 0, nil, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// completeList marks the list bought. Optional body {"items": [...]} with
// what was actually bought; without it the list's items are taken as bought.
func (s *Server) completeList(w http.ResponseWriter, r *http.Request, u store.User) {
	var req listReq
	if r.ContentLength != 0 && !readJSON(w, r, &req) {
		return
	}
	l, err := s.repo.CompleteList(r.PathValue("id"), u.ID, req.Items)
	respond(w, http.StatusOK, l, err)
}

func (s *Server) saveAsTemplate(w http.ResponseWriter, r *http.Request, u store.User) {
	var req listReq
	if !readJSON(w, r, &req) {
		return
	}
	l, err := s.repo.GetList(r.PathValue("id"), u.ID)
	if err != nil {
		respond(w, 0, nil, err)
		return
	}
	if req.Name == "" {
		req.Name = l.Name
	}
	t, err := s.repo.CreateTemplate(u.ID, req.Name, l.Items)
	respond(w, http.StatusCreated, t, err)
}

func (s *Server) listTemplates(w http.ResponseWriter, r *http.Request, u store.User) {
	ts := s.repo.Templates(u.ID)
	sort.Slice(ts, func(i, j int) bool { return ts[i].Name < ts[j].Name })
	writeJSON(w, http.StatusOK, ts)
}

func (s *Server) createTemplate(w http.ResponseWriter, r *http.Request, u store.User) {
	var req listReq
	if !readJSON(w, r, &req) {
		return
	}
	if req.Name == "" {
		httpError(w, http.StatusBadRequest, "name is required")
		return
	}
	t, err := s.repo.CreateTemplate(u.ID, req.Name, req.Items)
	respond(w, http.StatusCreated, t, err)
}

func (s *Server) getTemplate(w http.ResponseWriter, r *http.Request, u store.User) {
	t, err := s.repo.GetTemplate(r.PathValue("id"), u.ID)
	respond(w, http.StatusOK, t, err)
}

func (s *Server) updateTemplate(w http.ResponseWriter, r *http.Request, u store.User) {
	var req listReq
	if !readJSON(w, r, &req) {
		return
	}
	t, err := s.repo.UpdateTemplate(r.PathValue("id"), u.ID, req.Name, req.Items)
	respond(w, http.StatusOK, t, err)
}

func (s *Server) deleteTemplate(w http.ResponseWriter, r *http.Request, u store.User) {
	if err := s.repo.DeleteTemplate(r.PathValue("id"), u.ID); err != nil {
		respond(w, 0, nil, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) newListFromTemplate(w http.ResponseWriter, r *http.Request, u store.User) {
	l, err := s.repo.ListFromTemplate(r.PathValue("id"), u.ID, r.URL.Query().Get("name"))
	respond(w, http.StatusCreated, l, err)
}

// getStats: ?templateId= limits to lists made from that template and adds
// template suggestions; ?window=N looks at the last N completed lists.
func (s *Server) getStats(w http.ResponseWriter, r *http.Request, u store.User) {
	var tmpl *store.Template
	if id := r.URL.Query().Get("templateId"); id != "" {
		t, err := s.repo.GetTemplate(id, u.ID)
		if err != nil {
			respond(w, 0, nil, err)
			return
		}
		tmpl = &t
	}
	window, _ := strconv.Atoi(r.URL.Query().Get("window"))
	writeJSON(w, http.StatusOK, stats.Compute(s.repo.Lists(u.ID), tmpl, window))
}
