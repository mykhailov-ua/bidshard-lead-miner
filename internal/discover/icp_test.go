package discover

import "testing"

func TestTelegramHarvestDorks(t *testing.T) {
	t.Parallel()
	cfg := ICPConfig{
		SerpDorks:    []string{"site:t.me voluum"},
		PWADorks:     []string{"site:t.me PWA postback"},
		HostingDorks: []string{"site:t.me AlexHost keitaro"},
	}
	got := cfg.TelegramHarvestDorks()
	if len(got) != 3 {
		t.Fatalf("len=%d want 3", len(got))
	}
}
