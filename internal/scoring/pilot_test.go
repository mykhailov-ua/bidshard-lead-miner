package scoring

import (
	"testing"
)

func TestPilotQualifiedTable(t *testing.T) {
	tests := []struct {
		name          string
		text          string
		spendTier     string
		stack         []string
		wantQualified bool
		wantTag       string
	}{
		{
			name:          "1. Spend budget",
			text:          "Looking for tracker with $5k/mo budget",
			wantQualified: false,
			wantTag:       "pilot-spend-budget",
		},
		{
			name:          "2. Competitor stack",
			text:          "Currently using voluum for tracking",
			wantQualified: false,
			wantTag:       "pilot-competitor-stack",
		},
		{
			name:          "3. Tracker pain",
			text:          "Experiencing click loss and postback fail on current setup",
			wantQualified: false,
			wantTag:       "pilot-tracker-pain",
		},
		{
			name:          "4. Infra VPS",
			text:          "Need self-hosted docker installation on digitalocean vps",
			wantQualified: false,
			wantTag:       "pilot-infra-vps",
		},
		{
			name:          "5. USDT OK",
			text:          "Can pay via USDT TRC20 for monthly subscription",
			wantQualified: false,
			wantTag:       "pilot-usdt-ok",
		},
		{
			name:          "6. Buyer role",
			text:          "I am head of media buying team for igaming",
			wantQualified: false,
			wantTag:       "pilot-buyer-role",
		},
		{
			name:          "7. High volume",
			text:          "We drive 100+ FTD daily with high volume traffic",
			wantQualified: false,
			wantTag:       "pilot-high-volume",
		},
		{
			name:          "8. Migration intent",
			text:          "We are switching from keitaro to new tracker solution",
			wantQualified: false,
			wantTag:       "pilot-migration-intent",
		},
		{
			name:          "9. Nurture non-matched",
			text:          "Hello world plain message",
			wantQualified: false,
			wantTag:       "pilot-nurture",
		},
		{
			name:          "10. Qualified with 3 signals",
			text:          "Switching from voluum, $5k budget, self-hosted docker on vps",
			wantQualified: true,
			wantTag:       "pilot-qualified",
		},
		{
			name:          "11. Spend tier unknown ignored",
			spendTier:     "unknown",
			text:          "Hello world plain message",
			wantQualified: false,
			wantTag:       "pilot-nurture",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qual, tags := PilotQualified(tt.spendTier, tt.stack, tt.text)
			if qual != tt.wantQualified {
				t.Errorf("got qual %v, want %v", qual, tt.wantQualified)
			}
			found := false
			for _, tag := range tags {
				if tag == tt.wantTag {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected tag %s in tags %v", tt.wantTag, tags)
			}
		})
	}
}
