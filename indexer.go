package project03

import (
	"database/sql"
	"math"
	"sort"
	"strings"

	"github.com/kljensen/snowball/english"
	// https://github.com/mattn/go-sqlite3
	_ "modernc.org/sqlite"
)

// Hit is a scored search result.
type Hit struct {
	URL   string
	Score float64
}

// Indexer 接口定义了索引器需要实现的方法
// Indexer interface defines the methods that an indexer must implement
type Indexer interface {
	// AddDocument 添加文档到索引中
	// AddDocument adds a document to the index
	AddDocument(url string, words []string) error

	// Search 根据查询词搜索文档
	// Search searches for documents based on query term
	Search(query string) ([]Hit, error)

	// Close 关闭索引器，释放资源
	// Close closes the indexer and releases resources
	Close() error
}

// InMemIndexer 是基于内存的索引器实现
// InMemIndexer is an in-memory implementation of the Indexer interface
type InMemIndexer struct {
	tf     map[string]map[string]int // stem -> doc -> term freq
	df     map[string]int            // stem -> doc freq
	docLen map[string]int            // doc -> token count (after stop+stem)
	N      int                       // total documents
	stop   map[string]struct{}       // stopword set
}

// NewInMemIndexer 创建一个新的内存索引器
// NewInMemIndexer creates a new in-memory indexer
func NewInMemIndexer(stop map[string]struct{}) *InMemIndexer {
	if stop == nil {
		stop = DefaultStopwords()
	}
	return &InMemIndexer{
		tf:     make(map[string]map[string]int),
		df:     make(map[string]int),
		docLen: make(map[string]int),
		stop:   stop,
	}
}

// internal stemmer
func stem(w string) string { return english.Stem(w, true) }

