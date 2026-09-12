package serp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestAppendTelegramChannelDiscoveriesTriageSkipsNoise(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "channels.json")

	results := []SERPResult{
		{Title: "join", Snippet: "@voluum community", URL: "https://t.me/voluum"},
		{Title: "news", Snippet: "@igaming_news digest", URL: "https://t.me/igaming_news"},
		{Title: "jobs", Snippet: "@partnerkin_job board", URL: "https://t.me/partnerkin_job"},
	}
	if err := appendTelegramChannelDiscoveries(path, "site:t.me voluum", results); err != nil {
		t.Fatalf("append: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var file tgChannelFile
	if err := json.Unmarshal(raw, &file); err != nil {
		t.Fatalf("json: %v", err)
	}
	if len(file.Channels) != 1 {
		t.Fatalf("channels=%d want 1: %+v", len(file.Channels), file.Channels)
	}
	if file.Channels[0].Username != "voluum" {
		t.Fatalf("username=%q want voluum", file.Channels[0].Username)
	}
}
