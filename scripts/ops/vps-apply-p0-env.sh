#!/usr/bin/env bash
# Merge BidShard P0 keys into VPS .env (idempotent set_env per key).
#
# Usage:
#   bash scripts/ops/vps-apply-p0-env.sh
#
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
# shellcheck disable=SC1091
source "$ROOT/scripts/lib/vps_ssh.sh"

vps_load_config "$ROOT"
if ! vps_ssh_require "$ROOT"; then
	exit 1
fi

vps_ssh "set -euo pipefail
cd '${VPS_REMOTE_DIR}'
set_env() {
  k=\"\$1\"; v=\"\$2\"
  if grep -q \"^\${k}=\" .env 2>/dev/null; then
    sed -i \"s|^\${k}=.*|\${k}=\${v}|\" .env
  else
    echo \"\${k}=\${v}\" >> .env
  fi
}

# BidShard ICP profile (see config/env/.env.bidshard-icp.example)
# budget: gemini RPM cap disables warm embed prescan/cluster (incompatible with prod accept-quality gate).
set_env PARSER_SEED_PROFILE budget
# Hot poll: webpain + reviews only. No serp/reddit/forum/github (SEO noise, 429, 403).
# SERP discovery: PARSER_BG_WORKER jobs serp_forum_threads + serp_telegram_catalog.
# Buyer pain: cron telegram scrape (telegram-pain-cron.sh) + TELEGRAM_ALERT_*.
set_env PARSER_SOURCE 'forum,serp,jobboard,tgweb,webpain,reviews,discord'
set_env PARSER_AUTO_DISCOVER true
set_env DISCORD_AUTO_DISCOVER_CHANNELS true
set_env DISCORD_JOIN_ENABLED true
set_env DISCORD_JOIN_DAILY_LIMIT 5
set_env PARSER_BG_DISCORD_DISCOVER_INTERVAL 24h
set_env PARSER_BG_DISCORD_CHANNEL_DISCOVER_INTERVAL 6h
set_env PARSER_GITHUB_ENABLED false
set_env PARSER_SOURCE_CONCURRENCY 0
set_env PARSER_WORKERS 8
set_env PARSER_POLL_SEC 300
set_env PARSER_PROXY_LIST_FILE config/env/proxy.list
set_env PARSER_PROXY_SOURCES 'forum,tgweb,lander,webpain,serp,reviews'
set_env PARSER_PROXY_DAILY_MB_CAP 350
set_env PARSER_EXPORT_JSON /app/data/export/leads.jsonl
set_env PARSER_EXPORT_JSON_HOST data/export/leads.jsonl
set_env TELEGRAM_PREFILTER true
set_env TELEGRAM_GEO_HEURISTIC true
# Cron-only MTProto (no parser-telegram-realtime container).
set_env TELEGRAM_REALTIME 0
set_env TELEGRAM_REALTIME_BACKFILL 0
set_env TELEGRAM_SESSION_ROLE hot
set_env TELEGRAM_LEASE_ENABLED 1
set_env TELEGRAM_LEASE_TTL_SEC 300
set_env TELEGRAM_LEASE_MAX_CHATS 15
set_env TELEGRAM_HISTORY_CHUNK_SIZE 100
set_env TELEGRAM_HISTORY_CHUNK_DELAY_MIN 1.5
set_env TELEGRAM_HISTORY_CHUNK_DELAY_MAX 3.5
set_env TELEGRAM_DISCUSSION_SCRAPE true
# Global SearchGlobal micro-dose: OFF on VPS by default (high FloodWait risk).
# To enable after soak: uncomment and redeploy parser-telegram sidecar.
# set_env TELEGRAM_GLOBAL_SEARCH 1
# set_env TELEGRAM_GLOBAL_SEARCH_LIMIT 3
# set_env TELEGRAM_GLOBAL_SEARCH_DAILY_LIMIT 3
# set_env TELEGRAM_GLOBAL_SEARCH_UTC_HOURS 2-6
set_env TELEGRAM_ALERT_ENABLED 1
set_env TELEGRAM_ALERT_CHANNEL '-1004489522664'
set_env CRM_TELEGRAM_ALLOWED_CHAT_IDS '-5167737291,-1004489522664,-1004328627993,6995552007'
set_env CRM_TELEGRAM_LEAD_NOTIFY true
set_env CRM_TELEGRAM_LEAD_NOTIFY_CHAT_IDS '-1004489522664'
set_env CRM_TELEGRAM_LEAD_NOTIFY_MIN_SCORE 50
set_env CRM_TELEGRAM_LEAD_NOTIFY_MIN_SCORE_NON_TELEGRAM 70
set_env PARSER_ACCEPT_MIN_SCORE 70
set_env PARSER_TELEGRAM_ACCEPT_MIN_SCORE 50
set_env PARSER_BG_WORKER true
# Telegram scrape runs via VPS cron (telegram-pain-cron.sh), not parser bgworker.
set_env PARSER_BG_TELEGRAM false
set_env PARSER_BG_SERP_TELEGRAM_MIN 60
set_env PARSER_SERP_TELEGRAM_DORK_MAX 24
set_env PARSER_BG_TELEGRAM_DISCOVER_MIN 360
set_env PARSER_BG_TELEGRAM_SCRAPE_MIN 30
# Gemini off: raw keyword-scored leads -> CRM webhook -> Telegram cards.
set_env GEMINI_API_KEY ''
set_env PARSER_GEMINI_DEFER false
set_env PARSER_ICP_CLASSIFY false
set_env PARSER_ICP_CLASSIFY_TGWEB false
set_env PARSER_INTENT_CLASSIFY false
set_env PARSER_GEO_CLASSIFY false
set_env PARSER_ENTITY_HEAT_ENABLED true
set_env PARSER_ENTITY_GEMINI_ENABLED false
set_env PARSER_CHANNEL_TRIAGE false
set_env CRM_ENGAGE_PRIORITY_MIN 70
set_env PARSER_CRM_WEBHOOK true
set_env PARSER_CRM_WEBHOOK_AFTER_ANALYSIS false
set_env PARSER_LEAD_STATUS_ENABLED true
set_env PARSER_CRM_WEBHOOK_URL 'http://127.0.0.1:8080/v1/leads'
# Parser must send the same Bearer token crm-bot expects (CRM_WEBHOOK_SECRET).
crm_webhook_secret=\$(grep -E '^CRM_WEBHOOK_SECRET=' .env 2>/dev/null | tail -1 | cut -d= -f2- || true)
if [[ -n \"\$crm_webhook_secret\" ]]; then
  set_env PARSER_CRM_WEBHOOK_SECRET \"\$crm_webhook_secret\"
else
  echo 'warn: CRM_WEBHOOK_SECRET missing; PARSER_CRM_WEBHOOK_SECRET not synced' >&2
fi
set_env MONGO_URI 'mongodb://127.0.0.1:27017'
set_env REDDIT_SUBREDDITS 'affiliatemarketing,media_buying,adops,PPC,juststart'
set_env REDDIT_QUERIES 'voluum alternative;keitaro alternative;postback failing;postback timeout;tracker migration;self-hosted tracker;click id not found;keitaro too expensive;binom stuck;capi error;upstream timed out;502 bad gateway;tracker numbers do not match;parallel pilot'
set_env REDDIT_MAX_RESULTS 50
set_env GITHUB_SEARCH_QUERIES 'keitaro postback;voluum alternative;binom postback;orbitra postback;traffoflex;self-hosted tracker migration;click id not found;postback timeout'
# Proxy list permissions (parser UID 10001)
if [[ -f config/env/proxy.list ]]; then
  chown 10001:10001 config/env/proxy.list
  chmod 640 config/env/proxy.list
fi

mkdir -p data/export

echo 'P0 env keys applied'
grep -E '^(PARSER_SOURCE|PARSER_EXPORT_JSON|TELEGRAM_REALTIME|TELEGRAM_REALTIME_BACKFILL|TELEGRAM_REALTIME_ROLE_FILTER|TELETHON_IPC_SOCKET|TELETHON_IPC_FORMAT|TELEGRAM_ALERT_ENABLED|PARSER_PROXY_LIST_FILE|PARSER_CRM_WEBHOOK)=' .env
"

printf 'vps-apply-p0-env: ok\n'
