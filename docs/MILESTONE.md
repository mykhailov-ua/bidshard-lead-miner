# Milestone: Lead quality and MTProto-first funnel

Status snapshot: **2026-09-12** (post P0 deploy on VPS `hostiq`).

Goal: stop CRM/Telegram noise from SERP and keyword-only scoring; make **MTProto chat listener + historical export** the primary buyer discovery path.

Related: [OUTREACH.md](OUTREACH.md), [ICP.md](ICP.md), [OPS.md](OPS.md), [POST_MORTEM.md](POST_MORTEM.md).

---

## Executive summary

| Problem | Root cause | Fix direction |
|---------|------------|---------------|
| RedTrack/CloakingTool score 90+ | SERP SEO hits keyword registry | SERP = discovery only, not CRM accept |
| `telegram:@serp:domain` | Domain fallback as contact | Synthetic contact reject (deployed) |
| BHW snippet score 74 | Forum SERP without `forum:user` | Route to forum registry + fetch |
| 0 sales from alerts | Wrong primary source + no valid contact | MTProto realtime + pain AND-gate |
| PullPush/forum dead | 429 / CF 403 in hot poll | Offline batch or bgworker only |

**Strategic pivot:** one Telethon account on 30-40 **buyer supergroups** beats DuckDuckGo + PullPush for ICP v1.

---

## Target architecture

```
                    DISCOVERY (no CRM accept)
                    -----------------------
SERP dorks ------> discovered_forum_threads.json
              \--> discovered_telegram_channels.json
              \--> discovered_telegram_domains.json (tgweb seeds)

                    PRIMARY FUNNEL (CRM + alerts)
                    -----------------------------
Telethon session
  +-- realtime listener (NewMessage) ----> pain alert card (@user, link, date)
  |                                  \--> NDJSON --> Go processor --> CRM webhook --> Telegram lead card
  +-- history-export (batch, 6-12 mo) --> CSV/NDJSON --> manual outreach queue

                    SECONDARY / DEFERRED
                    --------------------
forum adapter (fetch thread HTML) --> forum:user/{author} + PostedAt
reddit archive (offline, 1 req/2s) --> reddit:u/...
webpain/reviews (low volume, strict gates)
```

**Accept rule (hard):** no lead without reachable contact (`telegram:@user`, `forum:user/...`, email, skype).

---

## Milestone map

| ID | Name | Priority | Effort | Status |
|----|------|----------|--------|--------|
| M0 | P0 deploy and prescan fixes | P0 | 1d | **Done** (2026-09-12) |
| M1 | Stop SERP hot-path pollution | P0 | 0.5d | **Done** (env: webpain,reviews only) |
| M2 | MTProto realtime as primary funnel | P1 | 2-3d | Partial (deployed; soak: `make vps-telegram-realtime-soak`) |
| M3 | Pain alert quality (AND-gate) | P1 | 1d | **Done** (deployed VPS) |
| M4 | Chat list curation (supergroups) | P1 | 2d | **Done** (40 buyer supergroups) |
| M5 | Historical mining export | P2 | 2-3d | Partial (CLI + `make vps-history-export ARGS="--detach"`; long run on VPS) |
| M6 | Forum thread fetch path | P2 | 3-5d | Partial (bgworker wired; 403 until proxy creds) |
| M7 | Reddit offline archive | P3 | 2d | Partial (CLI + `make reddit-offline-archive`; VPS run pending) |
| M8 | Scoring / ICP without Gemini | P2 | 2d | **Done** (MIN_SCORE 70 + telegram buyer gate) |
| M9 | Observability and soak gates | P2 | 1-2d | Partial (`icp-soak-report` M9 metrics; manual review pending) |
| M10 | Gemini or Ollama ICP classify | P3 | 2-5d | Deferred (quota) |
| M11 | Crypto-gray / antifraud ICP heuristics | P1 | 2-3d | **Done** (keywords + discover seeds + Go boost) |

---

## M0 - P0 deploy and prescan fixes [DONE]

### What was broken

