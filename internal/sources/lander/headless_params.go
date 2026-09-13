package lander

// DefaultHeadlessProfileRoot stores Playwright storage_state per proxy persona.
const DefaultHeadlessProfileRoot = "data/runtime/browser_profiles"

// HeadlessFetchParams wires HTTP crawl proxy selection into Playwright.
type HeadlessFetchParams struct {
	// ProxyIndex is the index in PARSER_PROXY_LIST used on the preceding HTTP fetch.
	// -1 means direct egress or unknown; Playwright falls back to index 0 when proxies exist.
	ProxyIndex int
	// PersonaAlreadyCounted is set when defer queue already reserved the daily persona slot.
	PersonaAlreadyCounted bool
}
