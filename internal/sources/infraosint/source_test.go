package infraosint

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/bidshard/parser/internal/model"
)

func TestCrawlerCollectIntelOnly(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "infra_clusters.json")
	payload := map[string]any{
		"exported_at": "2026-09-12",
		"source":      "shodan",
		"clusters": []map[string]any{
			{
				"ip":            "1.2.3.4",
				"domains":       []string{"a.com", "b.com", "c.com", "d.com"},
				"tracker_hint":  "keitaro",
				"country":       "US",
				"sample_domain": "a.com",
			},
		},
	}
	raw, _ := json.Marshal(payload)
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	c := NewCrawler(path)
	var items []model.RawItem
	err := c.Collect(context.Background(), EmitFunc(func(_ context.Context, item model.RawItem) error {
		items = append(items, item)
		return nil
	}))
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("items=%d want 1", len(items))
	}
	if items[0].Source != "infraosint:1.2.3.4" {
		t.Fatalf("source=%q", items[0].Source)
	}
}