- CRM webhook 401 (`PARSER_CRM_WEBHOOK_SECRET` missing)
- Keyword accepts: SEO listicles, competitor domains, GitHub code paste, `@serp:` contacts
- Hot poll burning on reddit 429 / forum 403

### What was shipped

| Area | Change |
|------|--------|
| Webhook | `PARSER_CRM_WEBHOOK_SECRET` synced from `CRM_WEBHOOK_SECRET` |
| Prescan | `IsSyntheticContact`, listicle -200, SEO marketing context drop |
| SERP crawler | No emit without real handle; forum hits -> registry |
| Processor | `RejectNewbieNoBudget`, `SerpForumSnippetOnly`, competitor blacklist |
| Env | `PARSER_SOURCE=serp,webpain,reviews`, `MIN_SCORE=50`, Gemini off |
| Deploy | `make vps-deploy-p0` on VPS |

### Verify

```bash
bash scripts/ops/p0-secrets-check.sh --vps
ssh -p 2222 root@hostiq 'cd /opt/lead-intent-processor && docker compose ps'
ssh -p 2222 root@hostiq 'cd /opt/lead-intent-processor && docker compose logs parser --tail 50'
```

### Do not

- Run `crm-replay-export.sh` blindly (re-sends old junk JSONL to Telegram).

---

## M1 - Stop SERP hot-path pollution [P0]

### Problem

SERP answers SEO pages, not chat messages. Even with prescan fixes, hot poll still wastes proxy budget and creates alert fatigue.

### Actions

1. **VPS env:** set `PARSER_SOURCE=webpain,reviews` or empty (discovery only via bgworker).
2. Keep SERP on bgworker jobs only:
   - `serp_forum_threads` -> `discovered_forum_threads.json`
   - `serp_telegram_catalog` -> channel registry
3. Remove generic dorks from hot `serp.Collect` emit path (already skips forum hosts in code).
4. Document in `.env.example` and `vps-apply-p0-env.sh`.

### Success criteria

- `scan round` logs: `source=serp` not in hot poll OR `accepted=0` from serp for 48h
- No new leads with `source=serp:*` in Mongo after deploy
- `discovered_forum_threads.json` grows weekly

### Files

- `scripts/ops/vps-apply-p0-env.sh`
- `internal/sources/serp/serp.go`
- `internal/app/bgworker_serp.go`

---

## M2 - MTProto realtime as primary funnel [P1]

### What already exists

| Component | Path |
|-----------|------|
| Realtime listener | `sources/telegram/realtime.py` |
| VPS container | `docker-compose.telegram-realtime.yaml` (`parser-telegram-realtime`) |
| Pain alerts | `sources/telegram/alert_dispatcher.py` |
| Chat config | `config/sources.telegram.yaml` |
| NDJSON ingest | Go `internal/telethon/` + processor |
| Backfill | `TELEGRAM_REALTIME_BACKFILL=300` (too low for mining) |

### Actions

1. Confirm realtime healthy:
   ```bash
   docker compose -f docker-compose.telegram-realtime.yaml ps
   docker compose logs parser-telegram-realtime --tail 100 | grep -E 'listening|pain alert'
   ```
2. Wire **one path**: pain alert content = CRM lead card fields (author, link, posted_at, snippet).
3. Raise backfill cautiously: `TELEGRAM_REALTIME_BACKFILL=1000` (watch FloodWait in logs).
4. Ensure NDJSON from realtime reaches parser ingest (not alert-only orphan path).
5. Set `CRM_TELEGRAM_LEAD_NOTIFY_MIN_SCORE` low for `telegram:*` only OR bypass score for alert-sourced leads.

### Success criteria

- >= 3 pain alerts/day with real `@username` and `t.me/.../msg_id` link
- >= 1 CRM lead/week from `telegram:@*` with buyer voice (manual review)
- Zero `@serp:` contacts in new leads

### Verify (M2 soak)

