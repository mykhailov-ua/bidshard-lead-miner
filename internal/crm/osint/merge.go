package osint

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/bidshard/parser/internal/crm/store"
)

type profilesFile struct {
	Profiles []profileRow `json:"profiles"`
}

type membersFile struct {
	Members []memberRow `json:"members"`
}

type profileRow struct {
	UserID          int64  `json:"user_id"`
	Username        string `json:"username"`
	About           string `json:"about"`
	ColdOutreachFit string `json:"cold_outreach_fit"`
	DiscoveredVia   string `json:"discovered_via"`
	SourceChat      string `json:"source_chat"`
}

type memberRow struct {
	UserID       int64  `json:"user_id"`
	Username     string `json:"username"`
	ChatUsername string `json:"chat_username"`
	ChatKey      string `json:"chat_key"`
}

// MergePeopleFiles builds upsert rows from Telethon runtime JSON exports.
func MergePeopleFiles(profilesPath, membersPath string) ([]store.TelegramPeopleSyncInput, error) {
	byID := map[int64]*store.TelegramPeopleSyncInput{}

	if profilesPath != "" {
		raw, err := os.ReadFile(profilesPath)
		if err != nil {
			return nil, fmt.Errorf("read profiles: %w", err)
		}
		var pf profilesFile
		if err := json.Unmarshal(raw, &pf); err != nil {
			return nil, fmt.Errorf("decode profiles: %w", err)
		}
		for _, p := range pf.Profiles {
			if p.UserID <= 0 {
				continue
			}
			row := byID[p.UserID]
			if row == nil {
				row = &store.TelegramPeopleSyncInput{UserID: p.UserID}
				byID[p.UserID] = row
			}
			row.PersonUsername = firstNonEmpty(p.Username, row.PersonUsername)
			row.OutreachFit = firstNonEmpty(p.ColdOutreachFit, row.OutreachFit)
			row.ProfileBio = firstNonEmpty(p.About, row.ProfileBio)
			row.SourceChat = firstNonEmpty(p.SourceChat, row.SourceChat)
			if row.SourceChatRef == "" && strings.TrimSpace(p.SourceChat) != "" {
				row.SourceChatRef = store.TelegramChatRefFromUsername(p.SourceChat)
			}
			row.DiscoveredVia = firstNonEmpty(p.DiscoveredVia, row.DiscoveredVia)
		}
	}

	if membersPath != "" {
		raw, err := os.ReadFile(membersPath)
		if err != nil {
			return nil, fmt.Errorf("read members: %w", err)
		}
		var mf membersFile
		if err := json.Unmarshal(raw, &mf); err != nil {
			return nil, fmt.Errorf("decode members: %w", err)
		}
		for _, m := range mf.Members {
			if m.UserID <= 0 {
				continue
			}
			row := byID[m.UserID]
			if row == nil {
				row = &store.TelegramPeopleSyncInput{UserID: m.UserID}
				byID[m.UserID] = row
			}
			row.PersonUsername = firstNonEmpty(m.Username, row.PersonUsername)
			if key := strings.TrimSpace(m.ChatKey); key != "" {
				row.MemberChatKey = key
			}
			chat := firstNonEmpty(m.ChatUsername, strings.TrimPrefix(m.ChatKey, "u:"))
			if chat != "" {
				row.MemberChat = chat
				if row.SourceChat == "" {
					row.SourceChat = chat
				}
				if row.SourceChatRef == "" && row.SourceChat == chat {
					row.SourceChatRef = store.TelegramChatRefFromUsername(chat)
				}
			}
			if row.DiscoveredVia == "" {
				row.DiscoveredVia = "member_harvest"
			}
		}
	}

	out := make([]store.TelegramPeopleSyncInput, 0, len(byID))
	for _, row := range byID {
		out = append(out, *row)
	}
	return out, nil
}

func firstNonEmpty(a, b string) string {
	a = strings.TrimSpace(a)
	if a != "" {
		return a
	}
	return strings.TrimSpace(b)
}
