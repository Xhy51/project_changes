# Project03 - Software Development, Fall 2025

This project is an upgraded version of Project02, implementing a search engine that supports both in-memory and SQLite database storage.

## Features

- Supports two data storage methods:
  1. **In-memory storage (inmem)**: Uses Go's `map[string]map[string]int` to store term frequencies and other information
  2. **SQLite database storage (sqlite)**: Persists data to `.db` files
- Storage mode can be switched via command-line arguments
- Maintains compatibility with all existing test cases
- Uses Go interface abstraction for storage layer, implementing a "pluggable" design

## Project Structure

```
.
├── cmd/                 # Main program entry
│   └── main.go          # Program entry point
├── top10/               # Sample HTML documents
├── indexer.go           # Indexer interface and implementations
├── search.go            # Search related functions
├── crawl.go             # Crawler implementation
├── download.go          # Downloader implementation
├── extract.go           # HTML content extractor
├── clean.go             # Data cleaning tools
├── stopwords.go         # Stop words processing
├── server.go            # HTTP server implementation
├── project02_test.go    # Test cases
├── sqlite_test.go       # SQLite related tests (requires CGO support)
├── go.mod               # Go module definition
├── go.sum               # Go dependency checksums
├── .gitignore           # Git ignore file
└── README.md            # Project documentation
```

## Installation and Running

### Dependencies

- Go 1.16 or higher
- CGO support (for SQLite)

### Build Project

```bash
go build -o project03 ./cmd
```

### Run Project

Use in-memory storage (default):
```bash
go run ./cmd -index=inmem
```

Use SQLite database storage:
```bash
go run ./cmd -index=sqlite
```

Specify database file path:
```bash
go run ./cmd -index=sqlite -db=myindex.db
```

### Run Tests

```bash
# Run basic tests
go test -v

# Run tests including SQLite (requires CGO support)
go test -v -tags cgo
```

## API Interface

After starting the server, you can access the following interfaces:

- `http://localhost:8080/` - Redirects to `/top10/`
- `http://localhost:8080/top10/` - Access sample HTML documents
- `http://localhost:8080/search?q=term` - Search for keywords

## Design Documentation

### Interface Abstraction

Storage layer abstraction is implemented by defining the `Indexer` interface:

```go
type Indexer interface {
    AddDocument(url string, words []string) error
    Search(query string) ([]Hit, error)
    Close() error
}
```

### Two Implementations

1. **InMemIndexer**: In-memory implementation, compatible with Project02
2. **SQLiteIndexer**: SQLite database implementation, supporting data persistence

### Database Design

The SQLite database contains the following tables:

```sql
-- URLs table
CREATE TABLE urls (
    id INTEGER PRIMARY KEY,
    name TEXT UNIQUE NOT NULL
);

-- Words table
CREATE TABLE words (
    id INTEGER PRIMARY KEY,
    word TEXT UNIQUE NOT NULL
);

-- Hits table
CREATE TABLE hits (
    url_id INTEGER,
    word_id INTEGER,
    count INTEGER,
    PRIMARY KEY (url_id, word_id),
    FOREIGN KEY (url_id) REFERENCES urls(id),
    FOREIGN KEY (word_id) REFERENCES words(id)
);
```

## Performance Optimization

- Create indexes on `hits.word_id` column to improve query performance
- Use prepared statements to prevent SQL injection
- Use transactions for batch data processing

## Notes

- SQLite implementation requires CGO support
- Database files are automatically created and updated
- Database files are ignored via `.gitignore` to avoid committing to version control