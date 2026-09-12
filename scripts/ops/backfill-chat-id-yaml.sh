#!/usr/bin/env bash
# P0 ops: export chat_id rows from crawler.db for top pain channels (yaml backfill hints).
#
# Usage:
#   bash scripts/ops/backfill-chat-id-yaml.sh
#   bash scripts/ops/backfill-chat-id-yaml.sh data/runtime/crawler.db 20

set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
DB="${1:-$ROOT/data/runtime/crawler.db}"
LIMIT="${2:-20}"

if [[ ! -f "$DB" ]]; then
	echo "backfill-chat-id-yaml: missing db $DB" >&2
	exit 1
fi

sqlite3 -header -column "$DB" <<SQL
SELECT username, chat_id, title, pain_hits_30d
FROM telegram_channels
WHERE enabled = 1
  AND chat_id IS NOT NULL
ORDER BY pain_hits_30d DESC, updated_at DESC
LIMIT ${LIMIT};
SQL

cat <<'EOF'

# Paste into config/sources.telegram.yaml when pinning hot chats:
#   - name: example
#     username: example
#     chat_id: -1001234567890
#     geo: global
#     role: buyer_supergroup
#     enabled: true
EOF
