package scoring

import "testing"

func TestScoreContactQualityRoleAccount(t *testing.T) {
	t.Parallel()
	if got := ScoreContactQuality([]string{"email:info@acme.com"}); got != ContactQualityRoleAccount {
		t.Fatalf("got %q", got)
	}
}

func TestScoreContactQualityNamed(t *testing.T) {
	t.Parallel()
	if got := ScoreContactQuality([]string{"email:john.doe@acme.com"}); got != ContactQualityNamed {
		t.Fatalf("got %q", got)
	}
}
