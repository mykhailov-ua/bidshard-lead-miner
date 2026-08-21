#!/usr/bin/env bash
#
# Create dev domain registry from example when discovered_telegram_domains.json is missing.
#
# Usage:
#   make tgweb-seed
#
# Note: never overwrites an existing registry (production discover data is preserved).
#
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
EXAMPLE="$ROOT/data/runtime/discovered_telegram_domains.json.example"
TARGET="$ROOT/data/runtime/discovered_telegram_domains.json"

if [[ -f "$TARGET" ]]; then
	# Never overwrite an existing registry; production discover data must survive seed runs.
	echo "tgweb-seed: registry exists: $TARGET"
	python3 - "$TARGET" <<'PY'
import json, sys
with open(sys.argv[1], encoding="utf-8") as f:
    n = len([d for d in (json.load(f).get("domains") or []) if (d.get("domain") or "").strip()])
print(f"tgweb-seed: {n} domain(s)")
PY
	exit 0
fi

if [[ ! -f "$EXAMPLE" ]]; then
	echo "tgweb-seed: missing example: $EXAMPLE" >&2
	exit 1
fi

mkdir -p "$(dirname "$TARGET")"
cp "$EXAMPLE" "$TARGET"
echo "tgweb-seed: created $TARGET from example (buylink.pro, topxpartners.com)"
