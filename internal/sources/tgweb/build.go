package tgweb

import (
	"github.com/bidshard/parser/internal/config"
	"github.com/bidshard/parser/internal/sources/lander"
)

// HeadlessFromConfig returns a headless fetcher when PARSER_LANDER_HEADLESS is set.
func HeadlessFromConfig(cfg config.Config) lander.HeadlessFetcher {
	if cfg.LanderHeadless {
		return lander.NewPlaywrightPoolFetcher(2, cfg.HTTPTimeout)
	}
	return lander.DisabledHeadless{}
}

// BuildCrawler wires HTTP, optional headless, and optional path ranker from config.
func BuildCrawler(cfg config.Config, ranker lander.PathRanker) (*Crawler, error) {
	return NewCrawler(cfg, nil, HeadlessFromConfig(cfg), ranker)
}
