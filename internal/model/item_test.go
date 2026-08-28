package model

import (
	"strings"
	"testing"
)

func TestRawItemTextSkipsStackOnLander(t *testing.T) {
	t.Parallel()

	item := RawItem{
		Source:    "lander:voluum.com",
		Raw:       "media buyer postback",
		CrawlHTML: `<script>voluumtrk.init()</script>`,
	}
	text := item.Text()
	if strings.Contains(text, "[stack:") {
		t.Fatalf("lander text must not include stack hint: %q", text)
	}
}

func TestRawItemTextKeepsStackOnTgWeb(t *testing.T) {
	t.Parallel()

	item := RawItem{
		Source:    "tgweb:buyer-site.com",
		Raw:       "media buyer postback",
		CrawlHTML: `<script>voluumtrk.init()</script>`,
	}
	text := item.Text()
	if !strings.Contains(text, "[stack:") {
		t.Fatalf("tgweb text should include stack hint: %q", text)
	}
}
