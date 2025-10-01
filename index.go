package project02

import (
	"github.com/kljensen/snowball/english"
)

// Hit is a scored search result.
type Hit struct {
	URL   string
	Score float64
}

// Indexer 定义索引接口
type Indexer interface {
	// Add indexes a single document. Pipeline: lower -> stop filter -> stem.
	Add(doc string, words []string)

	// SearchTFIDF ranks a single-term query using TF-IDF.
	SearchTFIDF(term string) []Hit

	// GetN returns the total number of documents
	GetN() int

	// Close closes the indexer resources
	Close() error
}

// lessHit orders two hits: higher score first; if scores are equal, URL ascending.
func lessHit(a, b Hit) bool {
	if a.Score != b.Score {
		return a.Score > b.Score
	}
	return a.URL < b.URL
}

// internal stemmer
func stem(w string) string { return english.Stem(w, true) }
