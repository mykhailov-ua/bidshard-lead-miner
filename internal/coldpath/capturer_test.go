package coldpath

import "testing"

func TestCapturerMasksContact(t *testing.T) {
	t.Parallel()

	c := NewCapturer(4)
	ch := c.Events()

	c.TryCapture(Event{
		Reason:      ReasonLowScore,
		ContactHint: "buyer@igaming-team.com",
		Snippet:     "voluum alternative",
	})

	ev := <-ch
	if ev.ContactHint != "b***@igaming-team.com" {
		t.Fatalf("contact_hint=%q", ev.ContactHint)
	}
}

func TestCapturerDropsWhenFull(t *testing.T) {
	t.Parallel()

	c := NewCapturer(1)
	c.TryCapture(Event{Reason: ReasonLowScore, Snippet: "one"})
	c.TryCapture(Event{Reason: ReasonLowScore, Snippet: "two"})

	if c.Dropped() != 1 {
		t.Fatalf("dropped=%d want 1", c.Dropped())
	}
}
