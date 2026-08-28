package gemini

import (
	"context"
	"testing"
)

func TestClassifyICPPromptRejectsProgrammaticDisplay(t *testing.T) {
	t.Parallel()

	cl := newTestClient(t, `{"icp":"none","hot":false,"spend_tier":"unknown","why":"programmatic display buyer not tracker ICP"}`)
	res, err := cl.ClassifyICP(context.Background(), "Head of programmatic running CPM brand awareness on openRTB exchange")
	if err != nil {
		t.Fatal(err)
	}
	if res.ICP != "none" {
		t.Fatalf("icp=%q want none", res.ICP)
	}
}

func TestClassifyICPAcceptsPerformanceBuyer(t *testing.T) {
	t.Parallel()

	cl := newTestClient(t, `{"icp":"starter","hot":true,"spend_tier":"15k-150k","why":"voluum postback pain"}`)
	res, err := cl.ClassifyICP(context.Background(), "Switching from Voluum, postback failing on high volume igaming traffic.")
	if err != nil {
		t.Fatal(err)
	}
	if res.ICP != "starter" || !res.Hot {
		t.Fatalf("icp=%q hot=%v", res.ICP, res.Hot)
	}
}
