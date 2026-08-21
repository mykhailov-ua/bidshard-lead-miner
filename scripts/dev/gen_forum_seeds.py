#!/usr/bin/env python3
"""Regenerate forum_threads.csv with offline fixture seeds (forum-fixture.test)."""

from pathlib import Path

topics = [
    ("voluum-alternative-recommendations", "voluum"),
    ("keitaro-vs-binom-for-nutra", "keitaro"),
    ("postback-missing-conversions-issue", "postback"),
    ("self-hosted-tracker-docker-setup", "postback"),
    ("redtrack-pricing-increase-alternatives", "redtrack"),
    ("clickflare-review-and-migration", "voluum"),
    ("best-tracker-for-native-ads-2026", "binom"),
    ("cloaker-integration-with-voluum", "voluum"),
    ("s2s-postback-latency-troubleshooting", "postback"),
    ("budget-overspend-protection-in-trackers", "binom"),
]

root = Path(__file__).resolve().parents[2]
out = root / "data" / "seeds" / "forum_threads.csv"
lines = [
    "# Forum seed threads for grey-market tracking pain harvesting.",
    "# forum-fixture.test URLs load HTML from testdata/forum/ (offline dev/CI).",
    "# Live hosts (affiliatefix, BHW) need residential proxy - see docs/ops.md.",
    "url,notes",
]
idx = 1001
while len(lines) - 4 < 50:
    for slug, tag in topics:
        if len(lines) - 4 >= 50:
            break
        lines.append(f"https://forum-fixture.test/threads/{slug}.{idx}/,{tag}")
        idx += 1

out.write_text("\n".join(lines) + "\n", encoding="utf-8")
print(f"wrote {out} ({len(lines) - 4} urls)")