```bash
# VPS report (warn by default; M2_STRICT=1 exits non-zero on gate fail)
bash scripts/ops/vps-telegram-realtime-soak.sh
SOAK_HOURS=48 M2_STRICT=1 bash scripts/ops/vps-telegram-realtime-soak.sh

# Checks: parser-telegram-realtime Up, listening log, pain alert tags
# (tracker_pain|crypto_gray), telethon ipc connected (msgpack ingest),
# telegram lead accepted logs, contact_valid_rate on export JSONL.
```

### Risks

- Session lock conflict between `parser` bgworker scrape and realtime (shared `telethon.session`)
- FloodWait on backfill increase - tune `TELEGRAM_HISTORY_CHUNK_DELAY_*`

---

## M3 - Pain alert quality (AND-gate) [P1]

### Problem

Current alert: OR on `TRACKER_PAIN_HINTS` (`keitaro` alone fires). Too noisy.

### Target rule

```
(trackers: keitaro|binom|voluum|redtrack|postback|tracker)
AND
(pain: ошибка|упал|oom|502|база|постбек|тормозит|failing|crash|timeout|migration)
AND
author has @username (not channel broadcast)
```

### Actions

1. Add `message_has_tracker_pain()` in `sources/telegram/pain.py` (AND logic).
2. Use in `should_alert_on_emit()` and optionally `should_emit_message()` for CRM path.
3. Skip channel broadcast without `reply_to` (mirror Go `TelegramChannelBroadcastReject`).
4. Tests in `sources/telegram/test_pain.py`, `test_alert_dispatcher.py`.

### Files

- `sources/telegram/pain.py`
- `sources/telegram/alert_dispatcher.py`
- `sources/telegram/prefilter.py`

### Success criteria

- Alert volume drops 50-70% vs baseline week
- Manual review: >= 80% alerts are outreach-worthy (spreadsheet 20 samples)

---

## M4 - Chat list curation [DONE]

### Problem

`config/sources.telegram.yaml` mixes vendor channels (`keitaro_tracker`), media teams (`frbs_team`), and digests. Buyer pain lives in **supergroups**, not channel descriptions.

### Shipped (2026-09-12)

| Item | Count / detail |
|------|----------------|
| `buyer_supergroup` enabled | **40** chats in `config/sources.telegram.yaml` |
| `vendor_support` + `intel_only` + `supply` | **28** rows with `enabled: false` |
| Realtime filter | `TELEGRAM_REALTIME_ROLE_FILTER=buyer_supergroup` |
| Discover example registry | `config/discovered_telegram_channels.example.json` |
| Loader | `role` validated in `sources/telegram/config.py` |

New buyer rows (8) from discover seeds / cross-mention example registry: `aff_lead`, `aff_net`, `affnet`, `affiliate_igaming`, `stmaffiliate`, `affiliate_latam_en`, `affiliate_latam`, `igaming_acquisition`. Moved `soltrending` to `intel_only` (block handle).

### Actions (remaining)

1. Copy VPS `data/runtime/discovered_telegram_channels.json` into yaml as more buyer rows (target 40+).
2. Join policy: `sources/telegram/join_policy.py` - rate limits, invite-only queue.
3. Optional: separate yaml `config/sources.telegram.buyer.yaml` for outreach chats.

### Seed categories (examples)

| Type | Examples | Use |
|------|----------|-----|
| Tracker user groups | Binom/Keitaro user chats | pain, migration |
| Media buying teams | private team supergroups | scale buyers |
| Cloak/infra chats | nginx/502/postback threads | technical buyers |
| Vendor channels | `@keitaro_tracker` | intel only, no CRM |

### Success criteria

- 30+ enabled chats, >= 70% classified `buyer_supergroup`
- `pain_hits_30d` top channels in cursor DB are buyer groups (not vendor)

---

## M5 - Historical mining export [P2]

### Problem

Realtime only sees new messages + 300 msg backfill. Need 6-12 month archive for outreach backlog.

### Actions

1. New CLI subcommand:
   ```bash
   parser telegram history-export \
     --since 2025-03-01 \
     --chats config/sources.telegram.yaml \
     --out data/export/tg_history_pain.ndjson
   ```
