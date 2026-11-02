package store

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

// These tests need a scratch database: set TEST_DATABASE_URL, e.g.
//
//	TEST_DATABASE_URL=postgres://user:pw@host:5432/shoppinglist_test go test ./internal/store
//
// They TRUNCATE every table, so never point them at real data.
func pgTestURL(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(schema); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`TRUNCATE stores, lists, templates, users, sessions, trips, meta RESTART IDENTITY`); err != nil {
		t.Fatal(err)
	}
	return dsn
}

func TestPostgresRoundTrip(t *testing.T) {
	dsn := pgTestURL(t)
	r, err := OpenPostgres(dsn, "")
	if err != nil {
		t.Fatal(err)
	}
	s, _ := r.Create("AH", "alice", []string{"Produce", "Dairy"})
	r.RecordTrip(s.ID, "alice", []string{"apples", "milk"})
	r.RecordTrip(s.ID, "bob", []string{"milk", "apples"})
	r.Update(s.ID, "bob", "", []string{"Dairy", "Produce"}, map[string]string{"milk": "Dairy"})
	l, _ := r.CreateList("alice", "wk", []Item{{Name: "milk", Qty: 2}}, "")
	tm, _ := r.CreateTemplate("alice", "base", []Item{{Name: "bread"}})
	token, err := r.UpsertUser(User{ID: "google:1", Email: "a@example.com", Name: "Alice"})
	if err != nil {
		t.Fatal(err)
	}
	gone, _ := r.CreateList("alice", "tmp", nil, "")
	if err := r.DeleteList(gone.ID, "alice"); err != nil {
		t.Fatal(err)
	}

	// A fresh process sees everything that was written.
	r2, err := OpenPostgres(dsn, "")
	if err != nil {
		t.Fatal(err)
	}
	v, err := r2.Get(s.ID, "alice")
	if err != nil || v.Trips != 2 || v.Pairs["apples|milk"] != 1 || v.ItemSection["milk"] != "Dairy" {
		t.Fatalf("store after reload: %+v err=%v", v, err)
	}
	if bob, _ := r2.Get(s.ID, "bob"); !bob.Forked || bob.Sections[0] != "Dairy" {
		t.Fatalf("fork lost: %+v", bob)
	}
	if got, err := r2.GetList(l.ID, "alice"); err != nil || got.Items[0].Qty != 2 {
		t.Fatalf("list after reload: %+v err=%v", got, err)
	}
	if _, err := r2.GetList(gone.ID, "alice"); err != ErrNotFound {
		t.Fatalf("deleted list came back: %v", err)
	}
	if _, err := r2.GetTemplate(tm.ID, "alice"); err != nil {
		t.Fatal(err)
	}
	if u, ok := r2.SessionUser(token); !ok || u.Email != "a@example.com" {
		t.Fatalf("session lost: %+v %v", u, ok)
	}
	if len(r2.d.Trips) != 2 {
		t.Fatalf("trips: %d", len(r2.d.Trips))
	}
	// IDs keep counting instead of colliding with existing rows.
	if n, _ := r2.CreateList("alice", "next", nil, ""); n.ID == l.ID || n.ID == gone.ID || n.ID == s.ID {
		t.Fatalf("id reused: %s", n.ID)
	}
	// Saving an unchanged repository must not rewrite rows or duplicate trips.
	if err := r2.save(); err != nil {
		t.Fatal(err)
	}
	r3, _ := OpenPostgres(dsn, "")
	if len(r3.d.Trips) != 2 {
		t.Fatalf("trips duplicated: %d", len(r3.d.Trips))
	}
}

func TestPostgresImportsJSONOnce(t *testing.T) {
	dsn := pgTestURL(t)
	file := filepath.Join(t.TempDir(), "data.json")
	jr, _ := Open(file)
	s, _ := jr.Create("AH", "alice", []string{"Produce"})
	jr.RecordTrip(s.ID, "alice", []string{"apples", "milk"})
	jr.CreateList("alice", "wk", []Item{{Name: "milk"}}, "")
	jr.UpsertUser(User{ID: "google:1", Name: "Alice"})

	r, err := OpenPostgres(dsn, file)
	if err != nil {
		t.Fatal(err)
	}
	if v, err := r.Get(s.ID, "alice"); err != nil || v.Trips != 1 {
		t.Fatalf("not imported: %+v %v", v, err)
	}

	// Later changes go to Postgres only; the JSON file is not re-imported.
	r.CreateList("alice", "second", nil, "")
	r2, err := OpenPostgres(dsn, file)
	if err != nil {
		t.Fatal(err)
	}
	if len(r2.Lists("alice")) != 2 {
		t.Fatalf("lists after reopen: %d", len(r2.Lists("alice")))
	}
}
