package store

import (
	"os"
	"testing"

	"go.mongodb.org/mongo-driver/bson"
)

func TestResolveEngagePriorityMin(t *testing.T) {
	t.Setenv("CRM_ENGAGE_PRIORITY_MIN", "80")
	if got := ResolveEngagePriorityMin("", true); got != 80 {
		t.Fatalf("default=%d", got)
	}
	if got := ResolveEngagePriorityMin("0", true); got != 0 {
		t.Fatalf("disable=%d", got)
	}
	if got := ResolveEngagePriorityMin("", false); got != 0 {
		t.Fatalf("non-inbox=%d", got)
	}
}

func TestListFilterEngagePriorityMin(t *testing.T) {
	t.Parallel()
	q := ListFilter{MinEngagePriority: 70}.matchQuery()
	min, ok := q["engage_priority"].(bson.M)["$gte"]
	if !ok || min != 70 {
		t.Fatalf("query=%v", q["engage_priority"])
	}
}

func TestInboxEngagePriorityMinUnset(t *testing.T) {
	os.Unsetenv("CRM_ENGAGE_PRIORITY_MIN")
	if got := InboxEngagePriorityMin(); got != defaultInboxEngagePriorityMin {
		t.Fatalf("min=%d", got)
	}
}
