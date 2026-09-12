#!/usr/bin/env bash
# P0-4: cross-mention discover + one scrape round on VPS (B9 soak).
#
# Usage:
#   bash scripts/ops/vps-p0-telegram-soak.sh
#
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
# shellcheck disable=SC1091
source "$ROOT/scripts/lib/vps_ssh.sh"

vps_load_config "$ROOT"
if ! vps_ssh_require "$ROOT"; then
	exit 1
fi

vps_ssh "set -euo pipefail
cd '${VPS_REMOTE_DIR}'
CFG=/app/config/sources.telegram.yaml

channels_before=\$(python3 -c \"import json; print(len(json.load(open('data/runtime/discovered_telegram_channels.json')).get('channels',[])))\" 2>/dev/null || echo 0)
domains_before=\$(python3 -c \"import json; print(len(json.load(open('data/runtime/discovered_telegram_domains.json')).get('domains',[])))\" 2>/dev/null || echo 0)

echo '=== telegram discover (SERP seeds + cross_mention; skip global search) ==='
timeout 1200 docker compose exec -T -e TELEGRAM_DISCOVER_SKIP_SEARCH=1 \\
	parser python3 -m sources.telegram.scraper --config \"\$CFG\" --discover-only || true

echo '=== telegram scrape (channel_search on 15 seeds) ==='
timeout 1800 docker compose exec -T parser python3 -m sources.telegram.scraper --config \"\$CFG\" --stdout >/tmp/tg-p0-soak.ndjson 2>/tmp/tg-p0-soak.log || true
lines=\$(wc -l < /tmp/tg-p0-soak.ndjson 2>/dev/null || echo 0)
pain=\$(grep -c 'postback\\|keitaro\\|voluum\\|binom\\|tracker' /tmp/tg-p0-soak.ndjson 2>/dev/null || echo 0)
tail -5 /tmp/tg-p0-soak.log 2>/dev/null || true

channels_after=\$(python3 -c \"import json; print(len(json.load(open('data/runtime/discovered_telegram_channels.json')).get('channels',[])))\" 2>/dev/null || echo 0)
domains_after=\$(python3 -c \"import json; print(len(json.load(open('data/runtime/discovered_telegram_domains.json')).get('domains',[])))\" 2>/dev/null || echo 0)

echo
echo '=== P0-4 cross-mention soak summary ==='
echo \"channels: \${channels_before} -> \${channels_after}\"
echo \"domains: \${domains_before} -> \${domains_after}\"
echo \"scrape_ndjson_lines: \${lines}\"
echo \"pain_keyword_hits: \${pain}\"
"

printf 'vps-p0-telegram-soak: ok\n'
