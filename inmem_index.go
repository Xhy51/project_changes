package project02

import (
	"math"
	"sort"
	"strings"
)

// InMemIndex stores data for TF-IDF ranking in memory.
type InMemIndex struct {
	tf     map[string]map[string]int // stem -> doc -> term freq
	df     map[string]int            // stem -> doc freq
	docLen map[string]int            // doc -> token count (after stop+stem)
	N      int                       // total documents
	stop   map[string]struct{}       // stopword set
}

// NewInMemIndex creates an empty in-memory index. If stop is nil, uses DefaultStopwords().
func NewInMemIndex(stop map[string]struct{}) *InMemIndex {
	if stop == nil {
		stop = DefaultStopwords()
	}
	return &InMemIndex{
		tf:     make(map[string]map[string]int),
		df:     make(map[string]int),
		docLen: make(map[string]int),
		stop:   stop,
	}
}

// Add indexes a single document. Pipeline: lower -> stop filter -> stem.
func (idx *InMemIndex) Add(doc string, words []string) {
	if _, dup := idx.docLen[doc]; dup {
		return
	}
	seen := make(map[string]bool)
	var kept int

	for _, w := range words {
		if w == "" {
			continue
		}
		lw := strings.ToLower(w)
		if _, bad := idx.stop[lw]; bad {
			continue
		}
		s := stem(w)
		if s == "" {
			continue
		}
		kept++
		if _, ok := idx.tf[s]; !ok {
			idx.tf[s] = make(map[string]int)
		}
		idx.tf[s][doc]++
		if !seen[s] {
			seen[s] = true
		}
	}
	for s := range seen {
		idx.df[s]++
	}
	idx.docLen[doc] = kept
	idx.N++
}

// GetN returns the total number of documents
func (idx *InMemIndex) GetN() int {
	return idx.N
}

// SearchTFIDF ranks a single-term query using TF-IDF.
func (idx *InMemIndex) SearchTFIDF(term string) []Hit {
	if term == "" || idx.N == 0 {
		return nil
	}
	q := strings.ToLower(term)
	if _, bad := idx.stop[q]; bad {
		return nil
	}
	s := stem(q)
	df := idx.df[s]
	if df == 0 {
		return nil
	}
	idf := math.Log(float64(idx.N) / float64(df))

	hits := make([]Hit, 0, len(idx.tf[s]))
	for doc, tfreq := range idx.tf[s] {
		den := idx.docLen[doc]
		if den == 0 {
			continue
		}
		tf := float64(tfreq) / float64(den)
		hits = append(hits, Hit{URL: doc, Score: tf * idf})
	}

	// Use the extracted comparator for clarity and reuse.
	sort.Slice(hits, func(i, j int) bool {
		return lessHit(hits[i], hits[j])
	})
	return hits
}

// Close closes the indexer resources
func (idx *InMemIndex) Close() error {
	// No resources to close for in-memory index
	return nil
}
