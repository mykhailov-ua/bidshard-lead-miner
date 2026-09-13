package store

import "go.mongodb.org/mongo-driver/bson"

// applyRawLeadFilter keeps keyword-scored leads without completed Gemini warm path.
func applyRawLeadFilter(q bson.M) {
	q["$or"] = bson.A{
		bson.M{"analysis_status": "raw"},
		bson.M{"analysis_status": ""},
		bson.M{"analysis_status": bson.M{"$exists": false}},
	}
}
