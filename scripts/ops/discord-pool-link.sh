#!/usr/bin/env bash
# Validate root Discord token pool files (discord.tokens.pool.json).
#
# Usage:
#   bash scripts/ops/discord-pool-link.sh
#   bash scripts/ops/discord-pool-link.sh --export
#
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
POOL="${DISCORD_POOL_FILE:-$ROOT/discord.tokens.pool.json}"
EXPORT="${1:-}"

if [[ ! -f "$POOL" ]]; then
	printf 'discord-pool-link: missing %s\n' "$POOL" >&2
	exit 1
fi

python3 - "$ROOT" "$POOL" "$EXPORT" <<'PY'
import json
import sys

root, pool_path, export = sys.argv[1], sys.argv[2], sys.argv[3]
with open(pool_path, encoding="utf-8") as fh:
    pool = json.load(fh)

tokens = []
for row in pool.get("tokens", []):
    path = root + "/" + row["file"]
    try:
        with open(path, encoding="utf-8") as fh:
            tok = fh.read().strip()
    except OSError:
        print(f"discord-pool-link: skip missing {row['file']}")
        continue
    if not tok:
        print(f"discord-pool-link: skip empty {row['file']}")
        continue
    tokens.append(tok)
    print(f"discord-pool-link: ok {row['file']} ({row.get('label', row['id'])})")

if not tokens:
    raise SystemExit("discord-pool-link: no token files found in repo root")

if export == "--export":
    print(",".join(tokens))
else:
    print(f"discord-pool-link: {len(tokens)} token(s) ready")
PY
