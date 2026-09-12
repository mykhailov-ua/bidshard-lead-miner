package config

import (
	"testing"
)

func TestLeadNotifyChatIDsExplicit(t *testing.T) {
	t.Setenv("CRM_TELEGRAM_LEAD_NOTIFY_CHAT_IDS", "-1004489522664")
	t.Setenv("TELEGRAM_ALERT_CHANNEL", "-100111")
	got := LeadNotifyChatIDs([]int64{6995552007})
	if len(got) != 1 || got[0] != -1004489522664 {
		t.Fatalf("got %v", got)
	}
}

func TestLeadNotifyChatIDsAlertFallback(t *testing.T) {
	t.Setenv("CRM_TELEGRAM_LEAD_NOTIFY_CHAT_IDS", "")
	t.Setenv("TELEGRAM_ALERT_CHANNEL", "-100222")
	got := LeadNotifyChatIDs([]int64{6995552007})
	if len(got) != 1 || got[0] != -100222 {
		t.Fatalf("got %v", got)
	}
}

func TestLeadNotifyChatIDsGroupFromAllowed(t *testing.T) {
	t.Setenv("CRM_TELEGRAM_LEAD_NOTIFY_CHAT_IDS", "")
	t.Setenv("TELEGRAM_ALERT_CHANNEL", "")
	got := LeadNotifyChatIDs([]int64{6995552007, -100333})
	if len(got) != 1 || got[0] != -100333 {
		t.Fatalf("got %v", got)
	}
}
