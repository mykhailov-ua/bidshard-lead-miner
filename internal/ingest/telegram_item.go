package ingest

import (
	"fmt"
	"strings"

	"github.com/bidshard/parser/internal/model"
)

// telegramItem is the Telethon sidecar message schema (NDJSON or MessagePack).
type telegramItem struct {
	Source           string `json:"source" msgpack:"source"`
	Text             string `json:"text" msgpack:"text"`
	Contact          string `json:"contact" msgpack:"contact"`
	Title            string `json:"title" msgpack:"title"`
	Username         string `json:"username" msgpack:"username"`
	SenderUserID     int64  `json:"sender_user_id" msgpack:"sender_user_id"`
	SenderBio        string `json:"sender_bio" msgpack:"sender_bio"`
	MessageID        int64  `json:"message_id" msgpack:"message_id"`
	ReplyToMessageID int64  `json:"reply_to_message_id" msgpack:"reply_to_message_id"`
	ReplyContext     string `json:"reply_context" msgpack:"reply_context"`
	ChatType         string `json:"chat_type" msgpack:"chat_type"`
	ChannelAbout     string `json:"channel_about" msgpack:"channel_about"`
}

func (item telegramItem) toRawItem() model.RawItem {
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

	return model.RawItem{
		Source:           item.Source,
		Raw:              item.Text,
		Contact:          contact,
		Title:            item.Title,
		Username:         item.Username,
		MessageID:        item.MessageID,
		SenderUserID:     item.SenderUserID,
		ReplyToMessageID: item.ReplyToMessageID,
		ReplyContext:     item.ReplyContext,
		ChatType:         item.ChatType,
		ChannelAbout:     item.ChannelAbout,
		SenderBio:        item.SenderBio,
	}
}
