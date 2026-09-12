"""BidShard pain bucket + tier hint (H10). Keep in sync with internal/classify/bidshard_pain.go."""

from __future__ import annotations

import re
from dataclasses import dataclass

_CLOAK_STACK_RE = [
    re.compile(r"(?i)\b(?:hideclick|adspect|cloak\s*it|cloaking\.house)\b"),
    re.compile(r"(?i)\bcloak\b.{0,40}\b(?:overprice|expensive|too\s+expensive|click\s+limit)"),
    re.compile(r"(?i)\b(?:overprice|expensive)\b.{0,40}\bcloak\b"),
]

_SHAVE_RE = [
    re.compile(r"(?i)\b(?:shav(?:e|es|ing)|scrub(?:bing)?)\b.{0,60}\b(?:lead|deposit|ftd|conversion)"),
    re.compile(r"(?i)\b(?:prove|proof)\b.{0,40}\b(?:shav|scrub|discrep)"),
    re.compile(r"(?i)\bdiscrep(?:ancy|ancies)\b"),
    re.compile(r"(?i)\b(?:tracker|трекер)\b.{0,30}\b(?:network|партнерк)"),
    re.compile(r"(?i)\b(?:network|партнерк)\b.{0,30}\b(?:tracker|трекер)"),
    re.compile(r"(?i)\bв\s+трекере\s+\d+.{0,30}\bпартнерк"),
]

_INFRA_SCALE_RE = [
    re.compile(r"(?i)\b(?:keitaro|binom|voluum)\b.{0,40}\b(?:32\s*gb|ram|oom|502|503|504)\b"),
    re.compile(r"(?i)\bmysql\b.{0,40}\b(?:cpu|100%|spike|slow)\b"),
    re.compile(r"(?i)\b(?:clickhouse|ingest)\b"),
    re.compile(r"(?i)\b(?:redirect|click-to-land).{0,40}\b(?:slow|1\.5s|kill)\b"),
    re.compile(r"(?i)\b\d{2,}\s*k\s*(?:click|clicks)\b"),
    re.compile(r"(?i)\b\d+\s*(?:к|k)\s*клик"),
    re.compile(r"(?i)\badmin\b.{0,30}\b(?:min|load|hang)\b"),
]

_ABUSE_DDOS_RE = [
    re.compile(r"(?i)\b(?:ddos|d\.dos)\b"),
    re.compile(r"(?i)\b(?:click\s+fraud|bot\s+link|competitor.{0,20}bot)\b"),
    re.compile(r"(?i)\b(?:hoster|hosting)\b.{0,40}\b(?:block|abuse|ddos)\b"),
    re.compile(r"(?i)\btracking\s+domain\b.{0,40}\b(?:attack|flood|bot)\b"),
]

_VOLUME_RE = re.compile(r"(?i)(?:\d{2,}\s*k\s*(?:click|clicks)|\d+\s*(?:к|k)\s*клик)")

_PITCH_LINES = {
    "cloak_filter_endpoint": "Built-in filter endpoint on VPS; decoy 202 vs external cloak APIs",
    "postback_trail": "Postback Trail: every hop + raw gateway response; export shave proof",
    "clickhouse_ingest": "Go ingest + ClickHouse analytics; ms redirects on same VPS",
    "ebpf_edge": "eBPF/XDP edge drop before TCP stack (qualify volume before Enterprise)",
}


@dataclass(frozen=True)
class BidShardPain:
    pain_bucket: str
    tier_hint: str
    tier_confidence: str
    pitch_key: str
    pitch_line: str

    def passes_m3_bucket_gate(self) -> bool:
        return self.tier_confidence in ("med", "high")


def classify_bidshard_pain(text: str) -> BidShardPain | None:
    body = (text or "").strip()
    if not body:
        return None

    if any(rx.search(body) for rx in _ABUSE_DDOS_RE):
        conf = "high" if _VOLUME_RE.search(body) else "med"
        if conf == "med" and not re.search(r"(?i)\b(?:volume|clicks|клик|traffic)\b", body):
            conf = "low"
        return BidShardPain(
            pain_bucket="abuse_ddos",
            tier_hint="enterprise",
            tier_confidence=conf,
            pitch_key="ebpf_edge",
            pitch_line=_PITCH_LINES["ebpf_edge"],
        )

    if any(rx.search(body) for rx in _INFRA_SCALE_RE):
        tier = "scale"
        if _VOLUME_RE.search(body) or re.search(r"(?i)\b(?:network|agency)\b", body):
            tier = "network"
        return BidShardPain(
            pain_bucket="infra_scale",
            tier_hint=tier,
            tier_confidence="med" if _VOLUME_RE.search(body) else "low",
            pitch_key="clickhouse_ingest",
            pitch_line=_PITCH_LINES["clickhouse_ingest"],
        )

    if any(rx.search(body) for rx in _SHAVE_RE):
        return BidShardPain(
            pain_bucket="shave_discrepancy",
            tier_hint="pro",
            tier_confidence="med",
            pitch_key="postback_trail",
            pitch_line=_PITCH_LINES["postback_trail"],
        )

    if any(rx.search(body) for rx in _CLOAK_STACK_RE):
        tier = "pro" if re.search(r"(?i)\b(?:fb|facebook|google)\b", body) else "starter"
        return BidShardPain(
            pain_bucket="cloak_stack_cost",
            tier_hint=tier,
            tier_confidence="med",
            pitch_key="cloak_filter_endpoint",
            pitch_line=_PITCH_LINES["cloak_filter_endpoint"],
        )

    return None


def format_tier_label(tier_hint: str) -> str:
    mapping = {
        "starter": "Starter",
        "pro": "Pro",
        "scale": "Scale",
        "network": "Network",
        "enterprise": "Enterprise",
    }
    return mapping.get(tier_hint, tier_hint.title())


def format_pain_bucket_label(bucket: str) -> str:
    mapping = {
        "cloak_stack_cost": "cloak overprice",
        "shave_discrepancy": "shave / discrepancy",
        "infra_scale": "infra scale",
        "abuse_ddos": "click fraud / DDoS",
    }
    return mapping.get(bucket, bucket.replace("_", " "))
