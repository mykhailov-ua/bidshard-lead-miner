package discord

import "testing"

func TestTokenPoolRotate(t *testing.T) {
	t.Parallel()

	pool := NewTokenPool([]string{"a", "b"})
	if pool.Pick() != "a" {
		t.Fatalf("first=%q want a", pool.Pick())
	}
	pool.Rotate()
	if pool.Pick() != "b" {
		t.Fatalf("after rotate=%q want b", pool.Pick())
	}
	pool.Rotate()
	if pool.Pick() != "a" {
		t.Fatalf("wrap=%q want a", pool.Pick())
	}
}

func TestTokenPoolEmpty(t *testing.T) {
	t.Parallel()
	pool := NewTokenPool([]string{"", "  "})
	if pool.Pick() != "" || pool.Len() != 0 {
		t.Fatal("expected empty pool")
	}
}
