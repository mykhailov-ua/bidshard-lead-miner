#!/usr/bin/env bash
# Weekly entity heat negative-selection report (HEAT-P5-01).
# Writes data/suggestions/entity_heat_YYYYMMDD.txt and entity_pain_YYYYMMDD.txt.
#
# Usage: make entity-heat-report
# Requires: mongosh, Mongo with ENTITY_COLLECTION (default entities).
#
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

MONGO_URI="${MONGO_URI:-mongodb://127.0.0.1:27017}"
MONGO_DB="${MONGO_DB:-parser}"
ENTITY_COLL="${ENTITY_COLLECTION:-entities}"
SUGGEST_DIR="${GEMINI_DISCOVER_DIFF_DIR:-data/suggestions}"
STAMP="$(date -u +%Y%m%d)"
REPORT_FILE="${SUGGEST_DIR}/entity_heat_${STAMP}.txt"
PAIN_FILE="${SUGGEST_DIR}/entity_pain_${STAMP}.txt"

if ! command -v mongosh >/dev/null 2>&1; then
	echo "ERROR: mongosh required" >&2
	exit 1
fi

mkdir -p "$SUGGEST_DIR"

mongosh "$MONGO_URI/$MONGO_DB" --quiet --eval "
const coll = db.getCollection('${ENTITY_COLL}');
const total = coll.countDocuments({});
print('entity_heat_report');
print('collection=' + '${ENTITY_COLL}' + ' db=' + '${MONGO_DB}' + ' total=' + total);
print('');

print('== hot tier + low actor_confidence (<0.5) ==');
coll.find({
  heat_tier: {\$in: ['hot', 'blazing']},
  actor_confidence: {\$lt: 0.5},
}).sort({heat_score: -1, last_seen: -1}).limit(20).forEach(d => {
  print('  ' + d.entity_id + ' heat=' + (d.heat_score||0).toFixed(1) +
    ' conf=' + (d.actor_confidence||0).toFixed(2) +
    ' pain=' + (d.unified_pain || '').slice(0, 60));
});
print('');

print('== 3+ source families but cold tier (under-hot) ==');
coll.find({
  source_families: {\$exists: true},
  \$expr: {\$gte: [{\$size: "\$source_families"}, 3]},
  heat_tier: 'cold',
}).sort({sighting_count: -1, last_seen: -1}).limit(20).forEach(d => {
  const fam = (d.source_families || []).join(',');
  print('  ' + d.entity_id + ' sightings=' + (d.sighting_count||0) +
    ' families=' + fam);
});
print('');

print('== needs_review (split_recommended from Gemini) ==');
const needsReview = coll.countDocuments({needs_review: true});
print('count=' + needsReview);
coll.find({needs_review: true}).sort({heat_score: -1}).limit(10).forEach(d => {
  print('  ' + d.entity_id + ' tier=' + (d.heat_tier||'') +
    ' pain=' + (d.unified_pain || '').slice(0, 50));
});
print('');

print('== forum_user key hit rate ==');
const forumFamily = coll.countDocuments({source_families: 'forum'});
const forumUserKey = coll.countDocuments({
  alias_keys: {\$elemMatch: {\$regex: /^forum_user:/}},
});
const warriorFamily = coll.countDocuments({source_families: 'warrior'});
const warriorUserKey = coll.countDocuments({
  alias_keys: {\$elemMatch: {\$regex: /^forum_user:warriorforum:/}},
});
const forumSightings = coll.aggregate([
  {\$match: {source_families: {\$in: ['forum', 'warrior']}}},
  {\$group: {_id: null, n: {\$sum: "\$sighting_count"}}},
]).toArray();
const forumSightingTotal = forumSightings.length ? forumSightings[0].n : 0;
print('entities with forum family: ' + forumFamily);
print('entities with forum_user alias: ' + forumUserKey);
print('entities with warrior family: ' + warriorFamily);
print('entities with warriorforum forum_user: ' + warriorUserKey);
if (forumFamily + warriorFamily > 0) {
  const denom = forumFamily + warriorFamily;
  const numer = forumUserKey;
  print('forum_user key hit rate: ' + (100 * numer / denom).toFixed(1) + '% (' + numer + '/' + denom + ')');
}
print('forum/warrior sighting rows (sum): ' + forumSightingTotal);
print('');

print('== top unified_pain (for keyword feedback) ==');
coll.aggregate([
  {\$match: {unified_pain: {\$exists: true, \$ne: ''}}},
  {\$group: {_id: '\$unified_pain', n: {\$sum: 1}, maxHeat: {\$max: '\$heat_score'}}},
  {\$sort: {n: -1, maxHeat: -1}},
  {\$limit: 25},
]).forEach(row => {
  print('  ' + row.n + 'x heat_max=' + (row.maxHeat||0).toFixed(0) + ' | ' + row._id);
});
" | tee "$REPORT_FILE"

mongosh "$MONGO_URI/$MONGO_DB" --quiet --eval "
const coll = db.getCollection('${ENTITY_COLL}');
print('# entity unified_pain suggestions ${STAMP}');
print('# Review manually; merge approved phrases into data/keywords.json or config/discover.icp.json');
print('');
coll.aggregate([
  {\$match: {unified_pain: {\$exists: true, \$ne: ''}}},
  {\$group: {_id: '\$unified_pain', n: {\$sum: 1}, maxHeat: {\$max: '\$heat_score'}}},
  {\$sort: {n: -1, maxHeat: -1}},
  {\$limit: 50},
]).forEach(row => {
  print(row.n + '\t' + (row.maxHeat||0).toFixed(0) + '\t' + row._id);
});
" > "$PAIN_FILE"

echo ""
echo "Wrote $REPORT_FILE"
echo "Wrote $PAIN_FILE"
echo "Tip: crm-bot db entity-split --id ENTITY --hash HASH --yes for false merges"
