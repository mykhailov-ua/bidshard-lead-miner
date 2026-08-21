package forum

import (
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/html"
)

var (
	postBodyRe    = regexp.MustCompile(`(?is)<div[^>]*class="[^"]*postbody[^"]*"[^>]*>(.*?)</div>`)
	xenforoBodyRe = regexp.MustCompile(`(?is)<div[^>]*class="[^"]*bbWrapper[^"]*"[^>]*>(.*?)</div>`)
	usernameRe    = regexp.MustCompile(`(?is)<div[^>]*class="[^"]*username[^"]*"[^>]*>(.*?)</div>`)
	xenforoUserRe = regexp.MustCompile(`(?is)<a[^>]*class="[^"]*username[^"]*"[^>]*>(.*?)</a>`)
	datetimeAttr  = regexp.MustCompile(`(?i)datetime="([^"]+)"`)
	dateTextRe    = regexp.MustCompile(`(?i)\b(\d{4}-\d{2}-\d{2}|\d{1,2}\s+[A-Za-z]{3}\s+\d{4})\b`)
	threadLinkRe  = regexp.MustCompile(`(?i)href="([^"]*(?:/threads/|/t/|thread-|\?t=)[^"]*)"`)
	paginationRe  = regexp.MustCompile(`(?i)href="([^"]*(?:/page-\d+|[?&]page=\d+|[?&]start=\d+)[^"]*)"`)
	stripTagRe    = regexp.MustCompile(`(?is)<[^>]+>`)
)

type Post struct {
	Author   string
	Body     string
	PostedAt time.Time
}

var painSignals = []string{
	"voluum alternative",
	"postback failing",
	"tracker too expensive",
	"keitaro alternative",
	"self-hosted tracker",
	"missing ftd",
	"postback",
}

func ParsePostsFromHTML(raw string) []Post {
	if posts := parseWithRegex(raw); len(posts) > 0 {
		return posts
	}
	return parseWithTokenizer(raw)
}

// ParseThreadURLsFromCategory extracts thread URLs from forum category/index pages.
func ParseThreadURLsFromCategory(html string) []string {
	matches := threadLinkRe.FindAllStringSubmatch(html, -1)
	if len(matches) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(matches))
	var urls []string
	for _, m := range matches {
		link := m[1]
		if _, exists := seen[link]; !exists {
			seen[link] = struct{}{}
			urls = append(urls, link)
		}
	}
	return urls
}

// ParsePaginationLinks extracts page navigation URLs from HTML.
func ParsePaginationLinks(html string) []string {
	matches := paginationRe.FindAllStringSubmatch(html, -1)
	if len(matches) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(matches))
	var urls []string
	for _, m := range matches {
		link := m[1]
		if _, exists := seen[link]; !exists {
			seen[link] = struct{}{}
			urls = append(urls, link)
		}
	}
	return urls
}

func parseWithRegex(raw string) []Post {
	bodies := postBodyRe.FindAllStringSubmatch(raw, -1)
	if len(bodies) == 0 {
		bodies = xenforoBodyRe.FindAllStringSubmatch(raw, -1)
	}
	if len(bodies) == 0 {
		return nil
	}
	users := usernameRe.FindAllStringSubmatch(raw, -1)
	if len(users) == 0 {
		users = xenforoUserRe.FindAllStringSubmatch(raw, -1)
	}
	postDate := parseDateFromRaw(raw)

	var posts []Post
	for i, match := range bodies {
		body := stripTags(match[1])
		author := ""
		if i < len(users) {
			author = stripTags(users[i][1])
		}
		if body == "" {
			continue
		}
		posts = append(posts, Post{Author: author, Body: body, PostedAt: postDate})
	}
	return posts
}

func parseWithTokenizer(raw string) []Post {
	doc, err := html.Parse(strings.NewReader(raw))
	if err != nil {
		return nil
	}
	postDate := parseDateFromRaw(raw)

	var posts []Post
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "article" {
			post := Post{PostedAt: postDate}
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				text := collectText(c)
				if text == "" {
					continue
				}
				if post.Body == "" {
					post.Body = text
				} else {
					post.Body += " " + text
				}
			}
			if post.Body != "" {
				posts = append(posts, post)
			}
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return posts
}

func parseDateFromRaw(raw string) time.Time {
	if m := datetimeAttr.FindStringSubmatch(raw); len(m) > 1 {
		if t, err := time.Parse(time.RFC3339, m[1]); err == nil {
			return t
		}
		if t, err := time.Parse("2006-01-02", m[1]); err == nil {
			return t
		}
	}
	if m := dateTextRe.FindStringSubmatch(raw); len(m) > 1 {
		if t, err := time.Parse("2006-01-02", m[1]); err == nil {
			return t
		}
		if t, err := time.Parse("2 Jan 2006", m[1]); err == nil {
			return t
		}
	}
	return time.Time{}
}

func HasPainSignal(text string) bool {
	lower := strings.ToLower(text)
	for _, signal := range painSignals {
		if strings.Contains(lower, signal) {
			return true
		}
	}
	return false
}

func stripTags(s string) string {
	s = stripTagRe.ReplaceAllString(s, " ")
	return strings.Join(strings.Fields(s), " ")
}

func collectText(n *html.Node) string {
	if n.Type == html.TextNode {
		return strings.TrimSpace(n.Data)
	}
	var parts []string
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if t := collectText(c); t != "" {
			parts = append(parts, t)
		}
	}
	return strings.Join(parts, " ")
}