// AddDocument 实现Indexer接口的AddDocument方法
// AddDocument implements the AddDocument method of the Indexer interface
func (idx *InMemIndexer) AddDocument(doc string, words []string) error {
	if _, dup := idx.docLen[doc]; dup {
		return nil
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
	return nil
}

// --- extracted comparator (outside Search) ---

// lessHit orders two hits: higher score first; if scores are equal, URL ascending.
// NOTE: `hits[j].URL > hits[i].URL` is equivalent to `hits[i].URL < hits[j].URL`.
func lessHit(a, b Hit) bool {
	if a.Score != b.Score {
		return a.Score > b.Score
	}
	return b.URL > a.URL
}

// Search 实现Indexer接口的Search方法
// Search implements the Search method of the Indexer interface
func (idx *InMemIndexer) Search(term string) ([]Hit, error) {
	if term == "" || idx.N == 0 {
		return nil, nil
	}
	q := strings.ToLower(term)
	if _, bad := idx.stop[q]; bad {
		return nil, nil
	}
	s := stem(q)
	df := idx.df[s]
	if df == 0 {
		return nil, nil
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
	return hits, nil
}

// Close 实现Indexer接口的Close方法
// Close implements the Close method of the Indexer interface
func (idx *InMemIndexer) Close() error {
	// 内存索引器不需要特殊关闭操作
	// In-memory indexer doesn't need special closing operations
	return nil
}

// SQLiteIndexer 是基于SQLite数据库的索引器实现
// SQLiteIndexer is a SQLite database-based implementation of the Indexer interface
type SQLiteIndexer struct {
	db   *sql.DB
	stop map[string]struct{} // stopword set
}

// NewSQLiteIndexer 创建一个新的SQLite索引器
// NewSQLiteIndexer creates a new SQLite indexer
func NewSQLiteIndexer(dbPath string, stop map[string]struct{}) (*SQLiteIndexer, error) {
	if stop == nil {
		stop = DefaultStopwords()
	}

	// 连接到SQLite数据库
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	// 设置连接池
	db.SetMaxOpenConns(1)

	// 创建表结构
	err = createTables(db)
	if err != nil {
		db.Close()
		return nil, err
	}

	// 开启外键支持
	_, err = db.Exec("PRAGMA foreign_keys = ON;")
	if err != nil {
		db.Close()
		return nil, err
	}

	return &SQLiteIndexer{
		db:   db,
		stop: stop,
	}, nil
}

// createTables 创建数据库表
// createTables creates database tables
func createTables(db *sql.DB) error {
	// 创建urls表
	_, err := db.Exec(`
	CREATE TABLE IF NOT EXISTS urls (
		id INTEGER PRIMARY KEY,
		name TEXT UNIQUE NOT NULL
	)`)
	if err != nil {
		return err
	}

	// 创建words表
	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS words (
		id INTEGER PRIMARY KEY,
		word TEXT UNIQUE NOT NULL
	)`)
	if err != nil {
		return err
	}

	// 创建hits表
	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS hits (
		url_id INTEGER,
		word_id INTEGER,
		count INTEGER,
		PRIMARY KEY (url_id, word_id),
		FOREIGN KEY (url_id) REFERENCES urls(id),
		FOREIGN KEY (word_id) REFERENCES words(id)
	)`)
	if err != nil {
		return err
	}

	// 为hits表的word_id列创建索引以提高查询性能
	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_hits_word ON hits(word_id)`)
	return err
}

// AddDocument 实现Indexer接口的AddDocument方法
// AddDocument implements the AddDocument method of the Indexer interface
func (idx *SQLiteIndexer) AddDocument(url string, words []string) error {
	// 开始事务
	tx, err := idx.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 插入URL
	urlStmt, err := tx.Prepare("INSERT OR IGNORE INTO urls (name) VALUES (?)")
	if err != nil {
		return err
	}
	defer urlStmt.Close()

	_, err = urlStmt.Exec(url)
	if err != nil {
		return err
	}

	// 获取URL ID
	var urlID int
	err = tx.QueryRow("SELECT id FROM urls WHERE name = ?", url).Scan(&urlID)
	if err != nil {
		return err
	}

	// 统计词频
	wordCount := make(map[string]int)
	seen := make(map[string]bool)

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
		wordCount[s]++
		seen[s] = true
	}

	// 插入词和词频
	wordStmt, err := tx.Prepare("INSERT OR IGNORE INTO words (word) VALUES (?)")
	if err != nil {
		return err
	}
	defer wordStmt.Close()

	hitStmt, err := tx.Prepare("INSERT OR REPLACE INTO hits (url_id, word_id, count) VALUES (?, (SELECT id FROM words WHERE word = ?), ?)")
	if err != nil {
		return err
	}
	defer hitStmt.Close()

	for word, count := range wordCount {
		// 插入词
		_, err = wordStmt.Exec(word)
		if err != nil {
			return err
		}

		// 插入词频
		_, err = hitStmt.Exec(urlID, word, count)
		if err != nil {
			return err
		}
	}

	// 提交事务
	return tx.Commit()
}

// Search 实现Indexer接口的Search方法
// Search implements the Search method of the Indexer interface
func (idx *SQLiteIndexer) Search(term string) ([]Hit, error) {
	if term == "" {
		return nil, nil
	}

	q := strings.ToLower(term)
	if _, bad := idx.stop[q]; bad {
		return nil, nil
	}
	s := stem(q)

	// 获取文档总数
	var totalDocs int
	err := idx.db.QueryRow("SELECT COUNT(*) FROM urls").Scan(&totalDocs)
	if err != nil {
		return nil, err
	}

	if totalDocs == 0 {
		return nil, nil
	}

	// 获取词的文档频率
	var df int
	err = idx.db.QueryRow("SELECT COUNT(*) FROM hits h JOIN words w ON h.word_id = w.id WHERE w.word = ?", s).Scan(&df)
	if err != nil {
		return nil, err
	}

	if df == 0 {
		return nil, nil
	}

	// 计算IDF
	idf := math.Log(float64(totalDocs) / float64(df))

	// 查询匹配的文档和词频
	rows, err := idx.db.Query(`
		SELECT u.name, h.count, 
		       (SELECT SUM(h2.count) FROM hits h2 WHERE h2.url_id = u.id) as total_words
		FROM urls u
		JOIN hits h ON u.id = h.url_id
		JOIN words w ON w.id = h.word_id
		WHERE w.word = ?`, s)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hits []Hit
	for rows.Next() {
		var url string
		var count, totalWords int
		err := rows.Scan(&url, &count, &totalWords)
		if err != nil {
			return nil, err
		}

		if totalWords > 0 {
			tf := float64(count) / float64(totalWords)
			score := tf * idf
			hits = append(hits, Hit{URL: url, Score: score})
		}
	}

	// 检查是否有迭代错误
	if err = rows.Err(); err != nil {
		return nil, err
	}

	// 按分数排序
	sort.Slice(hits, func(i, j int) bool {
		return lessHit(hits[i], hits[j])
	})

	return hits, nil
}

// Close 实现Indexer接口的Close方法
// Close implements the Close method of the Indexer interface
func (idx *SQLiteIndexer) Close() error {
	return idx.db.Close()
}
