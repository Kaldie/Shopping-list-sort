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
		DevLogin:       os.Getenv("DEV_LOGIN") == "true",
	}
	if cfg.GoogleClientID == "" {
		if !cfg.DevLogin {
			log.Fatal("GOOGLE_CLIENT_ID not set: configure Google Sign-In, or set DEV_LOGIN=true for local development only")
		}
		log.Print("WARNING: DEV_LOGIN=true: anyone can sign in as any name without a password")
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
		// Hashed assets never change; everything else (index.html) must be
		// revalidated, or browsers keep running an old bundle after a deploy.
		if strings.HasPrefix(r.URL.Path, "/assets/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		} else {
			w.Header().Set("Cache-Control", "no-cache")
		}
		p := filepath.Join(dir, filepath.Clean("/"+r.URL.Path))
		if _, err := os.Stat(p); err != nil && !strings.HasPrefix(r.URL.Path, "/assets/") {
			http.ServeFile(w, r, filepath.Join(dir, "index.html"))
			return
		}
		fs.ServeHTTP(w, r)
	})
}
