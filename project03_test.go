package project03

import (
	"bytes"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// --- TestExtract ---

func TestExtract(t *testing.T) {
	html := `
	<!doctype html>
	<html>
	  <head>
	    <style>body{color:red}</style>
	    <script>var x=1</script>
	  </head>
	  <body>
	    <p>Hello, world! 42</p>
	    <a href="a.html">A</a>
	    <a href="/abs/b.html#frag">B</a>
	  </body>
	</html>`

	words, hrefs := Extract([]byte(html))

	// normalize to lower-case for robust match
	var lc []string
	for _, w := range words {
		lc = append(lc, strings.ToLower(w))
	}

	// must contain visible text (p tag)
	must := []string{"hello", "world", "42"}
	for _, m := range must {
		found := false
		for _, w := range lc {
			if w == m {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("Extract words missing %q; got %#v", m, words)
		}
	}

	// should NOT include script/style content
	bad := []string{"var", "color"}
	for _, b := range bad {
		for _, w := range lc {
			if w == b {
				t.Fatalf("Extract words should exclude script/style; found %q in %#v", b, words)
			}
		}
	}

	// hrefs should be collected as-is (raw, not cleaned)
	wantHrefs := []string{"a.html", "/abs/b.html#frag"}
	if !reflect.DeepEqual(hrefs, wantHrefs) {
		t.Fatalf("Extract hrefs = %#v; want %#v", hrefs, wantHrefs)
	}
}

// --- TestCleanHref ---

func TestCleanHref(t *testing.T) {
	base := "http://example.com/base/"
	tests := []struct {
		href string
		want string
	}{
		{"a/b", "http://example.com/base/a/b"},
		{"/x", "http://example.com/x"},
		{"#frag", ""},                       // drop fragments-only
		{"javascript:alert(1)", ""},         // drop js
		{"data:text/plain;base64,AAAA", ""}, // drop data
		{"c.html#sec", "http://example.com/base/c.html"},
	}
	for _, tc := range tests {
		got := CleanHref(base, tc.href)
		if got != tc.want {
			t.Fatalf("CleanHref(%q,%q)=%q; want %q", base, tc.href, got, tc.want)
		}
	}
}

// --- TestDownload ---

func TestDownload(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/ok", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "hello")
	})
	mux.HandleFunc("/fail", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	b, err := Download(srv.URL + "/ok")
	if err != nil {
		t.Fatalf("Download ok error: %v", err)
	}
	if !bytes.Equal(b, []byte("hello")) {
		t.Fatalf("Download body=%q; want %q", string(b), "hello")
	}

	if _, err := Download(srv.URL + "/fail"); err == nil {
		t.Fatalf("Download should error on non-200")
	}
}

// --- TestCrawl ---

func TestCrawl(t *testing.T) {
	// root -> /d1, /d2, /d3, and an off-host link (must be ignored)
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `
			<html><body>
			  <a href="/d1">d1</a>
			  <a href="/d2">d2</a>
			  <a href="/d3">d3</a>
			  <a href="http://example.com/evil">off</a>
			</body></html>`)
	})
	mux.HandleFunc("/d1", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `<html><body><a href="/d2">to d2</a> d1 text</body></html>`)
	})
	mux.HandleFunc("/d2", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `<html><body><a href="/d3">to d3</a> d2 text</body></html>`)
	})
	mux.HandleFunc("/d3", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `<html><body>d3 text</body></html>`)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	start := srv.URL + "/"
	got, err := Crawl(start, 4)
	if err != nil {
		t.Fatalf("Crawl error: %v", err)
	}
	// Expect BFS order: start, /d1, /d2, /d3
	want0 := start
	want1 := srv.URL + "/d1"
	want2 := srv.URL + "/d2"
	want3 := srv.URL + "/d3"
	if len(got) < 4 || got[0] != want0 || got[1] != want1 || got[2] != want2 || got[3] != want3 {
		t.Fatalf("Crawl got=%#v; want prefix [%q %q %q %q]", got, want0, want1, want2, want3)
	}
	// Ensure no off-host URL
	for _, u := range got {
		if strings.Contains(u, "example.com") {
			t.Fatalf("Crawl should not include off-host link: %s", u)
		}
	}
}

// --- helpers for top10-based tests ---

