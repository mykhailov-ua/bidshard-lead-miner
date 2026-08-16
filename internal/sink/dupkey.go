package sink

import (
	"errors"

	"go.mongodb.org/mongo-driver/mongo"
)

func IsDuplicateKey(err error) bool {
	if err == nil {
		return false
	}
	var writeErr mongo.WriteException
	if errors.As(err, &writeErr) {
		for _, e := range writeErr.WriteErrors {
			if e.Code == 11000 {
				return true
			}
		}
	}
	var bulkErr mongo.BulkWriteException
	if errors.As(err, &bulkErr) {
		for _, e := range bulkErr.WriteErrors {
			if e.Code == 11000 {
				return true
			}
		}
	}
	return false
}

func onlyDuplicateKeys(err error) bool {
	var bulkErr mongo.BulkWriteException
	if !errors.As(err, &bulkErr) {
		return false
	}
	if len(bulkErr.WriteErrors) == 0 {
		return false
	}
	for _, e := range bulkErr.WriteErrors {
		if e.Code != 11000 {
			return false
		}
	}
	return true
}
