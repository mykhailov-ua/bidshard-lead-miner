package store

import (
	"testing"
	"time"
)

func TestParseSinceDuration(t *testing.T) {
	d, err := ParseSinceDuration("24h")
	if err != nil || d != 24*time.Hour {
		t.Fatalf("24h: d=%v err=%v", d, err)
	}
	d, err = ParseSinceDuration("7d")
	if err != nil || d != 7*24*time.Hour {
		t.Fatalf("7d: d=%v err=%v", d, err)
	}
	if _, err := ParseSinceDuration("bad"); err == nil {
		t.Fatal("expected error for bad duration")
	}
}
