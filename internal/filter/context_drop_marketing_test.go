package filter

import "testing"

func TestRejectNonBuyerContextSEOMarketing(t *testing.T) {
	drop, reason := RejectNonBuyerContext(
		"serp:digiexe.com",
		"voluum too expensive. Top picks for best affiliate tracking software in 2026.",
		"Best Affiliate Tracking Software in 2026",
	)
	if !drop || reason != "seo marketing copy" {
		t.Fatalf("drop=%v reason=%q", drop, reason)
	}
}

func TestRejectNonBuyerContextGitHubCode(t *testing.T) {
	code := `import React from 'react'
import { motion } from 'framer-motion'
export default function Page() {
  const x = useState(0);
  return <div className="x">keitaro voluum</div>;
}`
	drop, reason := RejectNonBuyerContext("github:user/repo", code, "landing")
	if !drop || reason != "github source code paste" {
		t.Fatalf("drop=%v reason=%q", drop, reason)
	}
}
