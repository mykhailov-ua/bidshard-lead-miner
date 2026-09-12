#!/usr/bin/env bash
# Link root MTProto pool files into data/runtime/ (see sessions.pool.json).
#
# Usage:
#   bash scripts/ops/session-pool-link.sh
#   bash scripts/ops/session-pool-link.sh --copy
#
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
POOL="${SESSION_POOL_FILE:-$ROOT/sessions.pool.json}"
MODE="${1:---copy}"

if [[ ! -f "$POOL" ]]; then
	printf 'session-pool-link: missing %s\n' "$POOL" >&2
	exit 1
fi

mkdir -p "$ROOT/data/runtime"

python3 - "$ROOT" "$POOL" "$MODE" <<'PY'
import json
import os
import shutil
import sys

root, pool_path, mode = sys.argv[1], sys.argv[2], sys.argv[3]
with open(pool_path, encoding="utf-8") as fh:
    pool = json.load(fh)

linked = 0
for row in pool.get("sessions", []):
    src = os.path.join(root, row["file"])
    dst = os.path.join(root, row["runtime"])
    if not os.path.isfile(src):
        print(f"session-pool-link: skip missing {row['file']}")
        continue
    os.makedirs(os.path.dirname(dst), exist_ok=True)
    if os.path.lexists(dst):
        os.remove(dst)
    if mode == "--copy":
        shutil.copy2(src, dst)
        print(f"session-pool-link: copy {row['file']} -> {row['runtime']}")
    else:
        os.symlink(os.path.abspath(src), dst)
        print(f"session-pool-link: link {row['file']} -> {row['runtime']}")
    linked += 1

if linked == 0:
    raise SystemExit("session-pool-link: no session files found in repo root")
print(f"session-pool-link: ok ({linked} sessions)")
PY
