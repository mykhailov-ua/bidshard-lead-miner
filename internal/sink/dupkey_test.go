package sink

import (
	"errors"
	"testing"

	"go.mongodb.org/mongo-driver/mongo"
)

func TestIsDuplicateKey(t *testing.T) {
	t.Parallel()

	err := mongo.WriteException{
		WriteErrors: []mongo.WriteError{{Code: 11000}},
	}
	if !IsDuplicateKey(err) {
		t.Fatal("expected duplicate key")
	}
	if IsDuplicateKey(errors.New("other")) {
		t.Fatal("unexpected duplicate")
	}
}
