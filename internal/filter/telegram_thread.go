package filter

import "strings"

var telegramReplyHelperPhrases = []string{
	"dm me for",
	"dm me privately",
	"free course",
	"affiliate course",
	"mentorship program",
	"signal group",
	"vip signals",
	"paid group",
	"join my channel",
	"contact me for",
	"my service",
	"i help ecommerce",
	"i help e-commerce",
	"we offer",
	"open to connect",
}

// TelegramReplyHelperReject drops service-seller replies in pain threads.
func TelegramReplyHelperReject(text string) (bool, string) {
	body := strings.ToLower(strings.TrimSpace(text))
	if body == "" {
		return false, ""
	}
	for _, phrase := range telegramReplyHelperPhrases {
		if strings.Contains(body, phrase) {
			return true, "reply helper: " + phrase
		}
	}
	return false, ""
}

// TelegramReplyThreadBuyer reports a person asking in a thread where the parent has buyer pain.
func TelegramReplyThreadBuyer(replyToMessageID int64, replyContext, text, username string) bool {
	if replyToMessageID <= 0 {
		return false
	}
	if strings.TrimSpace(username) == "" {
		return false
	}
	parent := strings.TrimSpace(replyContext)
	if parent == "" {
		return false
	}
	if !HasCommercialPainIntent(parent) && !hasPainKeyword(strings.ToLower(parent)) {
		return false
	}
	if drop, _ := TelegramReplyHelperReject(text); drop {
		return false
	}
	return true
}
