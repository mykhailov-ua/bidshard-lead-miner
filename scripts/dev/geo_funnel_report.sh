#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

MONGO_URI="${MONGO_URI:-mongodb://127.0.0.1:27017}"
MONGO_DB="${MONGO_DB:-parser}"
COLL="${PARSER_MONGO_COLLECTION:-leads}"

if ! command -v mongosh >/dev/null 2>&1; then
	echo "ERROR: mongosh required" >&2
	exit 1
fi

echo "== lead funnel (collection=${COLL}, db=${MONGO_DB}) =="
mongosh "$MONGO_URI/$MONGO_DB" --quiet --eval "
const c = db.getCollection('${COLL}');
print('total leads: ' + c.countDocuments({}));
print('');
print('analysis_status:');
c.aggregate([
  {\$group: {_id: '\$analysis_status', n: {\$sum: 1}}},
  {\$sort: {n: -1}}
]).forEach(d => print('  ' + (d._id || '(missing)') + ': ' + d.n));
print('');
print('status (CRM):');
c.aggregate([
  {\$group: {_id: '\$status', n: {\$sum: 1}}},
  {\$sort: {n: -1}}
]).forEach(d => print('  ' + (d._id || '(missing)') + ': ' + d.n));
print('');
print('geo_country top 10:');
c.aggregate([
  {\$match: {geo_country: {\$exists: true, \$ne: ''}}},
  {\$group: {_id: '\$geo_country', n: {\$sum: 1}}},
  {\$sort: {n: -1}},
  {\$limit: 10}
]).forEach(d => print('  ' + d._id + ': ' + d.n));
"

echo ""
echo "== junk geo rejects (collection=${COLD_JUNK_COLLECTION:-junk_leads}) =="
mongosh "$MONGO_URI/$MONGO_DB" --quiet --eval "
const junk = db.getCollection('${COLD_JUNK_COLLECTION:-junk_leads}');
if (!junk) { print('no junk collection'); quit(); }
['geo_reject','geo_gemini_reject'].forEach(r => {
  const n = junk.countDocuments({reason: r});
  print(r + ': ' + n);
});
" 2>/dev/null || true

echo ""
echo "Tip: parser_junk_total{reason=\"geo_gemini_reject\"} on PARSER_METRICS_ADDR when enabled"
