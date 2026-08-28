package scoring

import "testing"

func TestComputeEngagePriorityDisplacement(t *testing.T) {
	t.Parallel()
	base := ComputeEngagePriority(EngagePriorityInput{
		Priority: PriorityMedium,
		Score:    20,
	})
	hot := ComputeEngagePriority(EngagePriorityInput{
		Priority:         PriorityMedium,
		Score:            20,
		DisplacementTier: DisplacementHot,
		PilotQualified:   true,
		Stack:            []string{"voluum"},
	})
	if hot <= base {
		t.Fatalf("hot=%d base=%d", hot, base)
	}
}

func TestComputeEngagePriorityRedditBoost(t *testing.T) {
	t.Parallel()
	base := ComputeEngagePriority(EngagePriorityInput{
		Priority: PriorityMedium,
		Score:    30,
	})
	reddit := ComputeEngagePriority(EngagePriorityInput{
		Priority:     PriorityMedium,
		Score:        30,
		SourceFamily: "reddit:r/affiliatemarketing",
	})
	if reddit <= base {
		t.Fatalf("reddit=%d base=%d", reddit, base)
	}
}
