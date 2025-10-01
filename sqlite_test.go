//go:build cgo
// +build cgo

package project03

import (
	_ "strings"
	"testing"
)

// TestSQLiteIndexer 测试SQLite索引器实现
func TestSQLiteIndexer(t *testing.T) {
	// 创建临时SQLite索引器
	indexer, err := NewSQLiteIndexer(":memory:", nil)
	if err != nil {
		t.Fatalf("Failed to create SQLite indexer: %v", err)
	}
	defer indexer.Close()

	// 添加测试文档
	text := []byte(`<html><body>whale whale ship and the</body></html>`)
	words, _ := Extract(text)
	err = indexer.AddDocument("doc1", words)
	if err != nil {
		t.Fatalf("AddDocument error: %v", err)
	}

	// 搜索测试
	hits, err := indexer.Search("whale")
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(hits) == 0 {
		t.Fatalf("Expected hits for 'whale', got none")
	}

	// 搜索停用词应该返回空结果
	hits, err = indexer.Search("the")
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if hits != nil {
		t.Fatalf("stopword search should be nil; got %#v", hits)
	}
}
