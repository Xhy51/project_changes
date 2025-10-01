package project03

import (
	"net/url"
	"strings"
)

func CleanHref(base, href string) string {
	href = strings.TrimSpace(href)
	if href == "" || strings.HasPrefix(href, "#") {
		return ""
	}
	lower := strings.ToLower(href)
	if strings.HasPrefix(lower, "javascript:") || strings.HasPrefix(lower, "data:") {
		return ""
	}

	if !strings.HasSuffix(base, "/") {
		base += "/"
	}
	baseURL, err := url.Parse(base)
	if err != nil {
		return ""
	}

	var refURL *url.URL
	if u, err := url.Parse(href); err == nil {
		refURL = u
	} else {
		refURL = &url.URL{Path: href}
	}

	u := baseURL.ResolveReference(refURL)
	u.Fragment = "" // strip #fragment
	return u.String()
}
