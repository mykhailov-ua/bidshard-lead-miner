package telegrambot

import "testing"

func TestLeadNotifierNotifyMinScore(t *testing.T) {
	t.Parallel()
	n := &LeadNotifier{minScore: 50, minScoreNonTelegram: 70}
	if got := n.notifyMinScore("telegram:@voluum"); got != 50 {
		t.Fatalf("telegram min score: got %d want 50", got)
	}
	if got := n.notifyMinScore("webpain:example.com"); got != 70 {
		t.Fatalf("non-telegram min score: got %d want 70", got)
	}
}
