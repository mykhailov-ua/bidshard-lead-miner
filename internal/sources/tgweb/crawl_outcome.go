package tgweb

type domainCrawlOutcome int

// Domain crawl outcomes drive registry updates in Collect:
//   - emitted: mark crawled_at (success)
//   - hardFail / noContacts: leave pending for retry on next run
const (
	crawlOutcomeEmitted    domainCrawlOutcome = 1
	crawlOutcomeNoContacts domainCrawlOutcome = 2
	crawlOutcomeHardFail   domainCrawlOutcome = 3
)

// crawlRunStats aggregates per-run counters; deferred_retry = hardFail + noContacts.
type crawlRunStats struct {
	emitted       int
	contactsFound int
	hardFail      int
	noContacts    int
	spa404Stop    int
}
