#!/usr/bin/env bash
# H6 manual intel queue from infra_clusters.json (Keitaro/Shodan clusters).
#
# Usage:
#   bash scripts/ops/infra-osint-queue.sh
#   bash scripts/ops/infra-osint-queue.sh | column -t -s,
#
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

PATH_JSON="${INFRA_CLUSTERS_PATH:-data/runtime/infra_clusters.json}"
MIN_SCORE="${INFRA_OSINT_MIN_SCORE:-30}"

if [[ ! -f "$PATH_JSON" ]]; then
	printf 'infra-osint-queue: missing %s\n' "$PATH_JSON" >&2
	exit 1
fi

python3 - "$PATH_JSON" "$MIN_SCORE" <<'PY'
import json
import sys

path, min_score = sys.argv[1], int(sys.argv[2])
with open(path, encoding="utf-8") as f:
    data = json.load(f)
clusters = data.get("clusters") or data if isinstance(data, list) else []
print("ip,score,tracker,sample_domain,country")
for row in clusters:
    score = int(row.get("score") or 0)
    if score < min_score:
        continue
    ip = str(row.get("ip") or "").strip()
    tracker = str(row.get("tracker_hint") or row.get("tracker") or "").strip()
    domain = str(row.get("sample_domain") or "").strip()
    if not domain and row.get("domains"):
        domain = str(row["domains"][0])
    country = str(row.get("country") or "").strip()
    print(f"{ip},{score},{tracker},{domain},{country}")
PY
