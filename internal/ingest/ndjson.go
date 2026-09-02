package ingest

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/bidshard/parser/internal/model"
)

type ndjsonItem struct {
	Source           string `json:"source"`
	Text             string `json:"text"`
	Contact          string `json:"contact"`
	Title            string `json:"title"`
	Username         string `json:"username"`
	SenderUserID     int64  `json:"sender_user_id"`
	MessageID        int64  `json:"message_id"`
	ReplyToMessageID int64  `json:"reply_to_message_id"`
	ReplyContext     string `json:"reply_context"`
	ChatType         string `json:"chat_type"`
	ChannelAbout     string `json:"channel_about"`
}

func parseNDJSONLine(line []byte) (model.RawItem, error) {
	var item ndjsonItem
	if err := json.Unmarshal(line, &item); err != nil {
		return model.RawItem{}, err
	}
	return item.toRawItem(), nil
}

func (item ndjsonItem) toRawItem() model.RawItem {
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
		ReplyToMessageID: item.ReplyToMessageID,
		ReplyContext:     item.ReplyContext,
		ChatType:         item.ChatType,
		ChannelAbout:     item.ChannelAbout,
	}
}

func validateTelegramItem(item model.RawItem) error {
	if !strings.HasPrefix(item.Source, "telegram:") {
		return nil
	}
	if item.ContactTelegram() == "" {
		return fmt.Errorf("telegram item missing username/contact")
	}
	return nil
}
