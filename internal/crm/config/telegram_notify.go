package config

import (
	"os"
	"strings"
)

func envBool(key string, fallback bool) bool {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	switch strings.ToLower(v) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

// LeadNotifyChatIDs resolves CRM_TELEGRAM_LEAD_NOTIFY_CHAT_IDS with fallbacks:
// TELEGRAM_ALERT_CHANNEL, then negative ids from CRM_TELEGRAM_ALLOWED_CHAT_IDS (groups).
func LeadNotifyChatIDs(allowed []int64) []int64 {
	explicit := parseChatIDs(strings.TrimSpace(os.Getenv("CRM_TELEGRAM_LEAD_NOTIFY_CHAT_IDS")))
	if len(explicit) > 0 {
		return explicit
	}
	alert := parseChatIDs(strings.TrimSpace(os.Getenv("TELEGRAM_ALERT_CHANNEL")))
	if len(alert) > 0 {
		return alert
	}
	var groups []int64
	for _, id := range allowed {
		if id < 0 {
			groups = append(groups, id)
		}
	}
	return groups
}