2. Reuse `history_chunk.iter_messages_chunked` with date filter (`offset_date`).
3. Local filter: tracker AND pain regex (same as M3).
4. Output columns: `username`, `user_id`, `posted_at`, `chat`, `message_id`, `link`, `text`.
5. Optional CSV export for bizdev.
6. Store in `historical_prospects` collection (not `leads`) to avoid polluting CRM.

### Effort

- Dev: 2-3 days
- Runtime: 2-6 hours crawl (40 chats x 6 mo, chunked delays)

### Success criteria

- One export run: 200-500 rows passing AND-filter
- >= 50% rows have clickable `@username`
- Bizdev can outreach without opening SERP

### Files to add

- `cmd/parser/telegram_history_export.go` (or Python entry in `sources/telegram/`)
- `sources/telegram/history_export.py`
- `docs/OUTREACH.md` section for historical script

---

## M6 - Forum thread fetch [P2, partial]

### Problem

SERP finds BHW/AffiliateFix URLs but fetch returns 403. Snippet-only leads have no `forum:user` and no date.

### What exists

- `internal/sources/forum/adapter.go` - `ParsePostsFromHTML`, `forum:user/{author}`, `PostedAt`
- `discovered_forum_threads.json` registry from `serp_forum_threads` bgworker
- `forum_crawl` bgworker job (`PARSER_BG_FORUM_CRAWL`, default 6h) - not hot poll
- Processor gate: `forum_no_user_contact` when `forum:*` lacks `forum_user` handle
- `scripts/ops/forum-bg-smoke.sh` - one-thread fetch from registry

### Shipped (2026-09-12)

| Item | Detail |
|------|--------|
| bgworker | `forum_crawl` via `runForumCrawlOnce` + `forum.RunBGCrawl` proxy gate |
| Hot poll | `forum` stays out of `PARSER_SOURCE` (vps-apply-p0-env) |
| Proxy scope | Forum fetch uses `ProxyURLsForSource("forum")` only |
| Processor | Reject `forum:*` without `forum_user` contact type |
| Smoke | `bash scripts/ops/forum-bg-smoke.sh` (needs registry + proxy) |

### Remaining

1. Residential proxy creds on VPS (`PARSER_PROXY_LIST` or `PARSER_PROXY_LIST_FILE`).
2. Stale gate: reject `PostedAt` older than 18 months.
3. Headless fallback if proxy still 403 on BHW.

### Blocker

- **403 on datacenter VPS without residential proxy.** `forum.RunBGCrawl` skips when `forum` is in `PARSER_PROXY_SOURCES` but proxy list is empty. With proxy set, CF may still 403 if creds are datacenter or burned - check `parser_proxy_cf_block_total{source="forum"}` and `forum crawl skipped` logs.
- Operator action: sync `config/env/proxy.list` to VPS (`make vps-sync-proxy`), then `bash scripts/ops/forum-bg-smoke.sh` on VPS.

### Verify

```bash
# Unit proof (no network)
go test ./internal/pipeline/... ./internal/filter/... ./internal/sources/forum/...

# One-thread live smoke (needs proxy + registry)
bash scripts/ops/forum-bg-smoke.sh
```

### Success criteria

- `forum` adapter emits > 0 raw/week
- >= 1 accept/week with `forum:user/*` and pain context

---

## M7 - Reddit offline archive [P3]

### Problem

PullPush 429 in hot poll (14k+ errors). Historical gold exists but not reachable live.

### Actions

1. Standalone script (not `internal/sources/reddit/crawler.go` poll):
   - 1 request / 2s, exponential backoff on 429
   - Arctic Shift fallback URL (already in code, unwired)
   - Subs: `affiliatemarketing`, `media_buying`, `adops`, `juststart`
   - Date range: 2025-01-01 .. now
2. Output JSONL: `author`, `body`, `created_utc`, `permalink`
3. Bulk ingest or separate outreach queue

### Do not

- Re-add `reddit` to `PARSER_SOURCE` hot poll without token pool + backoff.

---

## M8 - Scoring and ICP without Gemini [P2]

### Problem

