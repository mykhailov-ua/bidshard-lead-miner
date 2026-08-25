package discord

// Package discord discovers public invite links (SERP/registry only).
// Bot join and private servers are out of scope; operators add channel IDs manually.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const DefaultRegistryPath = "data/runtime/discovered_discord_invites.json"

var inviteRe = regexp.MustCompile(`(?i)(?:discord\.gg/|discord\.com/invite/)([a-z0-9-]{2,32})`)

type InviteEntry struct {
	InviteCode   string `json:"invite_code"`
	GuildHint    string `json:"guild_hint,omitempty"`
	Source       string `json:"source"`
	Query        string `json:"query,omitempty"`
	TriageStatus string `json:"triage_status,omitempty"`
	At           string `json:"at"`
}

type InviteFile struct {
	Invites []InviteEntry `json:"invites"`
}

func LoadRegistry(path string) (InviteFile, error) {
	if path == "" {
		path = DefaultRegistryPath
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return InviteFile{}, nil
		}
		return InviteFile{}, err
	}
	var f InviteFile
	if err := json.Unmarshal(raw, &f); err != nil {
		return InviteFile{}, err
	}
	return f, nil
}

func SaveRegistry(path string, f InviteFile) error {
	if path == "" {
		path = DefaultRegistryPath
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func ExtractInviteCodes(text string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, m := range inviteRe.FindAllStringSubmatch(text, -1) {
		if len(m) < 2 {
			continue
		}
		code := strings.ToLower(strings.TrimSpace(m[1]))
		if code == "" {
			continue
		}
		if _, ok := seen[code]; ok {
			continue
		}
		seen[code] = struct{}{}
		out = append(out, code)
	}
	return out
}

// HeuristicTriageInvite filters obvious junk without Gemini (registry is discover-only).
func HeuristicTriageInvite(code, hint string) (status, why string) {
	text := strings.ToLower(code + " " + hint)
	for _, tok := range []string{"nsfw", "porn", "crypto pump", "onlyfans", "casino spam"} {
		if strings.Contains(text, tok) {
			return "drop", tok
		}
	}
	return "keep", "public_invite"
}

func AppendInvites(path, source, query string, codes []string, hints map[string]string) (added int, err error) {
	if len(codes) == 0 {
		return 0, nil
	}
	f, err := LoadRegistry(path)
	if err != nil {
		return 0, err
	}
	seen := map[string]struct{}{}
	for _, e := range f.Invites {
		seen[strings.ToLower(e.InviteCode)] = struct{}{}
	}
	now := time.Now().UTC().Format(time.RFC3339)
	for _, code := range codes {
		code = strings.ToLower(strings.TrimSpace(code))
		if code == "" {
			continue
		}
		if _, ok := seen[code]; ok {
			continue
		}
		seen[code] = struct{}{}
		hint := ""
		if hints != nil {
			hint = hints[code]
		}
		status, _ := HeuristicTriageInvite(code, hint)
		if status == "drop" {
			continue
		}
		f.Invites = append(f.Invites, InviteEntry{
			InviteCode:   code,
			GuildHint:    hint,
			Source:       source,
			Query:        query,
			TriageStatus: status,
			At:           now,
		})
		added++
	}
	if added == 0 {
		return 0, nil
	}
	return added, SaveRegistry(path, f)
}
