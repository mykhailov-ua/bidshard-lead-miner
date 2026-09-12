package classify

import (
	"regexp"
	"strings"
)

// BidShardPain is H10 pain bucket + tier hint for sales cards (not auto-quote).
type BidShardPain struct {
	PainBucket     string
	TierHint       string
	TierConfidence string
	PitchKey       string
	PitchLine      string
}

var (
	cloakStackRe = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\b(?:hideclick|adspect|cloak\s*it|cloaking\.house)\b`),
		regexp.MustCompile(`(?i)\bcloak\b.{0,40}\b(?:overprice|expensive|too\s+expensive|click\s+limit)`),
		regexp.MustCompile(`(?i)\b(?:overprice|expensive)\b.{0,40}\bcloak\b`),
	}
	shaveRe = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\b(?:shav(?:e|es|ing)|scrub(?:bing)?)\b.{0,60}\b(?:lead|deposit|ftd|conversion)`),
		regexp.MustCompile(`(?i)\b(?:prove|proof)\b.{0,40}\b(?:shav|scrub|discrep)`),
		regexp.MustCompile(`(?i)\bdiscrep(?:ancy|ancies)\b`),
		regexp.MustCompile(`(?i)\b(?:tracker|трекер)\b.{0,30}\b(?:network|партнерк)`),
		regexp.MustCompile(`(?i)\b(?:network|партнерк)\b.{0,30}\b(?:tracker|трекер)`),
	}
	infraScaleRe = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\b(?:keitaro|binom|voluum)\b.{0,40}\b(?:32\s*gb|ram|oom|502|503|504)\b`),
		regexp.MustCompile(`(?i)\bmysql\b.{0,40}\b(?:cpu|100%|spike|slow)\b`),
		regexp.MustCompile(`(?i)\b(?:clickhouse|ingest)\b`),
		regexp.MustCompile(`(?i)\b(?:redirect|click-to-land).{0,40}\b(?:slow|1\.5s|kill)\b`),
		regexp.MustCompile(`(?i)\b\d{2,}\s*k\s*(?:click|clicks)\b`),
		regexp.MustCompile(`(?i)\b\d+\s*(?:к|k)\s*клик`),
		regexp.MustCompile(`(?i)\badmin\b.{0,30}\b(?:min|load|hang)\b`),
	}
	abuseDDoSRe = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\b(?:ddos|d\.dos)\b`),
		regexp.MustCompile(`(?i)\b(?:click\s+fraud|bot\s+link|competitor.{0,20}bot)\b`),
		regexp.MustCompile(`(?i)\b(?:hoster|hosting)\b.{0,40}\b(?:block|abuse|ddos)\b`),
		regexp.MustCompile(`(?i)\btracking\s+domain\b.{0,40}\b(?:attack|flood|bot)\b`),
	}
	volumeRe = regexp.MustCompile(`(?i)(?:\d{2,}\s*k\s*(?:click|clicks)|\d+\s*(?:к|k)\s*клик)`)
)

func anyMatch(body string, patterns []*regexp.Regexp) bool {
	for _, rx := range patterns {
		if rx.MatchString(body) {
			return true
		}
	}
	return false
}

// ClassifyBidShardPain maps chat text to pain bucket + tier hint. Nil when no bucket matches.
func ClassifyBidShardPain(text string) *BidShardPain {
	body := strings.TrimSpace(text)
	if body == "" {
		return nil
	}

	if anyMatch(body, abuseDDoSRe) {
		conf := "med"
		if volumeRe.MatchString(body) {
			conf = "high"
		} else if !regexp.MustCompile(`(?i)\b(?:volume|clicks|клик|traffic)\b`).MatchString(body) {
			conf = "low"
		}
		return &BidShardPain{
			PainBucket:     "abuse_ddos",
			TierHint:       "enterprise",
			TierConfidence: conf,
			PitchKey:       "ebpf_edge",
			PitchLine:      "eBPF/XDP edge drop before TCP stack (qualify volume before Enterprise)",
		}
	}

	if anyMatch(body, infraScaleRe) {
		tier := "scale"
		if volumeRe.MatchString(body) || regexp.MustCompile(`(?i)\b(?:network|agency)\b`).MatchString(body) {
			tier = "network"
		}
		conf := "low"
		if volumeRe.MatchString(body) {
			conf = "med"
		}
		return &BidShardPain{
			PainBucket:     "infra_scale",
			TierHint:       tier,
			TierConfidence: conf,
			PitchKey:       "clickhouse_ingest",
			PitchLine:      "Go ingest + ClickHouse analytics; ms redirects on same VPS",
		}
	}

	if anyMatch(body, shaveRe) {
		return &BidShardPain{
			PainBucket:     "shave_discrepancy",
			TierHint:       "pro",
			TierConfidence: "med",
			PitchKey:       "postback_trail",
			PitchLine:      "Postback Trail: every hop + raw gateway response; export shave proof",
		}
	}

	if anyMatch(body, cloakStackRe) {
		tier := "starter"
		if regexp.MustCompile(`(?i)\b(?:fb|facebook|google)\b`).MatchString(body) {
			tier = "pro"
		}
		return &BidShardPain{
			PainBucket:     "cloak_stack_cost",
			TierHint:       tier,
			TierConfidence: "med",
			PitchKey:       "cloak_filter_endpoint",
			PitchLine:      "Built-in filter endpoint on VPS; decoy 202 vs external cloak APIs",
		}
	}

	return nil
}

// TierLabel renders tier hint for sales cards.
func TierLabel(tier string) string {
	switch tier {
	case "starter":
		return "Starter"
	case "pro":
		return "Pro"
	case "scale":
		return "Scale"
	case "network":
		return "Network"
	case "enterprise":
		return "Enterprise"
	default:
		if tier == "" {
			return ""
		}
		return strings.ToUpper(tier[:1]) + tier[1:]
	}
}

// PainBucketLabel renders bucket for sales cards.
func PainBucketLabel(bucket string) string {
	switch bucket {
	case "cloak_stack_cost":
		return "cloak overprice"
	case "shave_discrepancy":
		return "shave / discrepancy"
	case "infra_scale":
		return "infra scale"
	case "abuse_ddos":
		return "click fraud / DDoS"
	default:
		return strings.ReplaceAll(bucket, "_", " ")
	}
}
