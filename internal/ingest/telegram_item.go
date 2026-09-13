package ingest

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/bidshard/parser/internal/model"
)

// telegramItem is the Telethon sidecar message schema (NDJSON or MessagePack).
type telegramItem struct {
	Source           string         `json:"source" msgpack:"source"`
	Text             string         `json:"text" msgpack:"text"`
	Contact          string         `json:"contact" msgpack:"contact"`
	Title            string         `json:"title" msgpack:"title"`
	Username         string         `json:"username" msgpack:"username"`
	SenderUserID     int64          `json:"sender_user_id" msgpack:"sender_user_id"`
	SenderBio        string         `json:"sender_bio" msgpack:"sender_bio"`
	SenderProfile    map[string]any `json:"sender_profile" msgpack:"sender_profile"`
	MessageMeta      map[string]any `json:"message_meta" msgpack:"message_meta"`
	ColdOutreachFit  string         `json:"cold_outreach_fit" msgpack:"cold_outreach_fit"`
	MessageID        int64          `json:"message_id" msgpack:"message_id"`
	ReplyToMessageID int64          `json:"reply_to_message_id" msgpack:"reply_to_message_id"`
	ReplyContext     string         `json:"reply_context" msgpack:"reply_context"`
	ChatType         string         `json:"chat_type" msgpack:"chat_type"`
	ChannelRole      string         `json:"channel_role" msgpack:"channel_role"`
	ChannelAbout     string         `json:"channel_about" msgpack:"channel_about"`
	LeadType         string         `json:"lead_type" msgpack:"lead_type"`
	CompanyHint      string         `json:"company_hint" msgpack:"company_hint"`
	SourceChat       string         `json:"source_chat" msgpack:"source_chat"`
	DiscoveredVia    string         `json:"discovered_via" msgpack:"discovered_via"`
}

func (item telegramItem) toRawItem() model.RawItem {
	senderBio := strings.TrimSpace(item.SenderBio)
	if senderBio == "" && item.SenderProfile != nil {
		if about, ok := item.SenderProfile["about"].(string); ok {
			senderBio = strings.TrimSpace(about)
		}
	}
	if fit := strings.TrimSpace(item.ColdOutreachFit); fit != "" && senderBio != "" {
		senderBio = senderBio + " outreach_fit=" + fit
	} else if fit := strings.TrimSpace(item.ColdOutreachFit); fit != "" {
		senderBio = "outreach_fit=" + fit
	}

	contact := item.Contact
	if contact == "" && item.Username != "" {
		contact = item.Username
	}
	if contact == "" && item.SenderUserID > 0 {
		contact = fmt.Sprintf("telegram:user_id:%d", item.SenderUserID)
	}
	if contact == "" && strings.HasPrefix(item.Source, "telegram:") {
		username := strings.TrimPrefix(item.Source, "telegram:")
		username = strings.TrimPrefix(username, "@")
		if username != "" {
			contact = "@" + username
			if item.Username == "" {
				item.Username = username
			}
		}
	}

	raw := item.Text
	if hint := telegramMessageMetaHint(item.MessageMeta); hint != "" {
		raw = strings.TrimSpace(raw + "\n" + hint)
	}

	return model.RawItem{
		Source:           item.Source,
		Raw:              raw,
		Contact:          contact,
		Title:            item.Title,
		Username:         item.Username,
		MessageID:        item.MessageID,
		SenderUserID:     item.SenderUserID,
		ReplyToMessageID: item.ReplyToMessageID,
		ReplyContext:     item.ReplyContext,
		ChatType:         item.ChatType,
		ChannelRole:      item.ChannelRole,
		ChannelAbout:     item.ChannelAbout,
		SenderBio:        senderBio,
		LeadType:         strings.TrimSpace(item.LeadType),
		CompanyHint:      strings.TrimSpace(item.CompanyHint),
		SourceChat:       strings.TrimSpace(item.SourceChat),
		DiscoveredVia:    strings.TrimSpace(item.DiscoveredVia),
		ColdOutreachFit:  strings.TrimSpace(item.ColdOutreachFit),
	}
}

func telegramMessageMetaHint(meta map[string]any) string {
	if meta == nil {
		return ""
	}
	parts := make([]string, 0, 4)
	if v, ok := meta["views"]; ok {
		switch n := v.(type) {
		case float64:
			if n > 0 {
				parts = append(parts, "views="+strconv.Itoa(int(n)))
			}
		case int:
			if n > 0 {
				parts = append(parts, "views="+strconv.Itoa(n))
			}
		}
	}
	if fwd, ok := meta["forward"].(map[string]any); ok {
		if name, ok := fwd["from_name"].(string); ok && strings.TrimSpace(name) != "" {
			parts = append(parts, "forward_from="+strings.TrimSpace(name))
		}
	}
	if rx, ok := meta["reactions"].(map[string]any); ok {
		if total, ok := rx["total"].(float64); ok && total > 0 {
			parts = append(parts, "reactions="+strconv.Itoa(int(total)))
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return "message_meta: " + strings.Join(parts, " ")
}