// collectTop10HTMLDocs prefers discovering HTML docs via the student's crawler.
// If the crawl is too shallow (common with tricky relative paths / spaces in folder names),
// it falls back to walking the filesystem and converting paths to HTTP URLs that the
// test server can serve. This keeps the test robust on both local and CI.
func collectTop10HTMLDocs(t *testing.T, srv *httptest.Server) []string {
	t.Helper()

	start := srv.URL + "/"

	// 1) Try crawl first.
	urls, _ := Crawl(start, 1200) // ignore crawl error; we'll fallback if shallow
	var docs []string
	topIndex := strings.ToLower(start + "index.html")
	for _, u := range urls {
		lu := strings.ToLower(u)
		if strings.HasSuffix(lu, ".html") && lu != topIndex {
			docs = append(docs, u)
		}
	}
	if len(docs) >= 50 {
		return docs
	}

	// 2) Fallback: walk the filesystem under ./top10 and convert to URLs.
	var files []string
	err := filepath.WalkDir("top10", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(strings.ToLower(d.Name()), ".html") {
			// Skip the very top-level index.html to reduce noise.
			if strings.EqualFold(d.Name(), "index.html") && filepath.Clean(filepath.Dir(p)) == "top10" {
				return nil
			}
			files = append(files, p)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk top10: %v", err)
	}
	if len(files) == 0 {
		t.Fatalf("no .html files discovered in top10")
	}

	// Build HTTP URLs that map to the test server's root (which serves ./top10).
	// We must escape each path segment (spaces, commas, etc.) for a valid URL.
	pathToURL := func(p string) string {
		rel, _ := filepath.Rel("top10", p)
		segs := strings.Split(rel, string(filepath.Separator))
		for i, s := range segs {
			segs[i] = url.PathEscape(s)
		}
		return srv.URL + "/" + strings.Join(segs, "/")
	}

	docs = docs[:0]
	for _, f := range files {
		docs = append(docs, pathToURL(f))
	}
	return docs
}

// --- TestSearch (use real HTML corpus in ./top10; crawl first, then fallback) ---

func TestSearch(t *testing.T) {
	// Serve the local ./top10 folder over HTTP so Download/Extract see real HTML.
	fs := http.FileServer(http.Dir("top10"))
	srv := httptest.NewServer(fs)
	defer srv.Close()

	// Discover candidate documents (HTML pages) robustly.
	docs := collectTop10HTMLDocs(t, srv)
	if len(docs) < 50 {
		t.Fatalf("expected many .html docs, got %d", len(docs))
	}

	// Build the index from the discovered HTML pages (over HTTP).
	// 使用内存索引器测试
	indexer := NewInMemIndexer(nil)
	defer indexer.Close()

	for _, u := range docs {
		b, err := Download(u)
		if err != nil {
			continue
		}
		words, _ := Extract(b)
		indexer.AddDocument(u, words)
	}

	// Query: "romeo" should surface pages from "Romeo and Juliet".
	hits, err := indexer.Search("romeo")
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(hits) == 0 {
		t.Fatalf("Expected hits for 'romeo', got none")
	}
	top := strings.ToLower(hits[0].URL)
	// Filenames are generic (link2H_*.html), so we check parent-folder keywords.
	if !(strings.Contains(top, "romeo") || strings.Contains(top, "juliet") || strings.Contains(top, "shakespeare")) {
		t.Fatalf("Top hit for 'romeo' should come from Romeo and Juliet corpus, got %s", hits[0].URL)
	}
}

// --- TestStop (stopwords removed; docLen excludes them) ---

func TestStop(t *testing.T) {
	// 使用内存索引器测试
	idx := NewInMemIndexer(nil) // default stopwords
	defer idx.Close()

	text := []byte(`<html><body>whale whale ship and the</body></html>`)
	words, _ := Extract(text)
	idx.AddDocument("doc1", words)

	// Searching stopword -> nil
	hits, err := idx.Search("the")
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if hits != nil {
		t.Fatalf("stopword search should be nil; got %#v", hits)
	}
}

// --- TestTfIdf (ranking sanity on real HTML corpus; crawl first, then fallback) ---

func TestTfIdf(t *testing.T) {
	fs := http.FileServer(http.Dir("top10"))
	srv := httptest.NewServer(fs)
	defer srv.Close()

	// Discover candidate HTML documents with a robust strategy.
	docs := collectTop10HTMLDocs(t, srv)
	if len(docs) == 0 {
		t.Fatalf("No .html documents discovered")
	}

	// 使用内存索引器测试
	indexer := NewInMemIndexer(nil)
	defer indexer.Close()

	for _, u := range docs {
		b, err := Download(u)
		if err != nil {
			continue
		}
		words, _ := Extract(b)
		indexer.AddDocument(u, words)
	}

	// "dracula" should rank Dracula pages highest.
	h1, err := indexer.Search("dracula")
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(h1) == 0 {
		t.Fatalf("Expected hits for 'dracula', got none")
	}
	if !strings.Contains(strings.ToLower(h1[0].URL), "dracula") {
		t.Fatalf("Top hit for 'dracula' should be from Dracula corpus, got %s", h1[0].URL)
	}

	// "frankenstein" should rank Frankenstein pages highest.
	h2, err := indexer.Search("frankenstein")
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(h2) == 0 {
		t.Fatalf("Expected hits for 'frankenstein', got none")
	}
	if !strings.Contains(strings.ToLower(h2[0].URL), "frankenstein") {
		t.Fatalf("Top hit for 'frankenstein' should be from Frankenstein corpus, got %s", h2[0].URL)
	}
}
