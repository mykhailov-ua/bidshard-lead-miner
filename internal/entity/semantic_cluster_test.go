package entity

import "testing"

func TestSemanticClusterIDStable(t *testing.T) {
	t.Parallel()
	a := SemanticClusterID([]string{"voluum", "postback"}, "")
	b := SemanticClusterID([]string{"postback", "voluum"}, "")
	if a == "" || a != b {
		t.Fatalf("a=%q b=%q", a, b)
	}
}

func TestSemanticClusterIDPainPreferred(t *testing.T) {
	t.Parallel()
	got := SemanticClusterID([]string{"x"}, "Voluum migration pain")
	if got == "" {
		t.Fatal("expected cluster id")
	}
}
