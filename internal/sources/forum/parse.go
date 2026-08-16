package forum

import (
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

var (
	postBodyRe = regexp.MustCompile(`(?is)<div[^>]*class="[^"]*postbody[^"]*"[^>]*>(.*?)</div>`)
	usernameRe = regexp.MustCompile(`(?is)<div[^>]*class="[^"]*username[^"]*"[^>]*>(.*?)</div>`)
)

type Post struct {
	Author string
	Body   string
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

func parseWithRegex(raw string) []Post {
	bodies := postBodyRe.FindAllStringSubmatch(raw, -1)
	if len(bodies) == 0 {
		return nil
	}
	users := usernameRe.FindAllStringSubmatch(raw, -1)

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
		posts = append(posts, Post{Author: author, Body: body})
	}
	return posts
}

func parseWithTokenizer(raw string) []Post {
	doc, err := html.Parse(strings.NewReader(raw))
	if err != nil {
		return nil
	}

	var posts []Post
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "article" {
			post := Post{}
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
	s = regexp.MustCompile(`(?is)<[^>]+>`).ReplaceAllString(s, " ")
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
