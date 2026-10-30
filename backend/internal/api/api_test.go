package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kaldie/shopping-list-sort/backend/internal/store"
)

func do(t *testing.T, h http.Handler, method, path, body string, out any) int {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(method, path, bytes.NewBufferString(body)))
	if out != nil {
		if err := json.Unmarshal(rec.Body.Bytes(), out); err != nil {
			t.Fatalf("%s %s: %v (%s)", method, path, err, rec.Body)
		}
	}
	return rec.Code
}

func TestFlow(t *testing.T) {
	repo, _ := store.Open("")
	h := New(repo)

	var st store.Store
	if c := do(t, h, "POST", "/api/stores", `{"name":"AH","sections":["Produce","Dairy"]}`, &st); c != 201 {
		t.Fatalf("create: %d", c)
	}
	if c := do(t, h, "POST", "/api/stores/"+st.ID+"/trips", `{"items":["bananas","yoghurt"]}`, nil); c != 200 {
		t.Fatalf("trip: %d", c)
	}
	var res struct {
		Groups []struct {
			Section string   `json:"section"`
			Items   []string `json:"items"`
		} `json:"groups"`
	}
	do(t, h, "POST", "/api/stores/"+st.ID+"/sort", `{"items":["melk","bananas"]}`, &res)
	if len(res.Groups) != 2 || res.Groups[0].Section != "Produce" || res.Groups[1].Section != "Dairy" {
		t.Fatalf("unexpected sort: %+v", res)
	}
	if c := do(t, h, "GET", "/api/stores/nope/layout", "", nil); c != 404 {
		t.Fatalf("missing store: %d", c)
	}
}

func TestTemplateFlow(t *testing.T) {
	repo, _ := store.Open("")
	h := New(repo)
	var l store.List
	do(t, h, "POST", "/api/lists", `{"name":"wk1","items":[{"name":"milk","qty":2},{"name":"bread"}]}`, &l)
	var tm store.Template
	if c := do(t, h, "POST", "/api/lists/"+l.ID+"/save-as-template", `{"name":"Weekly"}`, &tm); c != 201 || len(tm.Items) != 2 {
		t.Fatalf("save as template: %d %+v", c, tm)
	}
	var l2 store.List
	do(t, h, "POST", "/api/templates/"+tm.ID+"/new-list", "", &l2)
	if l2.TemplateID != tm.ID || len(l2.Items) != 2 {
		t.Fatalf("new list: %+v", l2)
	}
	if c := do(t, h, "POST", "/api/lists/"+l2.ID+"/complete", "", nil); c != 200 {
		t.Fatalf("complete: %d", c)
	}
	var rep struct {
		Lists int `json:"lists"`
	}
	do(t, h, "GET", "/api/stats?templateId="+tm.ID, "", &rep)
	if rep.Lists != 1 {
		t.Fatalf("stats lists=%d", rep.Lists)
	}
}
