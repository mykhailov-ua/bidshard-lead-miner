package entity

// AppendUniqueTag adds tag when not already present.
func AppendUniqueTag(tags []string, tag string) []string {
	for _, existing := range tags {
		if existing == tag {
			return tags
		}
	}
	return append(tags, tag)
}
