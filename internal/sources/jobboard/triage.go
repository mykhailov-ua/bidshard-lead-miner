package jobboard

import "strings"

var mediaBuyerTokens = []string{
	"media buyer", "mediabuyer", "media buying", "медиабаер", "медіабаєр",
	"traffic manager", "traffic arbitrage", "search arbitrage", "арбитраж",
	"arbitrage", "affiliate marketing", "performance marketing",
	"user acquisition", "ppc", "google ads", "meta ads", "facebook ads",
	"keitaro", "binom", "clickflare", "redtrack", "gambling", "igaming",
	"nutra", "sweepstakes", "demand gen", "dsp",
}

// ShouldCrawl reports whether SERP metadata or URL looks like a media-buyer lead surface.
func ShouldCrawl(url, title, snippet string) bool {
	text := strings.ToLower(strings.TrimSpace(url + " " + title + " " + snippet))
	if text == "" {
		return false
	}
	for _, tok := range mediaBuyerTokens {
		if strings.Contains(text, tok) {
			return true
		}
	}
	return Kind(url) == KindProfile
}
