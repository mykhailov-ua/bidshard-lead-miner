package scoring

import (
	"context"
	"testing"
)

func TestPrescanPassesTgWebWithAffiliateContext(t *testing.T) {
	t.Parallel()
	reg := NewRegistry("../../testdata/keywords.json")
	if err := reg.Load(context.Background()); err != nil {
		t.Fatal(err)
	}
	text := "affiliate marketing igaming media buyer tracker postback s2s partner program contact us"
	if !PrescanPasses("tgweb:@aff_net:example.com", reg, text) {
		t.Fatal("expected tgweb prescan pass with affiliate context")
	}
}

func TestPrescanPassesTgWebStillRejectsBareContact(t *testing.T) {
	t.Parallel()
	reg := NewRegistry("../../testdata/keywords.json")
	if err := reg.Load(context.Background()); err != nil {
		t.Fatal(err)
	}
	if PrescanPasses("tgweb:example.com", reg, "hello@example.com") {
		t.Fatal("expected bare email page to fail prescan")
	}
}

func TestPrescanPassesNormalSourceStillUsesRegistry(t *testing.T) {
	t.Parallel()
	reg := NewRegistry("../../testdata/keywords.json")
	if err := reg.Load(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !PrescanPasses("lander:example.com", reg, "voluum alternative postback failing") {
		t.Fatal("expected normal prescan pass")
	}
	if PrescanPasses("lander:example.com", reg, "random unrelated text") {
		t.Fatal("expected unrelated lander text to fail")
	}
}
