package scoring

import "testing"

func TestAssignOutreachQueuePrefersTelegram(t *testing.T) {
	t.Parallel()

	ch, action := AssignOutreachQueue([]string{
		"telegram:@buyer_mx",
		"ops@igaming-team.com",
	}, "telegram")
	if ch != ContactChannelTelegram || action != NextActionTelegramDM {
		t.Fatalf("got %s %s", ch, action)
	}
}

func TestAssignOutreachQueueEmailWhenNoTelegram(t *testing.T) {
	t.Parallel()

	ch, action := AssignOutreachQueue([]string{"partnerships@buylink.pro"}, "telegram")
	if ch != ContactChannelEmail || action != NextActionColdEmail {
		t.Fatalf("got %s %s", ch, action)
	}
}

func TestAssignOutreachQueueForumUser(t *testing.T) {
	t.Parallel()

	ch, action := AssignOutreachQueue([]string{"forum:user/media_buyer"}, "")
	if ch != ContactChannelForum || action != NextActionForumManual {
		t.Fatalf("got %s %s", ch, action)
	}
}

func TestAssignOutreachQueueRespectsOutreachEmail(t *testing.T) {
	t.Parallel()

	ch, action := AssignOutreachQueue([]string{
		"telegram:@buyer_mx",
		"ops@igaming-team.com",
	}, "email")
	if ch != ContactChannelEmail || action != NextActionColdEmail {
		t.Fatalf("got %s %s", ch, action)
	}
}

func TestSourcePrefersEmailOutreach(t *testing.T) {
	t.Parallel()

	ch := []string{ContactChannelEmail}
	if !SourcePrefersEmailOutreach("ads_txt:acme.com", ch) {
		t.Fatal("expected ads_txt to prefer email")
	}
	if SourcePrefersEmailOutreach("forum:affiliatefix.com/x", ch) {
		t.Fatal("forum should not auto-prefer email")
	}
}

func TestChannelsFromContactTypes(t *testing.T) {
	t.Parallel()

	got := ChannelsFromContactTypes([]string{"email", "telegram", "forum_user"})
	if len(got) != 3 {
		t.Fatalf("got %v", got)
	}
}
