package telethon

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

const defaultCursorDBPath = "data/runtime/crawler.db"

// channelIDToCursorKey maps triage ids (user:foo, invite:hash) to crawler.db channel_key (u:foo, i:hash).
func channelIDToCursorKey(id string) string {
	id = strings.TrimSpace(id)
	switch {
	case strings.HasPrefix(id, "user:"):
		return "u:" + strings.TrimPrefix(id, "user:")
	case strings.HasPrefix(id, "invite:"):
		return "i:" + strings.TrimPrefix(id, "invite:")
	default:
		return ""
	}
}

func setChannelsEnabled(dbPath string, channelKeys []string, enabled bool) error {
	if dbPath == "" || len(channelKeys) == 0 {
		return nil
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	val := 0
	if enabled {
		val = 1
	}
	for _, key := range channelKeys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		if _, err := db.Exec(
			"UPDATE telegram_channels SET enabled = ?, updated_at = CURRENT_TIMESTAMP WHERE channel_key = ?",
			val, key,
		); err != nil {
			return fmt.Errorf("disable channel %s: %w", key, err)
		}
	}
	return nil
}

type exportChannelEntry struct {
	Username   string `json:"username,omitempty"`
	InviteHash string `json:"invite_hash,omitempty"`
	Title      string `json:"title,omitempty"`
	Source     string `json:"source,omitempty"`
}

// ExportChannelsJSON writes enabled channels from crawler.db to ops registry JSON.
func ExportChannelsJSON(dbPath, jsonPath string) error {
	if dbPath == "" || jsonPath == "" {
		return nil
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	rows, err := db.Query(`
		SELECT username, invite_hash, title, source
		FROM telegram_channels
		WHERE enabled = 1
		ORDER BY updated_at DESC
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	var channels []exportChannelEntry
	for rows.Next() {
		var username, inviteHash, title, source sql.NullString
		if err := rows.Scan(&username, &inviteHash, &title, &source); err != nil {
			return err
		}
		entry := exportChannelEntry{Title: title.String}
		if username.Valid && username.String != "" {
			entry.Username = username.String
		}
		if inviteHash.Valid && inviteHash.String != "" {
			entry.InviteHash = inviteHash.String
		}
		if source.Valid && source.String != "" {
			entry.Source = source.String
		}
		channels = append(channels, entry)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(jsonPath), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(map[string]any{"channels": channels}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(jsonPath, append(raw, '\n'), 0o644)
}
