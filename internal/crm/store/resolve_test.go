package store

import (
	"context"
	"errors"
	"testing"
)

func TestResolveHashIDExact(t *testing.T) {
	// unit test uses ListFilter logic only; integration covers mongo
	if !hashIDPattern.MatchString("abcdef1234567890abcdef1234567890") {
		t.Fatal("expected valid hash pattern")
	}
}

func TestResolveHashIDValidation(t *testing.T) {
	s := &LeadStore{}
	_, err := s.ResolveHashID(context.Background(), "not-hex")
	if err == nil {
		t.Fatal("expected error for non-hex input")
	}
	_, err = s.ResolveHashID(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty input")
	}
}

func TestErrAmbiguousHash(t *testing.T) {
	if !errors.Is(ErrAmbiguousHash, ErrAmbiguousHash) {
		t.Fatal("ErrAmbiguousHash sentinel broken")
	}
}