Gemini off -> keyword score only -> false positives even after prescan.

### Layers (keep all)

| Layer | Mechanism | Status |
|-------|-----------|--------|
| L1 | Prescan: synthetic contact, SEO, newbie, supply | Deployed |
| L2 | Spend gate + newbie cap even with competitor mention | Deployed |
| L3 | `MIN_SCORE=50` + require buyer voice for SERP/forum | Partial |
| L4 | Gemini/Ollama ICP | Off |

### Actions

1. Raise `CRM_TELEGRAM_LEAD_NOTIFY_MIN_SCORE` to **70** for non-telegram sources.
2. For `telegram:*`: accept if `validate.HasCommercialPainIntent` OR pain alert fired.
3. Add negative keyword weights in `data/keywords.json` for listicle phrases.
4. Optional: Ollama local classify (`internal/gemini/ollama.go` exists) for ICP only.

---

## M9 - Observability and soak gates [P2]

### Metrics to track

```
accept_rate           = accepted / raw
contact_valid_rate    = leads with telegram|forum_user|email / accepted
synthetic_reject_rate = synthetic_contact + serp_forum_snippet / raw
alert_per_day         = pain alerts sent
alert_to_crm_ratio    = alerts / CRM accepts
source_breakdown      = by source family
top_reject_reasons    = from round stats
```

### Actions

1. Extend `scripts/ops/icp-soak-report.sh` with contact_valid_rate and synthetic rejects. **Done** (M9 block + `scripts/lib/soak_gate.sh` helpers).
2. Prometheus alerts: `parser_accept_rate < 0.001` with `raw > 1000` (noise crawl).
3. Weekly manual review: 10 random accepts + 10 random alerts -> spreadsheet.

### Commands

```bash
# Local export + optional local metrics (PARSER_METRICS_ADDR=:9465)
bash scripts/ops/icp-soak-report.sh

# Pull VPS export + metrics + pain alert log proxy
SOAK_HOURS=168 bash scripts/ops/icp-soak-report.sh --vps

# Thresholds: config/env/.env.soak.example
bash scripts/ops/vps-status.sh
curl -s localhost:9465/metrics | grep parser_
```

M9 report fields: `contact_valid_rate`, `synthetic_reject_rate` (Prometheus proxy),
`alert_per_day` (pain alert log grep), `source_breakdown` (family + raw source id).

---

## M11 - Crypto-gray / antifraud ICP heuristics [P1]

### Who this targets

Buyers who **pay and think in USDT**, run gray igaming/nutra/CPA, and feel **click/deposit quality** in money terms (shaving, bot fraud, scrubbing) - not SEO tourists or CPA network AM promos.

Aligns with BidShard antifraud + tracker stack. Distinct from generic "voluum alternative" newbie.

**ICP boundary (from [ICP.md](ICP.md)):** the **webmaster/affiliate inside** a crypto CPA network is yes; the **network sales AM** is no.

### Signal model (AND-gate for alerts / boost for scoring)

Do not fire on crypto keywords alone (too much seller noise). Prefer:

```
(infra_or_tracker) AND (crypto_or_payout OR antifraud_pain)
```

| Tier | Examples | Role |
|------|----------|------|
| **Crypto payout** | `trc20`, `erc20`, `usdt`, `crypto payout`, `crypto settlement`, `weekly usdt`, `daily usdt`, `no kyc`, `capitalist`, `pst` | Wallet / settlement context |
| **Tracker / cloak stack** | `keitaro`, `binom`, `zeustrack`, `hideclick`, `cloaking.house`, `fraudfilter`, `js fingerprint`, `maxmind`, `ipqualityscore`, `ipqs` | Infra buyer |
| **Antifraud pain** | `shaving`, `scrubbing`, `bot click`, `fake leads`, `auto-fill`, `trash deposit`, `cr drop`, `incentivized traffic`, `fraudulent traffic`, `balance frozen`, `network froze` | Pain = pays for clean click |
| **Vertical bleed** | `ftd`, `reg`, `deposit`, `nutra`, `igaming`, `gambling`, `cpi` + bot/fraud words | Gray vertical under attack |

