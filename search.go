package project03

// BuildIndexFromURLList downloads and indexes a list of URLs.
// BuildIndexFromURLList下载并索引URL列表
func BuildIndexFromURLList(urls []string, indexer Indexer) error {
	for _, u := range urls {
		b, err := Download(u)
		if err != nil {
			continue
		}
		words, _ := Extract(b)
		indexer.AddDocument(u, words)
	}
	return nil
}
