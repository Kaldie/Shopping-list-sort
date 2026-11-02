package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/kaldie/shopping-list-sort/backend/internal/api"
	"github.com/kaldie/shopping-list-sort/backend/internal/store"
)

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	dataPath := flag.String("data", "data.json", "path to JSON data file (imported once when DATABASE_URL is set)")
	static := flag.String("static", "", "directory with the built frontend to serve (optional)")
	flag.Parse()

	// With DATABASE_URL the data lives in Postgres and -data is only read once,
	// to import an existing JSON file into an empty database.
	var repo *store.Repository
	var err error
	if dsn := os.Getenv("DATABASE_URL"); dsn != "" {
		repo, err = store.OpenPostgres(dsn, *dataPath)
	} else {
		repo, err = store.Open(*dataPath)
	}
	if err != nil {
		log.Fatal(err)
	}
	cfg := api.Config{
		GoogleClientID: os.Getenv("GOOGLE_CLIENT_ID"),
		SecureCookies:  os.Getenv("SECURE_COOKIES") == "true",
	}
	if cfg.GoogleClientID == "" {
		log.Print("GOOGLE_CLIENT_ID not set: using unauthenticated dev login")
	}

	mux := http.NewServeMux()
	mux.Handle("/api/", api.New(repo, cfg))
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("ok")) })
	if *static != "" {
		mux.Handle("/", spa(*static))
	}
	log.Printf("listening on %s (data: %s)", *addr, *dataPath)
	log.Fatal(http.ListenAndServe(*addr, mux))
}

// spa serves files from dir and falls back to index.html for app routes.
func spa(dir string) http.Handler {
	fs := http.FileServer(http.Dir(dir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := filepath.Join(dir, filepath.Clean("/"+r.URL.Path))
		if _, err := os.Stat(p); err != nil && !strings.HasPrefix(r.URL.Path, "/assets/") {
			http.ServeFile(w, r, filepath.Join(dir, "index.html"))
			return
		}
		fs.ServeHTTP(w, r)
	})
}
