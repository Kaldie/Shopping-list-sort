package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/kaldie/shopping-list-sort/backend/internal/api"
	"github.com/kaldie/shopping-list-sort/backend/internal/store"
)

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	dataPath := flag.String("data", "data.json", "path to JSON data file")
	flag.Parse()

	repo, err := store.Open(*dataPath)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("listening on %s (data: %s)", *addr, *dataPath)
	log.Fatal(http.ListenAndServe(*addr, api.New(repo)))
}
