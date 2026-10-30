package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kaldie/shopping-list-sort/backend/internal/store"
)

type client struct {
	t      *testing.T
	h      http.Handler
	cookie *http.Cookie
}

func login(t *testing.T, h http.Handler, name string) *client {
	c := &client{t: t, h: h}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest("POST", "/api/auth/dev", bytes.NewBufferString(`{"name":"`+name+`"}`)))
	for _, ck := range rec.Result().Cookies() {
		if ck.Name == sessionCookie {
			c.cookie = ck
		}
	}
	if c.cookie == nil {
		t.Fatalf("login %s: %d %s", name, rec.Code, rec.Body)
	}
	return c
}

func (c *client) do(method, path, body string, out any) int {
	c.t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	if c.cookie != nil {
		req.AddCookie(c.cookie)
	}
	c.h.ServeHTTP(rec, req)
	if out != nil {
		if err := json.Unmarshal(rec.Body.Bytes(), out); err != nil {
			c.t.Fatalf("%s %s: %v (%s)", method, path, err, rec.Body)
		}
	}
	return rec.Code
}

func newAPI() http.Handler {
	repo, _ := store.Open("")
	return New(repo, Config{})
}

func TestRequiresLogin(t *testing.T) {
	anon := &client{t: t, h: newAPI()}
	if c := anon.do("GET", "/api/stores", "", nil); c != 401 {
		t.Fatalf("anonymous: %d", c)
	}
}

func TestFlow(t *testing.T) {
	c := login(t, newAPI(), "alice")
	var st store.Store
	if code := c.do("POST", "/api/stores", `{"name":"AH","sections":["Produce","Dairy"]}`, &st); code != 201 {
		t.Fatalf("create: %d", code)
	}
	if code := c.do("POST", "/api/stores/"+st.ID+"/trips", `{"items":["bananas","yoghurt"]}`, nil); code != 200 {
		t.Fatalf("trip: %d", code)
	}
	var res struct {
		Groups []struct {
			Section string   `json:"section"`
			Items   []string `json:"items"`
		} `json:"groups"`
	}
	c.do("POST", "/api/stores/"+st.ID+"/sort", `{"items":["melk","bananas"]}`, &res)
	if len(res.Groups) != 2 || res.Groups[0].Section != "Produce" || res.Groups[1].Section != "Dairy" {
		t.Fatalf("unexpected sort: %+v", res)
	}
	if code := c.do("GET", "/api/stores/nope/layout", "", nil); code != 404 {
		t.Fatalf("missing store: %d", code)
	}
}

func TestTemplateFlow(t *testing.T) {
	h := newAPI()
	c := login(t, h, "alice")
	var l store.List
	c.do("POST", "/api/lists", `{"name":"wk1","items":[{"name":"milk","qty":2},{"name":"bread"}]}`, &l)
	var tm store.Template
	if code := c.do("POST", "/api/lists/"+l.ID+"/save-as-template", `{"name":"Weekly"}`, &tm); code != 201 || len(tm.Items) != 2 {
		t.Fatalf("save as template: %d %+v", code, tm)
	}
	var l2 store.List
	c.do("POST", "/api/templates/"+tm.ID+"/new-list", "", &l2)
	if l2.TemplateID != tm.ID || len(l2.Items) != 2 {
		t.Fatalf("new list: %+v", l2)
	}
	if code := c.do("POST", "/api/lists/"+l2.ID+"/complete", "", nil); code != 200 {
		t.Fatalf("complete: %d", code)
	}
	var rep struct {
		Lists int `json:"lists"`
	}
	c.do("GET", "/api/stats?templateId="+tm.ID, "", &rep)
	if rep.Lists != 1 {
		t.Fatalf("stats lists=%d", rep.Lists)
	}

	bob := login(t, h, "bob")
	if code := bob.do("GET", "/api/lists/"+l.ID, "", nil); code != 404 {
		t.Fatalf("bob read alice's list: %d", code)
	}
}

func TestReceiptFlow(t *testing.T) {
	c := login(t, newAPI(), "alice")
	var st store.Store
	c.do("POST", "/api/stores", `{"name":"AH"}`, &st)
	var l store.List
	c.do("POST", "/api/lists", `{"items":[{"name":"melk"},{"name":"bananen"}]}`, &l)

	var parsed struct {
		Matches []struct {
			Item    string `json:"item"`
			Matched bool   `json:"matched"`
		} `json:"matches"`
	}
	text := `AH BANANEN 1,99\nAH HALFV MELK 1L 1,19\nCHIPS 2,00\nTOTAAL 5,18`
	c.do("POST", "/api/receipt/parse", `{"listId":"`+l.ID+`","text":"`+text+`"}`, &parsed)
	if len(parsed.Matches) != 3 || parsed.Matches[1].Item != "melk" || parsed.Matches[2].Matched {
		t.Fatalf("matches: %+v", parsed.Matches)
	}

	var res struct {
		List  store.List  `json:"list"`
		Store store.Store `json:"store"`
	}
	body := `{"listId":"` + l.ID + `","lines":[
		{"item":"bananen","order":true,"bought":true},
		{"item":"melk","order":true,"bought":true},
		{"item":"CHIPS","order":false,"bought":false}]}`
	if code := c.do("POST", "/api/stores/"+st.ID+"/receipt", body, &res); code != 200 {
		t.Fatalf("apply: %d", code)
	}
	if res.List.CompletedAt == nil || len(res.List.Items) != 2 || res.Store.Trips != 1 {
		t.Fatalf("apply result: %+v", res)
	}
}
