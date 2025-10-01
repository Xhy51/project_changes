package project03

import (
	"net/url"
	"strings"
)

func Crawl(start string, max int) ([]string, error) {
	if max <= 0 {
		return []string{}, nil
	}

	startURL, err := url.Parse(start)
	if err != nil {
		return nil, err
	}
	// Used only for "same host" check; do NOT use this as the base for resolving links.
	hostBase := startURL.Scheme + "://" + startURL.Host + "/"

	visited := make(map[string]bool)
	queue := []string{start}
	order := make([]string, 0, max)

	for len(queue) > 0 && len(order) < max {
		// FIFO queue → BFS
		cur := queue[0]
		queue = queue[1:]

		if visited[cur] {
			continue
		}
		visited[cur] = true
		order = append(order, cur)

		// Download the current page
		body, err := Download(cur)
		if err != nil {
			// Skip transient errors; keep crawling the rest
			continue
		}

		// Extract words/links from the page
		_, hrefs := Extract(body)
		for _, h := range hrefs {

			abs := CleanHref(cur, h)
			if abs == "" {
				continue
			}

			if !strings.HasPrefix(abs, hostBase) {
				continue
			}

			if !visited[abs] {
				queue = append(queue, abs)
			}
		}
	}
	return order, nil
}
