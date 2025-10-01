package project02

import (
	"bytes"
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

func Extract(body []byte) ([]string, []string) {
	root, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return nil, nil
	}
	// Match sequences of letters or digits as "words"
	wordRe := regexp.MustCompile(`[\p{L}\p{N}]+`)

	var words []string
	var hrefs []string

	//track a "skip depth" to ignore text under <script> or <style>
	var skipDepth int

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		// Entering a script/style element: increase skip depth.
		if n.Type == html.ElementNode && (strings.EqualFold(n.Data, "script") || strings.EqualFold(n.Data, "style")) {
			skipDepth++
		}

		if skipDepth == 0 {
			// Collect words from text nodes
			if n.Type == html.TextNode {
				for _, tok := range wordRe.FindAllString(n.Data, -1) {
					if tok != "" {
						words = append(words, strings.ToLower(tok))
					}
				}
			}
			// Collect hrefs from <a> elements
			if n.Type == html.ElementNode && strings.EqualFold(n.Data, "a") {
				for _, a := range n.Attr {
					if strings.EqualFold(a.Key, "href") {
						val := strings.TrimSpace(a.Val)
						if val != "" {
							hrefs = append(hrefs, val)
						}
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
		// Leaving a script/style element: decrease skip depth.
		if n.Type == html.ElementNode && (strings.EqualFold(n.Data, "script") || strings.EqualFold(n.Data, "style")) {
			skipDepth--
		}
	}
	walk(root)
	return words, hrefs
}
