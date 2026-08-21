package store

import (
	"errors"
	"testing"
)

func TestNormalizeStatus(t *testing.T) {
	got, err := NormalizeStatus("Contacted")
	if err != nil {
		t.Fatal(err)
	}
	if got != "contacted" {
		t.Fatalf("got %q want contacted", got)
	}
}

func TestNormalizeStatusInvalid(t *testing.T) {
	_, err := NormalizeStatus("bogus")
	if !errors.Is(err, ErrInvalidStatus) {
		t.Fatalf("err=%v want ErrInvalidStatus", err)
	}
}
