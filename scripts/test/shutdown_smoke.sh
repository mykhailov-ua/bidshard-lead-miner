#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

timeout 15s env PARSER_OUTPUT=quiet go run ./cmd/parser -scan-once >/dev/null

echo "OK: shutdown smoke passed"
