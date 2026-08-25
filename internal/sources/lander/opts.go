package lander

import "github.com/bidshard/parser/internal/config"

// PageFetchOptsFromConfig builds fetch options for 24/7 static vs inline/defer headless.
func PageFetchOptsFromConfig(cfg config.Config, sourceFamily string) PageFetchOptions {
	inlineHeadless := cfg.LanderHeadless && !cfg.LanderHeadlessDefer
	return PageFetchOptions{
		HeadlessEnabled: inlineHeadless,
		HeadlessDefer:   cfg.LanderHeadlessDefer,
		QueuePath:       cfg.LanderHeadlessQueuePath,
		SourceFamily:    sourceFamily,
	}
}
