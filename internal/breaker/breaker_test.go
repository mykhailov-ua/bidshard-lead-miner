package breaker

import (
	"net/http"
	"testing"
	"time"
)

func TestSourceBreakerOpensOnCF403(t *testing.T) {
	t.Parallel()

	b := NewSourceBreaker()
	source := "forum"

	for i := 0; i < 2; i++ {
		b.RecordCloudflareBlock(source)
		if !b.Allow(source) {
			t.Fatalf("breaker opened early at %d", i+1)
		}
	}

	b.RecordCloudflareBlock(source)
	if b.Allow(source) {
		t.Fatal("expected breaker open after 3 cloudflare blocks")
	}
}

func TestSourceBreakerOpensAfterFive429(t *testing.T) {
	t.Parallel()

	b := NewSourceBreaker()
	source := "reddit"

	for i := 0; i < 4; i++ {
		b.RecordFailure(source, http.StatusTooManyRequests, 0)
		if !b.Allow(source) {
			t.Fatalf("breaker opened early at %d", i+1)
		}
	}

	b.RecordFailure(source, http.StatusTooManyRequests, 0)
	if b.Allow(source) {
		t.Fatal("expected breaker open after 5 failures")
	}
	if b.WaitDuration(source) <= 0 {
		t.Fatal("expected positive retry_after")
	}
}

func TestRetryAfterExtendsOpenDuration(t *testing.T) {
	t.Parallel()

	b := NewSourceBreaker()
	source := "github"

	for i := 0; i < 5; i++ {
		b.RecordFailure(source, http.StatusTooManyRequests, 0)
	}

	b.RecordFailure(source, http.StatusTooManyRequests, 2*time.Minute)
	wait := b.WaitDuration(source)
	if wait < 90*time.Second {
		t.Fatalf("expected extended open duration, got %v", wait)
	}
}

func TestParseRetryAfterSeconds(t *testing.T) {
	t.Parallel()

	if got := ParseRetryAfter("120"); got != 120*time.Second {
		t.Fatalf("got %v", got)
	}
}
