# Project03 - Software Development, Fall 2025

这个项目是基于Project02的升级版本，实现了支持内存和SQLite数据库两种存储方式的搜索引擎。

## 功能特性

- 支持两种数据存储方式：
  1. **内存存储（inmem）**：使用Go的`map[string]map[string]int`存储词频等信息
  2. **SQLite数据库存储（sqlite）**：将数据持久化到`.db`文件中
- 可通过命令行参数切换存储模式
- 保持原有所有测试用例的兼容性
- 使用Go接口抽象存储层，实现"可插拔"的设计

## 项目结构

```
.
├── cmd/                 # 主程序入口
│   └── main.go          # 程序入口点
├── top10/               # 示例HTML文档
├── indexer.go           # 索引器接口和实现
├── search.go            # 搜索相关函数
├── crawl.go             # 爬虫实现
├── download.go          # 下载器实现
├── extract.go           # HTML内容提取器
├── clean.go             # 数据清洗工具
├── stopwords.go         # 停用词处理
├── server.go            # HTTP服务器实现
├── project02_test.go    # 测试用例
├── sqlite_test.go       # SQLite相关测试（需要CGO支持）
├── go.mod               # Go模块定义
├── go.sum               # Go依赖校验和
├── .gitignore           # Git忽略文件
└── README.md            # 项目说明文档
```

## 安装和运行

### 依赖

- Go 1.16或更高版本
- CGO支持（用于SQLite）

### 构建项目

```bash
go build -o project03 ./cmd
```

### 运行项目

使用内存存储（默认）：
```bash
go run ./cmd -index=inmem
```

使用SQLite数据库存储：
```bash
go run ./cmd -index=sqlite
```

指定数据库文件路径：
```bash
go run ./cmd -index=sqlite -db=myindex.db
```

### 运行测试

```bash
# 运行基本测试 (需要先运行上面的命令)
go test -v

# 运行包含SQLite的测试（需要CGO支持）
go test -v -tags cgo
```

## API接口

启动服务器后，可以通过以下接口访问：

- `http://localhost:8080/` - 重定向到 `/top10/`
- `http://localhost:8080/top10/` - 访问示例HTML文档
- `http://localhost:8080/search?q=term` - 搜索关键词

## 设计说明

### 接口抽象

通过定义`Indexer`接口，实现了存储层的抽象：

```go
type Indexer interface {
    AddDocument(url string, words []string) error
    Search(query string) ([]Hit, error)
    Close() error
}
```

### 两种实现

1. **InMemIndexer**：基于内存的实现，与Project02兼容
2. **SQLiteIndexer**：基于SQLite数据库的实现，支持数据持久化

### 数据库设计

SQLite数据库包含以下表：

```sql
-- URLs表
CREATE TABLE urls (
    id INTEGER PRIMARY KEY,
    name TEXT UNIQUE NOT NULL
);

-- 词汇表
CREATE TABLE words (
    id INTEGER PRIMARY KEY,
    word TEXT UNIQUE NOT NULL
);

-- 命中记录表
CREATE TABLE hits (
    url_id INTEGER,
    word_id INTEGER,
    count INTEGER,
    PRIMARY KEY (url_id, word_id),
    FOREIGN KEY (url_id) REFERENCES urls(id),
    FOREIGN KEY (word_id) REFERENCES words(id)
);
```

## 性能优化

- 为`hits.word_id`列创建索引以提高查询性能
- 使用预处理语句防止SQL注入
- 使用事务批量处理数据插入

## 注意事项

- SQLite实现需要CGO支持
- 数据库文件会自动创建和更新
- 通过`.gitignore`忽略数据库文件，避免提交到版本控制