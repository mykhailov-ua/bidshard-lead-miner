package jobboard

import (
	"regexp"
	"strings"

	"golang.org/x/net/html"
)

var (
	ogTitleRe        = regexp.MustCompile(`(?is)<meta[^>]+property="og:title"[^>]+content="([^"]+)"`)
	ogDescRe         = regexp.MustCompile(`(?is)<meta[^>]+property="og:description"[^>]+content="([^"]+)"`)
	metaDescRe       = regexp.MustCompile(`(?is)<meta[^>]+name="description"[^>]+content="([^"]+)"`)
	douCompanyRe     = regexp.MustCompile(`(?is)<div[^>]*class="[^"]*b-compinfo[^"]*"[^>]*>.*?<a[^>]+href="[^"]*/companies/([^"/]+)/?"[^>]*>([^<]+)</a>`)
	douVacancyBodyRe = regexp.MustCompile(`(?is)<div[^>]*class="[^"]*b-typo[^"]*vacancy-section[^"]*"[^>]*>(.*?)</div>`)
	douH1Re          = regexp.MustCompile(`(?is)<h1[^>]*>(.*?)</h1>`)
	djinniCompanyRe  = regexp.MustCompile(`(?is)href="/jobs/company-([^"/]+)/"[^>]*>([^<]+)<`)
	djinniJobBodyRe  = regexp.MustCompile(`(?is)<div[^>]*class="[^"]*job-post__description[^"]*"[^>]*>(.*?)</div>`)
	profileSectionRe = regexp.MustCompile(`(?is)<section[^>]*id="chapter-experience"[^>]*>(.*?)</section>`)
	stripTagRe       = regexp.MustCompile(`(?is)<[^>]+>`)
	spaceRe          = regexp.MustCompile(`\s+`)
)

// Page is parsed job-board content for lead emission.
type Page struct {
	Kind    string
	Title   string
	Company string
	Body    string
	Slug    string
}

// ParseHTML extracts structured fields from DOU or Djinni HTML.
func ParseHTML(rawURL, html string) (Page, bool) {
	kind := Kind(rawURL)
	if kind == KindUnknown {
		return Page{}, false
	}
	page := Page{
		Kind: kind,
		Slug: CompanySlug(rawURL),
	}
	page.Title = firstNonEmpty(
		metaContent(ogTitleRe, html),
		stripTags(firstMatch(douH1Re, html)),
	)
	switch kind {
	case KindVacancy, KindCompany:
		page.Company, page.Body = parseEmployerPage(rawURL, html, kind)
	case KindProfile:
		page.Body = parseProfileBody(html)
	}
	page.Body = strings.TrimSpace(page.Body)
	if page.Title == "" && page.Body == "" {
		return Page{}, false
	}
	if page.Company == "" {
		page.Company = page.Slug
	}
	return page, true
}

func parseEmployerPage(rawURL, html string, kind string) (company, body string) {
	host := HostFromURL(rawURL)
	switch host {
	case HostDOU:
		if m := douCompanyRe.FindStringSubmatch(html); len(m) >= 3 {
			company = stripTags(m[2])
		}
		var parts []string
		for _, m := range douVacancyBodyRe.FindAllStringSubmatch(html, -1) {
			if text := stripTags(m[1]); text != "" {
				parts = append(parts, text)
			}
		}
		if len(parts) == 0 {
			parts = append(parts, metaContent(ogDescRe, html), metaContent(metaDescRe, html))
		}
		body = strings.Join(parts, "\n")
	case HostDjinni:
		if m := djinniCompanyRe.FindStringSubmatch(html); len(m) >= 3 {
			company = stripTags(m[2])
		}
		if m := djinniJobBodyRe.FindStringSubmatch(html); len(m) >= 2 {
			body = stripTags(m[1])
		}
		if body == "" {
			body = metaContent(ogDescRe, html)
		}
		if kind == KindCompany && body == "" {
			body = extractVisibleText(html)
		}
	}
	return strings.TrimSpace(company), strings.TrimSpace(body)
}

func parseProfileBody(html string) string {
	if m := profileSectionRe.FindStringSubmatch(html); len(m) >= 2 {
		return stripTags(m[1])
	}
	return metaContent(metaDescRe, html)
}

func metaContent(re *regexp.Regexp, html string) string {
	if m := re.FindStringSubmatch(html); len(m) >= 2 {
		return stripTags(m[1])
	}
	return ""
}

func firstMatch(re *regexp.Regexp, html string) string {
	if m := re.FindStringSubmatch(html); len(m) >= 2 {
		return m[1]
	}
	return ""
}

func stripTags(s string) string {
	s = strings.ReplaceAll(s, "<br>", "\n")
	s = strings.ReplaceAll(s, "<br/>", "\n")
	s = strings.ReplaceAll(s, "<br />", "\n")
	s = stripTagRe.ReplaceAllString(s, " ")
	return strings.TrimSpace(spaceRe.ReplaceAllString(s, " "))
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v != "" {
			return v
		}
	}
	return ""
}

func extractVisibleText(raw string) string {
	doc, err := html.Parse(strings.NewReader(raw))
	if err != nil {
		return ""
	}
	var buf strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			text := strings.TrimSpace(n.Data)
			if text != "" {
				if buf.Len() > 0 {
					buf.WriteByte(' ')
				}
				buf.WriteString(text)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return buf.String()
}
