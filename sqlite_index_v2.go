package project02

import (
	"database/sql"
	"math"
	"sort"
	"strings"

	_ "github.com/glebarez/sqlite"
	"github.com/kljensen/snowball/english"
)

// SQLiteIndexV2 是基于SQLite数据库的索引器实现的另一个版本
type SQLiteIndexV2 struct {
	db   *sql.DB
	stop map[string]struct{}
	N    int
}

// NewSQLiteIndexV2 创建一个新的SQLite索引器V2版本
func NewSQLiteIndexV2(dbPath string, stop map[string]struct{}) (*SQLiteIndexV2, error) {
	if stop == nil {
		stop = DefaultStopwords()
	}

	// Open SQLite database
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	// Enable foreign key constraints
	_, err = db.Exec("PRAGMA foreign_keys = ON")
	if err != nil {
		db.Close()
		return nil, err
	}

	// Create tables with a different schema structure
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS documents (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			url TEXT UNIQUE NOT NULL,
			word_count INTEGER DEFAULT 0
		);

		CREATE TABLE IF NOT EXISTS vocabulary (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			term TEXT UNIQUE NOT NULL,
			document_frequency INTEGER DEFAULT 0
		);

		CREATE TABLE IF NOT EXISTS term_frequencies (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			doc_id INTEGER NOT NULL,
			term_id INTEGER NOT NULL,
			frequency INTEGER DEFAULT 0,
			FOREIGN KEY (doc_id) REFERENCES documents(id) ON DELETE CASCADE,
			FOREIGN KEY (term_id) REFERENCES vocabulary(id) ON DELETE CASCADE,
			UNIQUE(doc_id, term_id)
		);

		CREATE INDEX IF NOT EXISTS idx_documents_url ON documents(url);
		CREATE INDEX IF NOT EXISTS idx_vocabulary_term ON vocabulary(term);
		CREATE INDEX IF NOT EXISTS idx_term_frequencies_doc ON term_frequencies(doc_id);
		CREATE INDEX IF NOT EXISTS idx_term_frequencies_term ON term_frequencies(term_id);
	`)
	if err != nil {
		db.Close()
		return nil, err
	}

	idx := &SQLiteIndexV2{
		db:   db,
		stop: stop,
	}

	// Get the total number of documents
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM documents").Scan(&count)
	if err != nil {
		db.Close()
		return nil, err
	}
	idx.N = count

	return idx, nil
}

// Add 将文档添加到索引中，使用不同的处理逻辑
func (idx *SQLiteIndexV2) Add(doc string, words []string) {
	// Start a transaction for better performance and consistency
	tx, err := idx.db.Begin()
	if err != nil {
		return
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		} else {
			tx.Commit()
		}
	}()

	// Check if document already exists
	var docID int64
	err = tx.QueryRow("SELECT id FROM documents WHERE url = ?", doc).Scan(&docID)
	if err == nil {
		// Document already exists, nothing to do
		return
	}

	// Create document record
	result, err := tx.Exec("INSERT INTO documents (url, word_count) VALUES (?, 0)", doc)
	if err != nil {
		return
	}
	docID, err = result.LastInsertId()
	if err != nil {
		return
	}

	// Process words with a different approach
	termFreq := make(map[string]int)
	uniqueTerms := make(map[string]bool)

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
		termFreq[s]++
		uniqueTerms[s] = true
	}

	// Update document word count
	_, err = tx.Exec("UPDATE documents SET word_count = ? WHERE id = ?", len(words), docID)
	if err != nil {
		return
	}

	// Process each unique term
	for term := range uniqueTerms {
		// Get or create term
		var termID int64
		err = tx.QueryRow("SELECT id FROM vocabulary WHERE term = ?", term).Scan(&termID)
		if err == sql.ErrNoRows {
			// Term doesn't exist, create it
			result, err := tx.Exec("INSERT INTO vocabulary (term, document_frequency) VALUES (?, 1)", term)
			if err != nil {
				continue
			}
			termID, err = result.LastInsertId()
			if err != nil {
				continue
			}
		} else if err != nil {
			continue
		} else {
			// Term exists, increment document frequency
			_, err := tx.Exec("UPDATE vocabulary SET document_frequency = document_frequency + 1 WHERE term = ?", term)
			if err != nil {
				continue
			}
		}

		// Insert or update term frequency
		_, err = tx.Exec(`
			INSERT INTO term_frequencies (doc_id, term_id, frequency) 
			VALUES (?, ?, ?)
			ON CONFLICT(doc_id, term_id) 
			DO UPDATE SET frequency = ?`,
			docID, termID, termFreq[term], termFreq[term])
		if err != nil {
			continue
		}
	}

	// Update document count
	idx.N++
}

// SearchTFIDF 使用TF-IDF算法搜索文档，采用不同的查询方式
func (idx *SQLiteIndexV2) SearchTFIDF(term string) []Hit {
	if term == "" || idx.N == 0 {
		return nil
	}

	q := strings.ToLower(term)
	if _, bad := idx.stop[q]; bad {
		return nil
	}
	s := english.Stem(q, true)

	// Use a single query to get all necessary data
	query := `
		SELECT d.url, tf.frequency, d.word_count, v.document_frequency
		FROM vocabulary v
		JOIN term_frequencies tf ON v.id = tf.term_id
		JOIN documents d ON tf.doc_id = d.id
		WHERE v.term = ?`

	rows, err := idx.db.Query(query, s)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var hits []Hit
	for rows.Next() {
		var url string
		var frequency, wordCount, docFreq int
		err := rows.Scan(&url, &frequency, &wordCount, &docFreq)
		if err != nil {
			continue
		}

		if wordCount > 0 && docFreq > 0 {
			// Calculate TF-IDF
			tf := float64(frequency) / float64(wordCount)
			idf := math.Log(float64(idx.N) / float64(docFreq))
			score := tf * idf
			hits = append(hits, Hit{URL: url, Score: score})
		}
	}

	// Check for errors from iterating over rows
	if err = rows.Err(); err != nil {
		return nil
	}

	// Sort hits by score (descending) and URL (ascending) for ties
	sort.Slice(hits, func(i, j int) bool {
		return lessHit(hits[i], hits[j])
	})

	return hits
}

// GetN 返回文档总数
func (idx *SQLiteIndexV2) GetN() int {
	return idx.N
}

// Close 关闭数据库连接
func (idx *SQLiteIndexV2) Close() error {
	return idx.db.Close()
}
