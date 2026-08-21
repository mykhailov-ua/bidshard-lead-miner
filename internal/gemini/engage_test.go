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
