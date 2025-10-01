package project02

import (
	"math"
	"sort"
	"strings"

	"github.com/kljensen/snowball/english"
)

// Hit is a scored search result.
type Hit struct {
	URL   string
	Score float64
}

// Index stores data for TF-IDF ranking.
type Index struct {
	tf     map[string]map[string]int // stem -> doc -> term freq
	df     map[string]int            // stem -> doc freq
	docLen map[string]int            // doc -> token count (after stop+stem)
	N      int                       // total documents
	stop   map[string]struct{}       // stopword set
}

// NewIndex creates an empty index. If stop is nil, uses DefaultStopwords().
func NewIndex(stop map[string]struct{}) *Index {
	if stop == nil {
		stop = DefaultStopwords()
	}
	return &Index{
		tf:     make(map[string]map[string]int),
		df:     make(map[string]int),
		docLen: make(map[string]int),
		stop:   stop,
	}
}

// internal stemmer
func stem(w string) string { return english.Stem(w, true) }

// Add indexes a single document. Pipeline: lower -> stop filter -> stem.
func (idx *Index) Add(doc string, words []string) {
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
		s := stem(lw)
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

// --- extracted comparator (outside SearchTFIDF) ---

// lessHit orders two hits: higher score first; if scores are equal, URL ascending.
// NOTE: `hits[j].URL > hits[i].URL` is equivalent to `hits[i].URL < hits[j].URL`.
func lessHit(a, b Hit) bool {
	if a.Score != b.Score {
		return a.Score > b.Score
	}
	return b.URL > a.URL
}

// SearchTFIDF ranks a single-term query using TF-IDF.
// Tie-break: if same score, order by URL (ascending) to keep results deterministic.
func (idx *Index) SearchTFIDF(term string) []Hit {
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

// BuildIndexFromURLList downloads and indexes a list of URLs.
func BuildIndexFromURLList(urls []string) (*Index, error) {
	idx := NewIndex(nil)
	for _, u := range urls {
		b, err := Download(u)
		if err != nil {
			continue
		}
		words, _ := Extract(b)
		idx.Add(u, words)
	}
	return idx, nil
}
