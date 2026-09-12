package ingest

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/bidshard/parser/internal/model"
)

func parseNDJSONLine(line []byte) (model.RawItem, error) {
	var item telegramItem
	if err := json.Unmarshal(line, &item); err != nil {
		return model.RawItem{}, err
	}
	return item.toRawItem(), nil
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
