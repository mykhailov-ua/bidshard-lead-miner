#!/usr/bin/env bash
# Warm-path ops snapshot: pending leads, DLQ rows, rescan env.
# Replay: stale pending leads re-queue automatically (WARM_ANALYSIS_PENDING_SCAN_INTERVAL).
# DLQ rows are audit-only; fix root cause then wait for rescan or restart parser.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

if [[ -f "$ROOT/.env" && -r "$ROOT/.env" ]]; then
	set -a
	# shellcheck disable=SC1091
	source "$ROOT/.env"
	set +a
fi

MONGO_URI="${MONGO_URI:-mongodb://127.0.0.1:27017}"
MONGO_DB="${MONGO_DB:-parser}"
LEADS_COLL="${PARSER_MONGO_COLLECTION:-leads}"
DLQ_COLL="${WARM_ANALYSIS_DLQ_COLLECTION:-warm_analysis_dlq}"

printf 'warm-path-status: MONGO_URI=%s db=%s leads=%s dlq=%s\n' \
	"$MONGO_URI" "$MONGO_DB" "$LEADS_COLL" "$DLQ_COLL"
printf 'warm-path-status: WARM_ANALYSIS_RETRY_MAX=%s PENDING_SCAN=%s PENDING_STALE=%s\n' \
	"${WARM_ANALYSIS_RETRY_MAX:-3}" \
	"${WARM_ANALYSIS_PENDING_SCAN_INTERVAL:-5m}" \
	"${WARM_ANALYSIS_PENDING_STALE:-1h}"

if ! command -v mongosh >/dev/null 2>&1; then
	printf 'warm-path-status: WARN mongosh missing; install for pending/DLQ counts\n' >&2
	exit 0
fi

mongosh "$MONGO_URI/$MONGO_DB" --quiet --eval "
const leads = db.getCollection('${LEADS_COLL}');
const dlq = db.getCollection('${DLQ_COLL}');
const pending = leads.countDocuments({analysis_status: 'pending'});
const done = leads.countDocuments({analysis_status: 'done'});
const failed = leads.countDocuments({analysis_status: {\$in: ['failed', 'geo_rejected']}});
print('pending=' + pending + ' done=' + done + ' failed=' + failed);
const recent = dlq.find().sort({ts: -1}).limit(5).toArray();
if (recent.length) {
  print('dlq_recent:');
  recent.forEach(r => print('  ' + r.ts + ' hash=' + r.hash_id + ' attempts=' + r.attempts + ' err=' + (r.error || '').slice(0,80)));
} else {
  print('dlq_recent: (empty)');
}
"

printf 'warm-path-status: rescan re-queues pending leads older than WARM_ANALYSIS_PENDING_STALE every WARM_ANALYSIS_PENDING_SCAN_INTERVAL\n'
printf 'warm-path-status: after GEMINI_MODEL fix, expect pending to drop within 2 warm intervals (PARSER_WARM_ANALYZE_SEC)\n'
