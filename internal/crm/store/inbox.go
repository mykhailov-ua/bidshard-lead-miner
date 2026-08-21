package store

import "go.mongodb.org/mongo-driver/bson"

var inboxExcludedStatuses = []string{"geo_rejected", "icp_rejected"}

// applyInboxFilter constrains list queries to actionable leads (post-analysis, not parser rejects).
// Empty status (CRM default inbox): hide parser rejects and analysis_status=pending defer queue.
func applyInboxFilter(q bson.M, status string) {
	if status != "" {
		q["status"] = status
	} else {
		q["status"] = bson.M{"$nin": inboxExcludedStatuses}
	}
	q["analysis_status"] = bson.M{"$nin": []string{"pending"}}
}
