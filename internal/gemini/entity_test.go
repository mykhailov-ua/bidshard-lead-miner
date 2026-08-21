package gemini

import (
	"context"
	"testing"
)

func TestClassifyEntitySameActorCombinesPain(t *testing.T) {
	t.Parallel()

	cl := newTestClient(t, `{
		"same_actor": true,
		"actor_confidence": 0.88,
		"unified_pain": "Voluum postback failures and missing FTD conversions across Telegram and forum posts",
		"buyer_intent": "hot",
		"split_recommended": false,
		"why": "Same telegram handle and consistent tracker pain"
	}`)

	res, err := cl.ClassifyEntity(context.Background(), EntityClassifyInput{
		EntityID: "ent-1",
		Sightings: EntitySightingInputsFromSnippets(
			[]string{"telegram:@buyer_mx", "forum:affiliatefix.com/thread"},
			[]string{
				"voluum postback failing on FTD, telegram @buyer_mx",
				"missing conversions in voluum after migration",
			},
			[][]string{{"postback"}, {"voluum"}},
		),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.SameActor {
		t.Fatal("expected same_actor=true")
	}
	if res.ActorConfidence < 0.7 {
		t.Fatalf("confidence=%v", res.ActorConfidence)
	}
	if res.UnifiedPain == "" {
		t.Fatal("expected unified_pain")
	}
	if res.BuyerIntent != "hot" {
		t.Fatalf("buyer_intent=%q", res.BuyerIntent)
	}
	if res.SplitRecommended {
		t.Fatal("expected split_recommended=false")
	}
}

func TestClassifyEntityRecruitingVsBuyerSplit(t *testing.T) {
	t.Parallel()

	cl := newTestClient(t, `{
		"same_actor": false,
		"actor_confidence": 0.82,
		"unified_pain": "",
		"buyer_intent": "cold",
		"split_recommended": true,
		"why": "Recruiting affiliate publishers is not the same actor as complaining about Voluum billing"
	}`)

	res, err := cl.ClassifyEntity(context.Background(), EntityClassifyInput{
		EntityID: "ent-2",
		Sightings: EntitySightingInputsFromSnippets(
			[]string{"forum:affiliatefix.com/a", "forum:affiliatefix.com/b"},
			[]string{
				"our affiliate program recruits high quality crypto traffic",
				"voluum bill too high looking for self-hosted tracker",
			},
			[][]string{{"affiliate"}, {"voluum"}},
		),
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.SameActor {
		t.Fatal("expected same_actor=false")
	}
	if !res.SplitRecommended {
		t.Fatal("expected split_recommended=true")
	}
	if res.BuyerIntent != "cold" && res.BuyerIntent != "none" {
		t.Fatalf("buyer_intent=%q want cold or none", res.BuyerIntent)
	}
}

func TestClassifyEntityEmptyInput(t *testing.T) {
	t.Parallel()

	cl := newTestClient(t, `{}`)
	res, err := cl.ClassifyEntity(context.Background(), EntityClassifyInput{})
	if err != nil {
		t.Fatal(err)
	}
	if res.BuyerIntent != "none" {
		t.Fatalf("buyer_intent=%q", res.BuyerIntent)
	}
}
