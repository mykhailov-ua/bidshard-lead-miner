package telethon

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestExportChannelsJSON(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "crawler.db")
	jsonPath := filepath.Join(dir, "channels.json")

	if err := seedCursorChannels(dbPath, []seedChannel{
		{key: "u:alpha", username: "alpha", enabled: 1},
		{key: "u:beta", username: "beta", enabled: 0},
	}); err != nil {
		t.Fatal(err)
	}

	if err := ExportChannelsJSON(dbPath, jsonPath); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatal(err)
	}
	var out struct {
		Channels []map[string]string `json:"channels"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Channels) != 1 || out.Channels[0]["username"] != "alpha" {
		t.Fatalf("export=%v want single alpha channel", out.Channels)
	}
}
