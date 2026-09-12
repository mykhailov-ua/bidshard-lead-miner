#!/usr/bin/env bash
# H6 weekly infra OSINT export (Shodan -> infra_clusters.json). Intel branch only; no auto-CRM.
#
# Usage:
#   SHODAN_API_KEY=... bash scripts/ops/infra-shodan-export.sh
#   bash scripts/ops/infra-shodan-export.sh data/runtime/infra_clusters.json
#
# Requires: curl, jq, SHODAN_API_KEY in env or config/env/.env.proxy.local

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

OUT="${1:-data/runtime/infra_clusters.json}"
mkdir -p "$(dirname "$OUT")"

if [[ -z "${SHODAN_API_KEY:-}" && -f "$ROOT/config/env/.env.proxy.local" ]]; then
	set -a
	# shellcheck disable=SC1090
	source "$ROOT/config/env/.env.proxy.local"
	set +a
fi

if [[ -z "${SHODAN_API_KEY:-}" ]]; then
	echo "infra-shodan-export: SHODAN_API_KEY missing; set key and re-run" >&2
	echo "infra-shodan-export: see data/runtime/infra_clusters.json.example for file shape" >&2
	exit 1
fi

QUERY='http.title:"Keitaro" OR http.html:"kclick_id"'
URL="https://api.shodan.io/shodan/host/search?key=${SHODAN_API_KEY}&query=$(python3 -c 'import urllib.parse,sys; print(urllib.parse.quote(sys.argv[1]))' "$QUERY")"

RAW="$(curl -fsS "$URL")"
MATCHES="$(echo "$RAW" | jq -c '.matches // []')"
EXPORTED_AT="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

jq -n \
	--arg exported_at "$EXPORTED_AT" \
	--argjson matches "$MATCHES" \
	'
{
  exported_at: $exported_at,
  source: "shodan",
  clusters: [
    $matches[] | {
      ip: .ip_str,
      domains: ((.hostnames // []) + ([.domains[]?] // [])) | unique,
      tracker_hint: (if (.http.title // "" | test("Keitaro"; "i")) then "keitaro" else "kclick_id" end),
      asn: ("AS" + ((.asn // "")|tostring)),
      country: (.location.country_code // ""),
      sample_domain: ((.hostnames[0] // .domains[0] // "")),
      notes: "shodan export; manual triage"
    }
  ]
}
' >"$OUT"

echo "infra-shodan-export: wrote $(jq '.clusters | length' "$OUT") clusters to $OUT"
