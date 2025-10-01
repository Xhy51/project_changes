package project03

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"

	// https://github.com/mattn/go-sqlite3
	_ "modernc.org/sqlite"
)

// 定义命令行参数
// Define command line flags
var (
	indexMode = flag.String("index", "inmem", "index backend: inmem or sqlite")
	dbPath    = flag.String("db", "index.db", "path to SQLite database file")
)

// NewMux serves ./top10 at /top10/ and provides /search?q=term.
// Library-only: does not start the server by itself.
// NewMux提供./top10目录服务和/search?q=term搜索接口
func NewMux(indexer Indexer) http.Handler {
	mux := http.NewServeMux()

	// Redirect root to /top10/
	// 重定向根路径到/top10/
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.Redirect(w, r, "/top10/", http.StatusFound)
			return
		}
		http.NotFound(w, r)
	})

	// Map /top10/* -> ./top10/*
	// 映射/top10/*到./top10/*
	mux.Handle("/top10/", http.StripPrefix("/top10/",
		http.FileServer(http.Dir("./top10"))))

	// /search?q=term -> JSON hits
	// /search?q=term返回JSON格式的搜索结果
	mux.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("q")
		hits, err := indexer.Search(q)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(hits)
	})

	return mux
}

// Run starts the server with the specified indexer
// Run使用指定的索引器启动服务器
func Run() {
	flag.Parse()

	var indexer Indexer
	var err error

	// 根据命令行参数选择索引器实现
	// Choose indexer implementation based on command line flag
	switch *indexMode {
	case "inmem":
		indexer = NewInMemIndexer(nil)
		log.Println("Using in-memory indexer")
	case "sqlite":
		indexer, err = NewSQLiteIndexer(*dbPath, nil)
		if err != nil {
			log.Fatalf("Failed to create SQLite indexer: %v", err)
		}
		log.Printf("Using SQLite indexer with database: %s", *dbPath)
	default:
		log.Fatalf("Unknown index mode: %s. Use 'inmem' or 'sqlite'", *indexMode)
	}

	// 确保在程序退出时关闭索引器
	// Ensure indexer is closed when program exits
	defer indexer.Close()

	// 爬取top10目录中的HTML文件并建立索引
	// Crawl HTML files in top10 directory and build index
	urls, err := Crawl("http://localhost:8080/top10/", 1000)
	if err != nil {
		log.Fatalf("Failed to crawl: %v", err)
	}

	err = BuildIndexFromURLList(urls, indexer)
	if err != nil {
		log.Fatalf("Failed to build index: %v", err)
	}

	// 创建HTTP处理器
	// Create HTTP handler
	handler := NewMux(indexer)

	// 启动服务器
	// Start server
	fmt.Println("Server starting on :8080")
	log.Fatal(http.ListenAndServe(":8080", handler))
}
