
---

# 📄 `few_shot_prompts.md`

````markdown
# Few-Shot Prompts for Project03

These prompts were used to guide the LLM to produce more consistent and correct Go code for Project03.  
Each example contains a **Prompt** (input to the model) and the **Expected Response** (desired output).

---

### Example 1

**Prompt:**
Write a Go interface that abstracts both an in-memory map of maps and a persistent SQLite database for indexing URLs. The interface should include methods for `Insert`, `Search`, and `Delete`.

**Expected Response:**
```go
type Indexer interface {
    Insert(url string, content string) error
    Search(query string) ([]string, error)
    Delete(url string) error
}
````

---

### Example 2

**Prompt:**
Show me how to safely insert a record into SQLite in Go without SQL injection, using `database/sql`.

**Expected Response:**

```go
stmt, err := db.Prepare("INSERT INTO urls(name) VALUES(?)")
if err != nil {
    return err
}
_, err = stmt.Exec(urlName)
return err
```

---

### Example 3

**Prompt:**
Write Go code to switch between an in-memory index and a SQLite index using a command-line flag `-index`.

**Expected Response:**

```go
var indexType = flag.String("index", "inmem", "choose index type: inmem or sqlite")
flag.Parse()

var idx Indexer
if *indexType == "sqlite" {
    idx = NewSQLiteIndexer("data.db")
} else {
    idx = NewInMemIndexer()
}
```

---

### Example 4

**Prompt:**
Give me a `.gitignore` example that ignores SQLite database files.

**Expected Response:**

```
# Ignore SQLite database
*.db
*.sqlite
```

---

### Example 5

**Prompt:**
Write a SQL query in SQLite to join `urls` and `hits` tables so that I can see each URL and its hit count.

**Expected Response:**

```sql
SELECT urls.name, hits.count
FROM urls
JOIN hits ON urls.id = hits.url_id;
```