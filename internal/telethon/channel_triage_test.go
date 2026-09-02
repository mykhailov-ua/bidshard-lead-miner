package telethon

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bidshard/parser/internal/gemini"
	_ "modernc.org/sqlite"
)

func TestChannelIDToCursorKey(t *testing.T) {
	tests := map[string]string{
		"user:foo":      "u:foo",
		"user:bar":      "u:bar",
		"invite:abc123": "i:abc123",
		"":              "",
		"chat:123":      "",
	}
	for id, want := range tests {
		if got := channelIDToCursorKey(id); got != want {
			t.Errorf("channelIDToCursorKey(%q) = %q, want %q", id, got, want)
		}
	}
}

func TestRunChannelTriageDisablesCursorDB(t *testing.T) {
	dir := t.TempDir()
	channelsPath := filepath.Join(dir, "channels.json")
	cachePath := filepath.Join(dir, "cache.json")
	dbPath := filepath.Join(dir, "crawler.db")

	channels := channelFile{
		Channels: []channelEntry{
			{Username: "job_board", Title: "Jobs CIS"},
			{InviteHash: "invitehash", Title: "Invite noise"},
		},
	}
	raw, err := json.MarshalIndent(channels, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(channelsPath, raw, 0o644); err != nil {
		t.Fatal(err)
	}

	cache := triageCacheFile{
		Decisions: map[string]string{
			"user:job_board": "drop",
		},
	}
	cacheRaw, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cachePath, cacheRaw, 0o644); err != nil {
		t.Fatal(err)
	}

	if err := seedCursorChannels(dbPath, []seedChannel{
		{key: "u:job_board", username: "job_board", enabled: 1},
		{key: "i:invitehash", inviteHash: "invitehash", enabled: 1},
	}); err != nil {
		t.Fatal(err)
	}

	client := newChannelTriageTestClient(t, `{"items":[{"id":"invite:invitehash","action":"drop","why":"noise"}]}`)

	if err := RunChannelTriage(context.Background(), ChannelTriageConfig{
		ChannelsPath: channelsPath,
		CachePath:    cachePath,
		CursorDBPath: dbPath,
	}, client); err != nil {
		t.Fatal(err)
	}

	updated, err := os.ReadFile(channelsPath)
	if err != nil {
		t.Fatal(err)
	}
	var out channelFile
	if err := json.Unmarshal(updated, &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Channels) != 0 {
		t.Fatalf("expected empty channels file, got %d", len(out.Channels))
	}

	if enabled, err := cursorChannelEnabled(dbPath, "u:job_board"); err != nil {
		t.Fatal(err)
	} else if enabled {
		t.Fatal("expected u:job_board disabled")
	}
	if enabled, err := cursorChannelEnabled(dbPath, "i:invitehash"); err != nil {
		t.Fatal(err)
	} else if enabled {
		t.Fatal("expected i:invitehash disabled")
	}
}

type seedChannel struct {
	key        string
	username   string
	inviteHash string
	enabled    int
}

func seedCursorChannels(dbPath string, rows []seedChannel) error {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Exec(`
		CREATE TABLE telegram_channels (
			channel_key TEXT PRIMARY KEY,
			username TEXT,
			invite_hash TEXT,
			chat_id INTEGER,
			title TEXT NOT NULL DEFAULT '',
			source TEXT NOT NULL DEFAULT 'manual',
			geo TEXT NOT NULL DEFAULT 'global',
			enabled INTEGER NOT NULL DEFAULT 1,
			discovered_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
			cross_mention_at TEXT
		)
	`)
	if err != nil {
		return err
	}
	for _, row := range rows {
		_, err = db.Exec(
			`INSERT INTO telegram_channels (channel_key, username, invite_hash, enabled) VALUES (?, ?, ?, ?)`,
			row.key, row.username, row.inviteHash, row.enabled,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func cursorChannelEnabled(dbPath, channelKey string) (bool, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return false, err
	}
	defer db.Close()
	var enabled int
	err = db.QueryRow(
		"SELECT enabled FROM telegram_channels WHERE channel_key = ?",
		channelKey,
	).Scan(&enabled)
	if err != nil {
		return false, err
	}
	return enabled == 1, nil
}

func newChannelTriageTestClient(t *testing.T, responseText string) *gemini.Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.Path, ":generateContent") {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"candidates": []map[string]any{
				{"content": map[string]any{
					"parts": []map[string]string{
						{"text": responseText},
					},
				}},
			},
		})
	}))
	t.Cleanup(srv.Close)
	cl, err := gemini.NewClient("test-key", "gemini-2.5-flash", gemini.WithBaseURL(srv.URL), gemini.WithHTTPClient(srv.Client()))
	if err != nil {
		t.Fatal(err)
	}
	return cl
}
