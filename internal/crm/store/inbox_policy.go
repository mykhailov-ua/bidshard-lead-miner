package store

import (
	"os"
	"strconv"
	"strings"
)

const defaultInboxEngagePriorityMin = 70

// InboxEngagePriorityMin returns CRM sales inbox floor from CRM_ENGAGE_PRIORITY_MIN (0 disables).
func InboxEngagePriorityMin() int {
	raw := strings.TrimSpace(os.Getenv("CRM_ENGAGE_PRIORITY_MIN"))
	if raw == "" {
		return defaultInboxEngagePriorityMin
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 {
		return defaultInboxEngagePriorityMin
	}
	return n
}

// ResolveEngagePriorityMin applies inbox default when query param is unset (negative = use default).
func ResolveEngagePriorityMin(raw string, inboxDefault bool) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		if inboxDefault {
			return InboxEngagePriorityMin()
		}
		return 0
	}
	if raw == "0" || strings.EqualFold(raw, "false") {
		return 0
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 {
		return 0
	}
	return n
}
