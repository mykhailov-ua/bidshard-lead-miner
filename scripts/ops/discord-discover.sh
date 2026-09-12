#!/usr/bin/env bash
# Discord catalog discover: SERP invite harvest + guild join + channel registry.
#
# Usage:
#   bash scripts/ops/discord-discover.sh
#
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT"

if command -v docker >/dev/null 2>&1 && [[ -f docker-compose.yaml ]]; then
	docker compose run --rm parser discord discover
elif [[ -x "$ROOT/bin/parser" ]]; then
	"$ROOT/bin/parser" discord discover
else
	go run ./cmd/parser discord discover
fi
