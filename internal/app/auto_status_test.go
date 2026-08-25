package app

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bidshard/parser/internal/config"
	"github.com/bidshard/parser/internal/sources/forum"
)

func TestCollectAutoStatusFromRegistries(t *testing.T) {
	dir := t.TempDir()
	forumPath := filepath.Join(dir, "forum.json")
	supplyPath := filepath.Join(dir, "registry.json")
	headlessPath := filepath.Join(dir, "headless.json")

	_, err := forum.AppendThreadDiscoveries(forumPath, "test", "dork", []forum.ThreadDiscovery{
		{URL: "https://affiliatefix.com/threads/voluum-pain.1", Title: "Voluum pain"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := writeTestJSON(supplyPath, map[string]any{
		"sources": []map[string]any{
			{"domain": "affnet.com", "types": []string{"supply", "tgweb"}, "triage_status": "keep"},
			{"domain": "noise.example", "types": []string{"tgweb"}, "triage_status": "drop"},
		},
	}); err != nil {
		t.Fatal(err)
	}
	if err := writeTestJSON(headlessPath, map[string]any{
		"items": []map[string]any{{"url": "https://example.com/affiliate"}},
	}); err != nil {
		t.Fatal(err)
	}

	cfg := config.Config{
		ParserAutoDiscover:      true,
		ForumRegistryPath:       forumPath,
		SourceRegistryPath:      supplyPath,
		LanderHeadlessQueuePath: headlessPath,
		ProxyDailyMBCap:         100,
		ProxyBudgetStatePath:    filepath.Join(dir, "proxy_budget.json"),
	}
	st := CollectAutoStatus(t.Context(), cfg)
	if st.ForumThreads != 1 {
		t.Fatalf("forum_threads=%d want 1", st.ForumThreads)
	}
	if st.SourceRegistry["supply"] != 1 || st.SourceRegistry["tgweb"] != 2 {
		t.Fatalf("registry=%v", st.SourceRegistry)
	}
	if st.RegistryDropped != 1 {
		t.Fatalf("dropped=%d want 1", st.RegistryDropped)
	}
	if st.HeadlessQueued != 1 {
		t.Fatalf("headless=%d want 1", st.HeadlessQueued)
	}

	var buf bytes.Buffer
	WriteAutoStatus(&buf, st)
	if !strings.Contains(buf.String(), "forum_threads=1") {
		t.Fatalf("output=%q", buf.String())
	}
}

func TestWriteAutoReportJSONL(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "auto_report.jsonl")
	st := AutoStatus{ForumThreads: 2}
	if err := WriteAutoReportJSONL(path, st); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"forum_threads":2`) {
		t.Fatalf("raw=%q", raw)
	}
}

func writeTestJSON(path string, v any) error {
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	return os.WriteFile(path, raw, 0o644)
}
