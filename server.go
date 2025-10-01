package project02

import (
	"encoding/json"
	"net/http"
	"sort"
)

// NewMux serves ./top10 at /top10/ and provides /search?q=term.
// Library-only: does not start the server by itself.
func NewMux(indexer Indexer) http.Handler {
	mux := http.NewServeMux()

	// Redirect root to /top10/
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.Redirect(w, r, "/top10/", http.StatusFound)
			return
		}
		http.NotFound(w, r)
	})

	// Map /top10/* -> ./top10/*
	mux.Handle("/top10/", http.StripPrefix("/top10/",
		http.FileServer(http.Dir("./top10"))))

	// /search?q=term -> JSON hits
	mux.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("q")
		// Create a temporary in-memory index for search if using SQLite
		var hits []Hit
		if indexer != nil {
			hits = indexer.SearchTFIDF(q)
			// Sort hits by score (descending) and URL (ascending) for ties
			sort.Slice(hits, func(i, j int) bool {
				return lessHit(hits[i], hits[j])
			})
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(hits)
	})

	return mux
}
