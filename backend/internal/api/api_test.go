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