**Boost (+score):** all three tiers in one message, or `(tracker) AND (antifraud_pain)`.

**Hard drop (keep):** seller consumables from `instant_drop` (`buy fp`, `farm accounts for sale`, `virtual card`) even if USDT mentioned.

**Soft drop:** newbie `can't afford` without spend/infra signals (`RejectNewbieNoBudget`).

### Where they live (chat sources for M4)

| Source type | Examples | Join as |
|-------------|----------|---------|
| Crypto CPA webmaster groups | USDT payout discussion, not official AM channel | supergroup |
| Private farm / agency cab setup chats | Capitalist, PST, virtual cards for **buying** ads | supergroup (not seller broadcast) |
| Tracker + cloak user chats | Keitaro/Binom/HideClick operators | supergroup |
| Anti-fraud / quality threads | shaving accusations, network holds | reply threads in buyer groups |

**Do not add:** pump/signal channels, `@partneroff_pro`-style AM cards, official network support bots.

Discover seeds: extend `config/discover.icp.json` `telegram_search` with `usdt payout affiliate`, `scrubbing network`, `keitaro bot traffic`.

### Implementation map (repo)

| Layer | File | Change |
|-------|------|--------|
| TG prefilter | `sources/telegram/prefilter.py` | `CRYPTO_PAYOUT_HINTS`, `ANTIFRAUD_PAIN_HINTS`, `has_crypto_gray_icp_signal()` |
| TG pain alerts | `sources/telegram/pain.py`, `alert_dispatcher.py` | AND-gate: tracker OR cloak + (crypto OR antifraud pain) |
| Go prescan boost | `internal/validate/crypto_icp.go` (new) | `HasCryptoGrayBuyerSignal()` for processor boost |
| Keywords | `data/keywords.json` | Weighted phrases, separate from generic `kw-crypto-cpa` |
| Negative | `instant_drop.go` / `prefilter.py` | No change to farm **seller** phrases; buyer voice in farm **chats** passes via pain AND-gate |
| Chat yaml | `config/sources.telegram.yaml` | Curated crypto/gray supergroups (manual) |
| History export | M5 filter | Same AND-regex in export script |

### Regex sketch (historical export / alerts)

```text
(trackers:  keitaro|binom|zeustrack|hideclick|cloaking|fraudfilter)
(payout:    trc20|erc20|usdt|crypto payout|crypto settlement|no kyc|capitalist)
(pain:      shaving|scrubbing|bot click|fake lead|auto.?fill|trash deposit|
            cr drop|incentivized|fraudulent traffic|balance frozen|ipqs|maxmind)
```

Fire when: `(trackers OR pain) AND (payout OR pain)` and author has `@username`.

### Success criteria

- Pain alerts: >= 30% of weekly alerts match crypto-gray AND-gate (tag in log)
- Manual review: >= 70% are media buyers / team leads (not AMs, not account sellers)
- At least 1 outreach/week mentions USDT settlement or network scrubbing in their own words

### Conflicts to watch

| Risk | Mitigation |
|------|------------|
| Farm chat = seller spam | `instant_drop` on sell phrases; require buyer question/pain |
| `usdt` in channel bio only | Ignore bio-only; require message body |
| CPA network AM | `RejectAffiliateNetworkSupply`, agency outreach reject |
| RU/BY geo | `geo_heuristic` + `PARSER_GEO` when Gemini returns |

---

## M10 - Gemini or Ollama ICP [P3, deferred]

### When

- Billing/quota fixed on Gemini, OR
- Ollama deployed on VPS with soak proving RPM stability

### Scope

- `PARSER_ICP_CLASSIFY=true` for `telegram:` and `forum:` only
- Keep SERP/forum snippet out of classify path entirely
- Do not block raw ingest on Gemini failure (defer queue)

---

## Priority execution order

