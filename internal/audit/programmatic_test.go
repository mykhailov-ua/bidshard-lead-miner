package audit

import (
	"strings"
	"testing"
)

func TestScanLeadsJSONLReader(t *testing.T) {
	t.Parallel()
	input := strings.Join([]string{
		`{"source":"forum:test","snippet":"Head of programmatic display CPM brand awareness campaign"}`,
		`{"source":"forum:test","snippet":"voluum postback failing need alternative"}`,
		`{"source":"reddit:r/test","snippet":"programmatic guaranteed header bidding on publisher SSP"}`,
	}, "\n")
	rep, err := ScanLeadsJSONLReader(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if rep.Total != 3 {
		t.Fatalf("total=%d", rep.Total)
	}
	if rep.WouldDrop != 2 {
		t.Fatalf("would_drop=%d want 2", rep.WouldDrop)
	}
	if rep.ByReason["programmatic vertical"] != 2 {
		t.Fatalf("by_reason=%v", rep.ByReason)
	}
}
