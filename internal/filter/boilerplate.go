package filter

import (
	"strings"
	"unicode"
)

var htmlBoilerplateSignals = []string{
	"viewport width=device-width",
	"meta charset",
	"x-ua-compatible",
	"script nonce",
	"hreflang=",
	"data-theme=",
	"data-head-attrs",
	"content-type text/html",
	"initial-scale=1",
	"theme-color",
	"font-face",
	"async></script",
	"<!--",
}

// RejectHTMLBoilerplate drops tgweb crawl text that is mostly HTML head/meta noise.
func RejectHTMLBoilerplate(text string) (bool, string) {
	text = strings.TrimSpace(text)
	if text == "" {
		return false, ""
	}
	lower := strings.ToLower(text)
	hits := 0
	for _, sig := range htmlBoilerplateSignals {
		if strings.Contains(lower, sig) {
			hits++
		}
	}
	if hits >= 2 {
		return true, "html boilerplate"
	}
	if strings.Count(text, "<") >= 2 && strings.Count(text, ">") >= 2 && naturalWordCount(text) < 10 {
		return true, "html markup noise"
	}
	return false, ""
}

func naturalWordCount(text string) int {
	fields := strings.Fields(text)
	n := 0
	for _, f := range fields {
		letters := 0
		for _, r := range f {
			if unicode.IsLetter(r) {
				letters++
			}
		}
		if letters >= 3 {
			n++
		}
	}
	return n
}
