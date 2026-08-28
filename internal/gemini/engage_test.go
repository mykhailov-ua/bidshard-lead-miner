package gemini

import (
	"context"
	"testing"
)

func TestClassifyEngagement(t *testing.T) {
	t.Parallel()

	cl := newTestClient(t, `{"pilot_signals":["migration_intent","competitor_stack","tracker_pain"],"pilot_qualified":true,"pilot_why":"switching from voluum with postback pain","outreach_channel":"telegram","outreach_angle":"Voluum migration and postback reliability","outreach_draft":"Hi - saw you're moving off Voluum due to postback issues. We run a self-hosted tracker built for that pain."}`)

	result, err := cl.ClassifyEngagement(context.Background(), EngagementInput{
		Text:         "switching from voluum, postback failing",
		Stack:        []string{"voluum"},
		ICP:          "starter",
		ContactTypes: []string{"telegram"},
		Source:       "telegram:test",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.PilotQualified {
		t.Fatal("expected pilot qualified")
	}
	if result.OutreachChannel != "telegram" {
		t.Fatalf("channel=%q", result.OutreachChannel)
	}
	if result.OutreachDraft == "" {
		t.Fatal("expected outreach draft")
	}

	qual, tags := ApplyEngagementPilot(result)
	if !qual {
		t.Fatal("expected qualified tags")
	}
	if tags[0] != "pilot-qualified" {
		t.Fatalf("tags=%v", tags)
	}
}

func TestApplyEngagementPilotNurture(t *testing.T) {
	t.Parallel()

	qual, tags := ApplyEngagementPilot(EngagementResult{
		PilotSignals:   []string{"tracker_pain"},
		PilotQualified: false,
	})
	if qual {
		t.Fatal("expected not qualified")
	}
	if tags[0] != "pilot-nurture" {
		t.Fatalf("tags=%v", tags)
	}
}

func TestClassifyEngagementEmailOutreach(t *testing.T) {
	t.Parallel()

	cl := newTestClient(t, `{"pilot_signals":["spend_budget","competitor_stack","tracker_pain"],"pilot_qualified":true,"pilot_why":"supply ads contact with voluum stack","outreach_channel":"email","outreach_subject":"Self-hosted tracker for your media buy ops","outreach_angle":"Voluum billing pain on supply side","outreach_draft":"Hi - we help media buyers running Voluum move to self-hosted tracking with predictable billing. Open to a short call this week?"}`)

	result, err := cl.ClassifyEngagement(context.Background(), EngagementInput{
		Text:         "partnerships@acme.com voluum billing issues on supply",
		Stack:        []string{"voluum"},
		ICP:          "pro",
		SpendTier:    "15k-150k",
		ContactTypes: []string{"email", "telegram"},
		Source:       "ads_txt:acme.com",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.OutreachChannel != "email" {
		t.Fatalf("channel=%q want email", result.OutreachChannel)
	}
	if result.OutreachSubject == "" {
		t.Fatal("expected outreach subject for email channel")
	}
	if result.OutreachDraft == "" {
		t.Fatal("expected outreach draft")
	}
}

func TestReconcileEngagementPrefersEmailForSupply(t *testing.T) {
	t.Parallel()

	r := ReconcileEngagement(EngagementResult{
		OutreachChannel: "telegram",
		OutreachSubject: "should clear",
		OutreachAngle:   "angle",
		OutreachDraft:   "draft",
	}, []string{"email", "telegram"}, "supply:partner.com")

	if r.OutreachChannel != "email" {
		t.Fatalf("channel=%q want email", r.OutreachChannel)
	}
	if r.OutreachSubject != "should clear" {
		t.Fatalf("subject=%q want preserved for email", r.OutreachSubject)
	}
}

func TestReconcileEngagementClearsSubjectForTelegram(t *testing.T) {
	t.Parallel()

	r := ReconcileEngagement(EngagementResult{
		OutreachChannel: "email",
		OutreachSubject: "RE: fake",
		OutreachDraft:   "draft",
	}, []string{"telegram"}, "forum:affiliatefix.com/x")

	if r.OutreachChannel != "telegram" {
		t.Fatalf("channel=%q", r.OutreachChannel)
	}
	if r.OutreachSubject != "" {
		t.Fatalf("subject=%q want empty for telegram", r.OutreachSubject)
	}
}

func TestLeadBatchResultFromItemReconcilesEmail(t *testing.T) {
	t.Parallel()

	item := leadBatchItem{
		ID:              "h1",
		ICP:             "pro",
		OutreachChannel: "telegram",
		OutreachSubject: "Partnership inquiry",
		OutreachDraft:   "email body",
	}
	in := LeadBatchInput{
		ID:           "h1",
		Source:       "ads_txt:acme.com",
		ContactTypes: []string{"email", "telegram"},
	}
	res := leadBatchResultFromItem(item, false, in)
	if res.Engagement.OutreachChannel != "email" {
		t.Fatalf("channel=%q", res.Engagement.OutreachChannel)
	}
	if res.Engagement.OutreachSubject != "Partnership inquiry" {
		t.Fatalf("subject=%q", res.Engagement.OutreachSubject)
	}
}
