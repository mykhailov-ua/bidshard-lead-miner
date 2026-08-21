package lander

import (
	"regexp"
	"strings"
)

var (
	mailtoHrefRe  = regexp.MustCompile(`(?i)mailto:([a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,})`)
	skypeHrefRe   = regexp.MustCompile(`(?i)skype:([a-z][a-z0-9.,\-_]{2,31})`)
	footerTagRe   = regexp.MustCompile(`(?is)<footer[^>]*>(.*?)</footer>`)
	footerIDRe    = regexp.MustCompile(`(?is)<(?:div|section)[^>]+id=["']footer["'][^>]*>(.*?)</(?:div|section)>`)
	footerClassRe = regexp.MustCompile(`(?is)<(?:div|section)[^>]+class=["'][^"']*\bfooter\b[^"']*["'][^>]*>(.*?)</(?:div|section)>`)
	contentInfoRe = regexp.MustCompile(`(?is)<(?:footer|div|section)[^>]+role=["']contentinfo["'][^>]*>(.*?)</(?:footer|div|section)>`)
	blockTagRe    = regexp.MustCompile(`(?is)<(script|style|noscript|svg)[^>]*>[\s\S]*?</(script|style|noscript|svg)>`)
	headRe        = regexp.MustCompile(`(?is)<head[^>]*>.*?</head>`)
)

// NetworkLandingPaths returns crawl paths prioritized for affiliate network sites:
// home and about first (SPA sites often 404 contact paths with same shell), then LPR paths.
func NetworkLandingPaths() []string {
	return []string{
		"/",
		"/about",
		"/about-us",
		"/contact",
		"/contacts",
		"/contact-us",
		"/affiliate",
		"/affiliates",
		"/affiliate-program",
		"/partner",
		"/partners",
		"/partner-program",
		"/become-a-partner",
		"/become-partner",
		"/for-affiliates",
		"/for-partners",
	}
}

// ExtractStaticLandingText parses primitive HTML/Tailwind landings (footer LPR contacts, mailto, visible copy).
func ExtractStaticLandingText(html string) string {
	if strings.TrimSpace(html) == "" {
		return ""
	}
	var parts []string
	for _, email := range mailtoHrefRe.FindAllStringSubmatch(html, -1) {
		if len(email) > 1 {
			parts = append(parts, strings.ToLower(email[1]))
		}
	}
	for _, m := range skypeHrefRe.FindAllStringSubmatch(html, -1) {
		if len(m) > 1 {
			parts = append(parts, "skype:"+strings.ToLower(m[1]))
		}
	}
	parts = append(parts, extractFooterSegments(html)...)
	if body := extractVisibleBodyText(html); body != "" {
		parts = append(parts, body)
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}

// TextForContactExtract merges framework payloads (Next/RSC) with static HTML contact regions.
func TextForContactExtract(html string) (text string, method string) {
	diagnosed, method := DiagnoseExtract(html)
	static := ExtractStaticLandingText(html)
	if static == "" {
		return diagnosed, method
	}
	if diagnosed == "" {
		return static, "static_html"
	}
	// Union framework extract with footer/mailto static text; do not pick one or the other.
	return strings.TrimSpace(diagnosed + " " + static), method + "+static"
}

func extractFooterSegments(html string) []string {
	seen := make(map[string]struct{})
	var out []string
	add := func(raw string) {
		raw = stripTags(raw)
		if raw == "" {
			return
		}
		if _, ok := seen[raw]; ok {
			return
		}
		seen[raw] = struct{}{}
		out = append(out, raw)
	}
	for _, re := range []struct {
		pattern *regexp.Regexp
	}{
		{footerTagRe},
		{footerIDRe},
		{footerClassRe},
		{contentInfoRe},
	} {
		for _, m := range re.pattern.FindAllStringSubmatch(html, -1) {
			if len(m) > 1 {
				add(m[1])
			}
		}
	}
	return out
}

func extractVisibleBodyText(html string) string {
	html = headRe.ReplaceAllString(html, " ")
	html = blockTagRe.ReplaceAllString(html, " ")
	text := stripTags(html)
	if len(text) > 50000 {
		text = text[:50000]
	}
	return text
}
