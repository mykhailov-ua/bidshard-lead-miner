#!/usr/bin/env bash
# Discord catalog discover: SERP invite harvest + guild join + channel registry.
#
# Usage:
#   bash scripts/ops/discord-discover.sh
#
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

FAST="${DISCORD_DISCOVER_FAST:-1}"

if command -v docker >/dev/null 2>&1 && [[ -f docker-compose.yaml ]]; then
	if [[ "$FAST" == "1" ]]; then
		docker compose run --rm parser discord discover --fast
	else
		docker compose run --rm parser discord discover
	fi
elif [[ -x "$ROOT/bin/parser" ]]; then
	"$ROOT/bin/parser" discord discover
else
	go run ./cmd/parser discord discover
fi
