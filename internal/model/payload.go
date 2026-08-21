package model

// MaxCrawlHTMLBytes caps HTML retained on the hot pipeline path (taskCh + workers).
// Full page fetch may be larger; only a prefix is kept for stack/prescan regexes.
const MaxCrawlHTMLBytes = 256 << 10

// LimitCrawlHTML truncates crawl HTML to MaxCrawlHTMLBytes.
func LimitCrawlHTML(html string) string {
	if len(html) <= MaxCrawlHTMLBytes {
		return html
	}
	return html[:MaxCrawlHTMLBytes]
}
