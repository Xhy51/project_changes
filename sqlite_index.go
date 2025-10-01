package project02

import (
	"database/sql"
	"math"
	"sort"
	"strings"

	_ "github.com/glebarez/sqlite"
	"github.com/kljensen/snowball/english"
)

// SQLiteIndex stores data for TF-IDF ranking in SQLite database.
type SQLiteIndex struct {
	db   *sql.DB
	stop map[string]struct{}
	N    int
}

// NewSQLiteIndex creates a new SQLite index
func NewSQLiteIndex(dbPath string, stop map[string]struct{}) (*SQLiteIndex, error) {
	if stop == nil {
		stop = DefaultStopwords()
	}

	// Open SQLite database
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	// Create tables if they don't exist
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS urls (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			url TEXT UNIQUE NOT NULL,
			len INTEGER
		);
		
		CREATE TABLE IF NOT EXISTS terms (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			word TEXT UNIQUE NOT NULL,
			df INTEGER
		);
		
		CREATE TABLE IF NOT EXISTS hits (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			term_id INTEGER,
			url_id INTEGER,
			count INTEGER,
			FOREIGN KEY(term_id) REFERENCES terms(id),
			FOREIGN KEY(url_id) REFERENCES urls(id)
		);
		
		CREATE INDEX IF NOT EXISTS idx_hits_term_id ON hits(term_id);
		CREATE INDEX IF NOT EXISTS idx_hits_url_id ON hits(url_id);
		
		PRAGMA foreign_keys = ON;
	`)
	if err != nil {
		db.Close()
		return nil, err
	}

	idx := &SQLiteIndex{
		db:   db,
		stop: stop,
	}

	// Get the total number of documents
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM urls").Scan(&count)
	if err != nil {
		db.Close()
		return nil, err
	}
	idx.N = count

	return idx, nil
}

// Add indexes a single document. Pipeline: lower -> stop filter -> stem.
func (idx *SQLiteIndex) Add(doc string, words []string) {
	// Check if document already exists
	var existingID int
	err := idx.db.QueryRow("SELECT id FROM urls WHERE url = ?", doc).Scan(&existingID)
	if err == nil {
		// Document already exists
		return
	}

	seen := make(map[string]bool)
	var kept int

	// Create URL record first
	result, err := idx.db.Exec("INSERT INTO urls (url, len) VALUES (?, 0)", doc)
	if err != nil {
		return
	}
	urlID, err := result.LastInsertId()
	if err != nil {
		return
	}

	// Process words
	for _, w := range words {
		if w == "" {
			continue
		}
		lw := strings.ToLower(w)
		if _, bad := idx.stop[lw]; bad {
			continue
		}
		s := english.Stem(lw, true)
		if s == "" {
			continue
		}
		kept++

		// Get or create term
		var termID int64
		err = idx.db.QueryRow("SELECT id FROM terms WHERE word = ?", s).Scan(&termID)
		if err == sql.ErrNoRows {
			// Term doesn't exist, create it with df=1
			result, err := idx.db.Exec("INSERT INTO terms (word, df) VALUES (?, 1)", s)
			if err != nil {
				continue
			}
			termID, err = result.LastInsertId()
			if err != nil {
				continue
			}
			seen[s] = true
		} else if err != nil {
			// Other database error
			continue
		} else {
			// Term exists, increment document frequency if this is the first time we see this term in this document
			if !seen[s] {
				_, err := idx.db.Exec("UPDATE terms SET df = df + 1 WHERE word = ?", s)
				if err != nil {
					continue
				}
				seen[s] = true
			}
		}

		// Create or update hit
		var hitID int64
		err = idx.db.QueryRow("SELECT id FROM hits WHERE term_id = ? AND url_id = ?", termID, urlID).Scan(&hitID)
		if err == sql.ErrNoRows {
			// Hit doesn't exist, create it
			_, err := idx.db.Exec("INSERT INTO hits (term_id, url_id, count) VALUES (?, ?, 1)", termID, urlID)
			if err != nil {
				continue
			}
		} else if err == nil {
			// Hit exists, increment count
			_, err := idx.db.Exec("UPDATE hits SET count = count + 1 WHERE term_id = ? AND url_id = ?", termID, urlID)
			if err != nil {
				continue
			}
		}
	}

	// Update URL length
	_, err = idx.db.Exec("UPDATE urls SET len = ? WHERE id = ?", kept, urlID)
	if err != nil {
		return
	}

	// Update document count
	idx.N++
}

// GetN returns the total number of documents
func (idx *SQLiteIndex) GetN() int {
	// Refresh N from database to ensure consistency
	var count int
	err := idx.db.QueryRow("SELECT COUNT(*) FROM urls").Scan(&count)
	if err != nil {
		return idx.N
	}
	idx.N = count
	return idx.N
}

// SearchTFIDF ranks a single-term query using TF-IDF.
func (idx *SQLiteIndex) SearchTFIDF(term string) []Hit {
	if term == "" || idx.N == 0 {
		return nil
	}
	q := strings.ToLower(term)
	if _, bad := idx.stop[q]; bad {
		return nil
	}
	s := english.Stem(q, true)

	// Find the term
	var termID int
	var df int
	err := idx.db.QueryRow("SELECT id, df FROM terms WHERE word = ?", s).Scan(&termID, &df)
	if err != nil {
		return nil
	}

	// Calculate IDF
	idf := math.Log(float64(idx.N) / float64(df))

	// Get hits for this term
	rows, err := idx.db.Query(`
		SELECT h.count, u.url, u.len 
		FROM hits h 
		JOIN urls u ON h.url_id = u.id 
		WHERE h.term_id = ?`, termID)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var hits []Hit
	for rows.Next() {
		var count, docLen int
		var url string
		err := rows.Scan(&count, &url, &docLen)
		if err != nil {
			continue
		}

		if docLen > 0 {
			tf := float64(count) / float64(docLen)
			hits = append(hits, Hit{URL: url, Score: tf * idf})
		}
	}

	// Sort hits by score (descending) and URL (ascending) for ties
	sort.Slice(hits, func(i, j int) bool {
		return lessHit(hits[i], hits[j])
	})

	return hits
}

// Close closes the database connection
func (idx *SQLiteIndex) Close() error {
	return idx.db.Close()
}