```
Week 1 (now)
  [x] M0 P0 deploy
  [x] M1 Remove SERP from hot poll (env)
  [ ] M3 Pain AND-gate for alerts
  [ ] M2 Verify realtime + unify alert/CRM card

Week 2
  [x] M4 Curate buyer supergroups (29 enabled; expand from VPS discover registry)
  [ ] M11 Crypto-gray AND-gate in TG prefilter + keywords
  [ ] M8 MIN_SCORE 70 + telegram buyer voice gate
  [ ] M9 Soak report + manual review baseline

Week 3-4
  [ ] M5 history-export CLI + first 6mo run
  [ ] M6 Forum fetch (if proxy fixed)

Backlog
  [ ] M7 Reddit offline
  [ ] M10 Gemini/Ollama ICP
  [ ] M11 history-export regex pack (depends M5)
```

---

## What NOT to do

| Anti-pattern | Why |
|--------------|-----|
| Fix PullPush in 300s hot poll | 429 ban, zero yield |
| Accept SERP snippet as lead | No contact, no date, SEO noise |
| Score-only accept without contact gate | RedTrack problem repeats |
| `crm-replay-export` old JSONL | Re-spams Telegram with junk |
| Global SearchGlobal on VPS | FloodWait risk (keep off) |
| Add more SERP dorks to hot poll | More SEO competitors |
| Outreach to newbie "can't afford" | ICP mismatch, wastes bizdev time |

---

## Outreach readiness checklist

Before bizdev writes to a lead:

- [ ] Contact is `telegram:@user` or `forum:user/name` (not `@serp:`)
- [ ] `posted_at` known and < 18 months
- [ ] Snippet shows buyer voice (question/pain), not SEO/listicle
- [ ] Not vendor channel broadcast (team promo, AM card)
- [ ] Link to original message/thread exists
- [ ] For crypto-gray segment: infra + (USDT/payout OR antifraud pain) in their message, not keyword stuffing

Template: see [OUTREACH.md](OUTREACH.md) "historical pain" variant.

---

## Key env vars (reference)

| Var | Production target |
|-----|-------------------|
| `PARSER_SOURCE` | `webpain,reviews` or empty (no serp hot poll) |
| `PARSER_GITHUB_ENABLED` | `false` |
| `GEMINI_API_KEY` | empty until M10 |
| `CRM_TELEGRAM_LEAD_NOTIFY_MIN_SCORE` | `70` (non-TG); `50` for TG optional |
| `TELEGRAM_REALTIME` | `1` in realtime container only |
| `TELEGRAM_REALTIME_BACKFILL` | `300` -> `1000` after soak |
| `TELEGRAM_ALERT_ENABLED` | `1` |
| `TELEGRAM_ALERT_CHANNEL` | bizdev group id |
| `PARSER_CRM_WEBHOOK_SECRET` | synced with `CRM_WEBHOOK_SECRET` |
| `PARSER_BG_WORKER` | `true` (SERP discovery jobs) |

---

## Ops commands

```bash
# Deploy
make vps-deploy-p0

# Secrets
bash scripts/ops/p0-secrets-check.sh --vps

# Status
bash scripts/ops/vps-status.sh

# Parser one round
ssh -p 2222 root@hostiq 'cd /opt/lead-intent-processor && docker compose exec -T parser parser scan'

# Realtime logs
ssh -p 2222 root@hostiq 'cd /opt/lead-intent-processor && docker compose -f docker-compose.telegram-realtime.yaml logs --tail 100'

# Soak
bash scripts/ops/icp-soak-report.sh --vps
bash scripts/ops/vps-telegram-realtime-soak.sh
```

---

## Definition of done (program level)

The funnel is healthy when **all** are true for 7 consecutive days:

1. **Zero** new CRM leads with `serp:` contact or synthetic telegram handle.
2. **>= 5** pain alerts/week with valid `@username` + message link.
3. **>= 1** CRM accept/week from `telegram:@*` or `forum:user/*` (manual ICP ok).
4. **contact_valid_rate >= 95%** on accepts.
5. Bizdev confirms >= 2 outreach-worthy leads/week (subjective, logged in sheet).

Until M5 ships, criterion 3 may be alerts-only (not CRM) for historical backlog.
