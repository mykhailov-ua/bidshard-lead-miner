# Telethon session sharding and ICP hypotheses

Plan for scaling MTProto **without realtime** (cron scrape + pain alerts + batch export/discover) across multiple Telegram accounts.

Related: [docs/MILESTONE.md](docs/MILESTONE.md) (M2/M4/M5), [docs/ICP.md](docs/ICP.md), [docs/OUTREACH.md](docs/OUTREACH.md), [docs/POST_MORTEM.md](docs/POST_MORTEM.md), [config/sources.telegram.yaml](config/sources.telegram.yaml).

Hypotheses **H1-H14** below are experiments to run and pass/fail; status table at the bottom.

---

## Decision: no realtime

**We are not building or running 24/7 `NewMessage` realtime + IPC.**

| Instead | Why |
|---------|-----|
| Cron `scraper.py` / pain poll on curated chats | Enough for M3 pain AND-gate; lower ops cost |
| `history-export` on cold session | Backfill without stealing hot lock |
| `buyer-discover` + SERP on cold session | Channel pool growth |
| Go ingest from NDJSON batch only | No multi-UDS mux required for MVP |

Realtime container (`docker-compose.telegram-realtime.yaml`) is **deprecated for production**. Keep code for reference; do not invest in P1 multi-realtime compose until cron path proves alerts.

Pain alerts still fire from Python `alert_dispatcher` on **cron emit**, not live listener.

---

## Problem

One `telethon.session` resolves every enabled chat on startup via `ResolveUsername`. At 40+ `buyer_supergroup` rows this triggers **FloodWait** (hours). Even cron scrape skips chats that failed resolve until the next cycle.

Current constraints:

| Constraint | Where |
|------------|--------|
| Single session file | `data/runtime/telethon.session` |
| Exclusive session lock | `telethon.session.lock` (Go + Python) |
| Resolve by `@username` only | `resolve_chat_entity()` ignores `chat_id` in `crawler.db` |
| Dedup by contact hash | `sink.LeadHashID()` (not `chat_id` + `message_id`) |

**Consensus (Raft, etcd) is not required.** Coordination is a lease table or static hash shard.

---

## What "warming" (прогрев) means

**Account warming** = gradual, human-like use of a new Telegram MTProto account before automated high-volume operations.

Telegram rate-limits new or cold accounts harder than aged ones. Symptoms: short FloodWait on `ResolveUsername`, `JoinChannel`, `GetHistory`, bans on mass join.

### Warming checklist (per new session)

1. **Login once** via `parser telegram login --qr` (or phone); keep `TELEGRAM_API_ID` / `TELEGRAM_API_HASH` stable for that session file.
2. **Age 24-72h** with light manual use (read a few chats, no bot-like bursts).
3. **Join slowly**: 1-3 supergroups per day, not 20 on day one. Prefer invite links; avoid Global Search join storms.
4. **No parallel heavy jobs** on the same session: do not run history-export + discover + cron scrape on a brand-new account in the same hour.
5. **Separate roles** (recommended):
   - **Hot session**: cron pain scrape only, 15-20 curated chats.
   - **Cold session**: discover, history-export, channel_search, join queue.
6. **2FA / real SIM**: virtual or recycled numbers get restricted faster; document which number owns which `telethon.session.N`.

Warming is **ops discipline**, not a code flag. Sharding multiplies warming work: **N sessions = N warmups**.

---

## Target architecture (cron-first)

```
                    CHAT POOL (yaml + discovered registry)
                              |
            +-----------------+------------------+
            |                 |                  |
     session A (hot)    session B (hot)    session C (cold)
     cron pain shard0  cron pain shard1   export/discover only
            |                 |                  |
            +--------+--------+                  |
                     |                           |
              NDJSON batch ingest (cron append)
                     |
              Go processor + Mongo
                     |
              pain alert (Python, on emit)
```

Goals:

- No two workers scrape the same chat at the same time.
- ResolveUsername budget spread across accounts.
- Export/discover never steals the hot session lock from cron scrape.

---

## Hypotheses to test

All items below are **experiments**, not shipped claims. Each needs a pass/fail gate before production.

### H1 - Geo / CIS grey hard drop (processor + alerts parity)

**Hypothesis:** Cutting RU/BY/CIS grey market before scoring raises lead quality without killing WW/UA buyers.

**Three layers:**

| Layer | Signals | Drop reason |
|-------|---------|-------------|
| Bookmaker / grey offer | 1win, 1xbet, melbet, mostbet, pin-up, vavada (word-boundary; not bare `1x` or `cpa`) | `cis_grey_market` |
| Geo infra | RF, RU, RB, BY, MIR, Sber, Tinkoff, rub cards, RU drops (extend `geo.Filter` / `geo_heuristic.py`) | `geo_block_ru_by` |
| Metadata | Username/channel_about/sender_bio: .ru handle, tricolor flag emoji, RU channel binding | `geo_block_ru_by` |

**Already in repo:** `geo.Filter` at `processor.go` line ~142; `is_ru_infrastructure_hard_stop` in Python prefilter; UA affinity must **not** drop Kyiv/+380 teams.

**Gap to implement:**

- Shared stop-list (Go + Python mirror).
- Bookmaker regex with buyer-voice exemption (voluum/redtrack/USDT pain in same message).
- **`passes_pain_emit_gate`** must call same gate (alerts bypass Go processor today).

**Test gate:**

```bash
go test ./internal/geo/... ./internal/filter/... ./internal/pipeline/...
python3 -m unittest sources.telegram.test_geo_heuristic sources.telegram.test_prefilter -q
```

**Fail if:** UA team with USDT + voluum postback pain is dropped.

---

### H2 - Chat pool pivot: WW + UA, not RU/CIS arbitrage

**Hypothesis:** 500-600 curated chats (70% EN/WW) yields 60-90 hot leads/month after filters; RU-heavy yaml is recall poison.

**Target segments:**

| Segment | Examples | Pain hooks |
|---------|----------|------------|
| UA arbitrage (WW spend) | Affhub, Conversion, SiGMA UA, WMC | Keitaro + CF Workers/Fastly, high click volume |
| EN/WW buyers | Afflift/STM unofficial TG, Voluum/RedTrack/Binom compare chats | event overage, click loss, self-hosted high volume |

**Remove / deprioritize:** RU affiliate supergroups (`maximaffiliate`, `zuevaff`, etc.), `cpa.rip`-style CIS funnels unless WW stack signal present.

**Actions:**

- Expand [config/discover.icp.json](config/discover.icp.json) seeds (voluum pricing, redtrack overage, binom vs voluum).
- Manual triage yaml -> 40+ WW/UA `buyer_supergroup` rows.
- `channel_geo_reject` on discover ingress.

**Test gate:** 7-day soak: `pain_alerts >= N` (set N after baseline), `accepted` leads with `heat_score` trend up, RU snippet rate down in Mongo sample.

**Funnel math (aspirational, verify):**

| Metric | Before filter | After WW/UA + geo |
|--------|---------------|-------------------|
| Chats in pool | 250 | 500-600 |
| Raw signals/month | ~1000 | ~400-500 |
| RU/1win/CIS ban | 0% targeted | ~40% cut |
| Hot leads/month | ~150 (noisy) | 60-90 (target) |

---

### H3 - PWA / rent app provider chats (highest conversational ICP)

**Hypothesis:** Buyers in PWA/iOS rent client support chats who complain about tracker/postback/webview are tier-1 leads (USDT budgets, real volume).

**Signals:** `postback from app`, `webview stuck`, `sub_id not in keitaro`, `redirect delay`, `s2s from PWA`.

**Where:** PWA.Group, PWA.market, iRent, TD Apps and analogs - mostly **closed client chats** (invite after payment).

**Gap:** No handles in yaml today; discover dorks missing PWA-specific queries.

**Actions:**

- Add SERP dorks: `site:t.me PWA postback tracker`, `webview redirect keitaro`, `sub_id tracker`.
- Manual yaml rows for **public** support channels where invite is obtainable.
- Account must be **inside** ecosystem (paid rent or partner) for closed groups.

**Test gate:** At least 1 pain alert/week from PWA-class source with `HasTrackerPainMessage` true and seller broadcast false.

**Do not claim:** "100% confirmed gray buyer" - may be junior buyer or dev.

---

### H4 - Bulletproof / offshore hosting communities

**Hypothesis:** Public TG around AlexHost, PQ.Hosting, Zomro, Friendhosting surfaces teams with live infra pain (502, abuse suspend, redirect down) during incidents.

**Signals:** hosting incident + tracker stack in same message (`502`, `upstream`, `nginx`, `keitaro`, `binom`).

**Gap:** No status/RSS monitor; only SERP -> discover -> cron scrape.

**Actions:**

- SERP dorks: `site:t.me AlexHost keitaro`, `site:t.me PQ hosting tracker`.
- AND-gate: infra incident language + tracker keyword (not generic "when server up").

**Test gate:** Compare alert rate vs H3; lower priority if signal/noise worse.

---

### H5 - Vendor support chats (agency cabs, PWA, hosting) - buyer voice only

**Hypothesis:** Support chats of agency account sellers and infra vendors contain **buyer** messages ($50k+ spend, redirect/postback pain) worth capturing; seller broadcasts are not.

**Conflict with current filters:**

- `instant_drop`: `agency account`, `accounts for sale`
- `SellerAuthorProfile`: `support`, `agency`, `seller` tokens on channel title

**Required policy (to implement):**

- `channel_class: vendor_support` in yaml - do not drop channel by seller profile.
- Drop seller **broadcast**; accept message with `HasBuyerQuestionPattern` + `HasTrackerPainMessage`.
- Optional boost: `agency spend` + `postback` + `USDT` in one message.

**Test gate:** Fixture messages from buyer in vendor support pass processor; seller promo in same channel still drops.

---

### H6 - Infra OSINT: Shodan / Censys / FOFA Keitaro fingerprint

**Hypothesis:** Reverse IP + HTTP signatures (`kclick_id`, `http.title:Keitaro`, cert SAN clusters) finds high-volume teams; white-page TG contact enables cold outreach.

**Not in repo:** No Shodan/Censys API, no `kclick_id` probe, no domain-per-IP clustering. Only light DNS CNAME in `internal/enrich/dns.go` and `supply` (ads.txt, intel_only).

**Approach (intel branch, not hot processor):**

1. Weekly offline export (Shodan/Censys dorks) -> `data/runtime/infra_clusters.json`.
2. Cluster score: `domains_on_ip >= N` + tracker hint + non-RU ASN.
3. Sample domain -> `tgweb`/lander crawl for `@telegram` on white page.
4. Ingest as `intel_only` or manual outreach queue - **no auto-CRM without pain text**.

**Caveats:** Cloudflare hides bare-IP signatures; 1-2 domains != poor buyer; Whois often privacy.

**Test gate:** Manual review of 50 cluster rows -> >= 10% with reachable TG and plausible buyer infra.

---

### H7 - OTC / crypto desk chats

**Hypothesis:** Closed OTC USDT desks host network owners and fin dirs with tracker infra needs.

**Priority:** Low. Closed access, financial noise, weak parser fit.

**Action:** Manual OSINT only until H1-H5 prove cron path. Do not automate scraping OTC groups.

---

### H8 - Payment / vertical filter table (M11 alignment)

| Param | Drop | Accept |
|-------|------|--------|
| Payments | FOP, TOV, rub cards, RU infra | USDT TRC20/ERC20, crypto invoice (+ infra + pain) |
| Verticals | White goods, courses, infobiz | Gambling WW, crypto, nutra, grey PWA |
| Infra | Solo $5 Keitaro hobby | Clusters, CF Workers, high-load, self-hosted |
| Pain | "how to buy domain", newbie | click loss, postback drop, mysql deadlock, 502 redirect |

**Already in repo:** `validate/crypto_icp.go` (M11), `HasTrackerPainMessage`, `instant_drop` for sellers.

**Gap:** Strict USDT-only without tracker pain should **not** auto-accept; grivna/FOP not fully in geo (UA USDT teams often write in Russian - do not drop on Cyrillic alone).

---

### H9 - Cron pain path vs realtime (control experiment)

**Hypothesis:** Cron scrape + M3 pain gate matches or beats realtime for alert quality at lower FloodWait/ops cost.

**Setup:**

- Disable `parser-telegram-realtime` on VPS.
- Cron: scrape curated yaml chats every `PARSER_POLL_SEC` (or dedicated `scripts/ops/` cron).
- Same M3 AND-gate in `alert_dispatcher`.

**Test gate:** 48h soak - compare `pain_alerts`, FloodWait count, leads accepted vs prior realtime baseline in logs.

**Expected:** Fewer restarts, no IPC complexity; acceptable latency (minutes, not seconds).

---

### H10 - Pain taxonomy -> BidShard tier (product-led discovery)

**Hypothesis:** Parser should classify **pain bucket** and **tier hint** (Starter/Pro/Scale/Enterprise), not only generic "tracker pain". Sales outreach maps 1:1 to BidShard features.

**Four buckets (extend vocab in `keywords.json` + Python pain):**

| pain_bucket | BidShard tier hint | Chat triggers (add to lexicon) | Pitch angle (for card, not auto-DM) |
|-------------|-------------------|--------------------------------|-------------------------------------|
| `cloak_stack_cost` | Starter ($129) / Pro ($399) | hideclick overprice, adspect, cloak it, click limit on cloak, "what cloak for keitaro" | Built-in filter endpoint on VPS; decoy 202; one USDT invoice vs tracker + external cloak APIs |
| `shave_discrepancy` | Pro ($399) | network shaves leads, tracker vs network deposit count, prove shave, postback log blind spot | Postback Trail: every hop + raw gateway response; export bundle with click hashes |
| `infra_scale` | Scale ($749) / Network ($1,399) | keitaro 32gb ram, 502 on pour, mysql cpu 100% on report, slow click-to-land, redirect 1.5s kills fb cr | Go ingest + ClickHouse analytics; same $40 VPS, higher spike tolerance, ms redirects |
| `abuse_ddos` | Enterprise ($2,999) | competitors bot link, hoster blocked channel abuse, ddos on tracking domain | eBPF/XDP edge drop before TCP stack (qualify volume before Enterprise pitch) |

**Already in repo:** `hideclick`, `shaving`, `scrubbing`, `502`, `mysql`, `clickhouse`, `ebpf` in `data/keywords.json`; M11 in `crypto_icp.go`. **Missing:** `adspect`, `cloak it`, RU volume (`200к кликов`), standalone shave without tracker keyword, DDoS/click-fraud competitor patterns.

**Classifier output (target shape):**

```json
{
  "pain_bucket": "infra_scale",
  "tier_hint": "scale",
  "tier_confidence": "med",
  "pitch_key": "clickhouse_ingest"
}
```

**Rules:**

- `tier_hint` is **suggestion**, not auto-quote. Low spend + generic "recommend tracker" -> cap at Starter/Pro.
- `abuse_ddos` -> Enterprise only with volume/hosting context (not one angry message).
- Shave messages mentioning `1win` / `mostbet` conflict with **H1** geo drop - decide: drop geo vs keep shave intel (tag `cis_grey`).

**Test gate:** 20 hand-labeled TG messages -> bucket + tier match human label >= 70%. False Enterprise tier on newbie questions = fail.

---

### H11 - M3 alert gate: expand second leg (do not replace Go scoring)

**Hypothesis:** Missed alerts come from **narrow M3 second leg**, not from lack of weighted scoring in Go.

**Two pipelines (do not conflate):**

| Path | Gate | Output |
|------|------|--------|
| Python cron -> `passes_pain_emit_gate` | M3: tracker leg AND (operational OR commercial intent) | Pain alert TG channel |
| Go ingest -> `processor` | `keywords.json` + `ApplySpendGate` + `CompetitorPainBoost` + displacement | Mongo + CRM webhook |

Go path is **already weighted** (`internal/scoring/engine.go`). M3 is **alert-only**.

**M3 already passes (commercial intent leg):** "посоветуйте трекер", "alternative to keitaro", "какой трекер взять" (`sources/telegram/pain.py`).

**Real M3 gaps to fix:**

| Message | Why missed today |
|---------|------------------|
| "PP shaves leads, how to prove" | No tracker/postback keyword in `TRACKER_PAIN_HINTS` |
| "Adspect too expensive for FB" | Cloak vendor without keitaro leg |
| "200k clicks/day, admin 2 min" | No operational hint without "slow"/502 |
| "в трекере 100, в партнерке 75" | Cyrillic `трекер` not in hints (Latin `tracker` only) |

**Proposed M3 extension (not full 60-point scorer):**

- Add second-leg signals: `shave`, `scrub`, `discrepancy`, cloak vendors (`adspect`, `hideclick` price), volume RU/EN (`\d+\s*k\s*click`, `200к кликов`).
- Add Cyrillic tracker tokens: `трекер`, `трекере`, `постбек`.
- Optional: if `pain_bucket` from H10 is set with `tier_confidence >= med`, pass M3 even without operational verb.

**Do not:** Rip out M3 AND entirely for alerts without measuring false-positive rate on seller spam.

**Test gate:**

```bash
python3 -m unittest sources.telegram.test_alert_dispatcher sources.telegram.test_pain -q
```

Fixture set: 10 should-alert + 10 should-drop (seller broadcast, job noise).

---

### H12 - Unified sales card (alert + CRM bot)

**Hypothesis:** Conversion to 10-day trial rises when alert/CRM card is a **closer script**, not JSON-ish metadata.

**Current:** `internal/crm/telegrambot/lead_card.go` renders Score, Source, Geo, Keywords, snippet. No tier, bucket, pitch, or deep link.

**Target card shape:**

```text
[HOT] POTENTIAL CLIENT [Scale / Pro]
Contact: @traffic_lead_pavel
Chat: @arbitraj_ua_chat
Pain: click fraud / MySQL spike / cloak overprice
Message: "keitaro on 200k clicks/day hangs server, admin 2 min load"
BidShard angle: ClickHouse ingest + silent reject (~$600/mo saved vs tracker+cloak stack)
Open: t.me/traffic_lead_pavel
hash: <id>
```

**Implementation sketch:**

1. `ClassifyBidShardPain(text) -> {bucket, tier_hint, confidence, pitch_line}` shared Go + Python.
2. `FormatLeadNotifyHTML` + `dispatch_pain_alert` use same formatter.
3. **Two tone layers:** card = factual + suggested angle; DM template from [docs/OUTREACH.md](docs/OUTREACH.md) stays professional (no auto-aggressive copy).
4. Savings line (`~$600/mo`) only when text mentions competitor prices or volume - else omit (no invented math).

**Test gate:** Sales review of 10 generated cards -> "would DM" rate >= 50% vs current format blind test.

---

### H13 - Source mix: TG-first, not TG-only

**Hypothesis:** Dropping forum/SERP entirely loses WW technical posts (nginx logs, long threads); TG-only is ops-simpler but lower recall on Scale/Enterprise ICP.

**Keep (hot/warm):**

| Source | Role |
|--------|------|
| Telegram cron pain | 70% effort; primary buyer voice |
| Forum (afflift-style pain threads) | Long technical posts -> `infra_scale` bucket |
| `buyer-discover` SERP | Seeds for yaml triage only |
| `history-export` | Manual outreach queue (M5) |

**Cron-only / deprioritize:**

| Source | Role |
|--------|------|
| Broad Reddit | 429 + newbie noise ([POST_MORTEM](docs/POST_MORTEM.md)) |
| `webpain` open web | Low precision unless narrowed |
| `supply` ads.txt 24/7 | Intel only |
| Realtime | Off (H9) |

**Test gate:** 30-day compare: leads with `pain_bucket=infra_scale` from forum vs TG counts. If forum < 5% of quality leads, consider dropping.

---

### H14 - Go scoring weights aligned with H10 buckets

**Hypothesis:** Keyword boosts should reinforce tier hints without duplicating M3 logic on every ingest.

**Existing weights (do not duplicate blindly):**

- `ApplySpendGate`: +15 spend, cap without spend/competitor
- `CompetitorPainBoost`: voluum/keitaro/binom mention
- `DetectDisplacementTier`: +12 warm / +22 hot migration language
- `CryptoGrayScoreBoost`: +18 (M11)

**Proposed additive boosts (test in `keywords.json` or bucket classifier):**

| Signal | Boost | Notes |
|--------|-------|-------|
| Competitor stack mention | +20 | Already partial via keywords |
| Spend / volume (`$50k`, `200k clicks`) | +30 | Extend RU volume patterns |
| Alternative / migration question | +40 | Overlaps `has_commercial_pain_intent` |
| Infra operational pain (502, mysql lock) | +40 | Overlaps operational hints |

**Threshold experiment:** alert/accept priority when `score >= 60` **and** `pain_bucket` set - compare to current High/Medium from registry.

**Test gate:** Replay last 100 accepted leads with new boosts - no >20% rank inversion on manually ranked top 10.

---

## Implementation phases (sharding)

### P0 - chat_id cache (single session, do first)

**Effort:** 1-2 days. **Unblocks:** fewer FloodWait on cron restart without multi-account.

1. Extend `resolve_chat_entity()` (`sources/telegram/scraper.py`):
   - If yaml `chat_id` set, use it.
   - Else if `store.get_chat_id(channel_key)` from `telegram_channels`, use it.
   - Else resolve by `@username` / invite; on success `store.set_chat_id(...)`.
2. After first successful `get_entity`, persist `chat_id` (already partially wired in `join_policy.py`).
3. Ops: one-time backfill script reading `crawler.db` -> optional `chat_id:` in yaml for top chats.
4. Test: mock store with `chat_id`; assert no `ResolveUsername` call.

**Success:** cron restart resolves 0-5 usernames (new chats only), not 40.

### P1 - static shard (2-3 sessions)

**Effort:** 2-4 days. **No lease logic.** Applies to **cron scrape**, not realtime.

Assign each chat a shard id:

```yaml
# config/sources.telegram.yaml (future field)
- name: media_buyer
  username: media_buyer
  role: buyer_supergroup
  shard: 0   # 0 .. N-1
  # channel_class: vendor_support  # H5 optional
```

| Component | Change |
|-----------|--------|
| Env | `TELEGRAM_SESSION=data/runtime/telethon.session.0`, `TELEGRAM_SHARD=0`, `TELEGRAM_SHARD_COUNT=2` |
| Cron | Two scrape workers, disjoint shard yaml |
| Lock | One lock **per session file** (already `session_path + ".lock"`) |

Shard function:

```text
shard = stable_hash(channel_key) % TELEGRAM_SHARD_COUNT
```

### P2 - dynamic pool + lease (optional, 10+ workers or churning registry)

**Effort:** 1-2 weeks. Shared SQLite `crawler.db` or Mongo `telegram_channel_leases`.

Worker loop: CLAIM -> scrape claimed chats (P0 cache) -> HEARTBEAT -> release on FloodWait.

### P3 - ingest dedup for multi-worker

**Effort:** 2-3 days.

```text
hash_id = SHA256("tgmsg:" + chat_peer_id + ":" + message_id)
```

Pain alerts cannot be "un-sent" by dedup cron.

---

## Role split (recommended production, no realtime)

| Session | Jobs | Chat count |
|---------|------|------------|
| `session.0` | cron pain scrape shard 0 | 15-20 buyer supergroups |
| `session.1` | cron pain scrape shard 1 | 15-20 buyer supergroups |
| `session.2` | history-export, discover, `channel_search`, join queue | 0 scrape (batch only) |

Env on main parser: `PARSER_BG_TELEGRAM=false` when Telethon sidecar holds MTProto.

Never run `history-export-inplace.sh` against a session that cron scrape is using.

---

## Three ingest roads (hypothesis map)

```text
HOT (buyer voice)     -> cron TG + forum pain -> processor -> CRM card (H12)
WARM (discover)       -> SERP + buyer-discover -> yaml triage -> cron
INTEL (no auto CRM)   -> Shodan batch (H6) + supply/tgweb white pages -> manual outreach
CLASSIFY              -> H10 pain_bucket + tier_hint on alert and CRM (H11/H14)
```

---

## Files to touch (implementation checklist)

| Area | Files |
|------|--------|
| H1 geo/bookmaker | `internal/geo/filter.go`, `internal/filter/` (new cis_grey), `sources/telegram/geo_heuristic.py`, `sources/telegram/prefilter.py`, `sources/telegram/alert_dispatcher.py` |
| H2 discover | `config/discover.icp.json`, `config/sources.telegram.yaml`, `sources/telegram/discover.py` |
| H3 PWA pain | `internal/filter/pwa_pain.go`, `sources/telegram/pwa_pain.py`, `pwa_dorks` in discover.icp.json |
| H4 hosting incident | `internal/filter/hosting_incident.go`, `sources/telegram/hosting_incident.py`, `hosting_dorks` |
| H5 vendor_support | `internal/filter/author_profile.go`, `channel_role` in NDJSON/cursor, `vendor_support.py` |
| H6 infra OSINT | `scripts/ops/infra-shodan-export.sh`, `internal/sources/infraosint/`, `parser discover infra-clusters` |
| H7 OTC manual | `docs/H7_OTC_OSINT.md`, `scripts/ops/otc-osint-manual.sh` (no auto scrape) |
| H10 pain taxonomy | new `internal/classify/bidshard_pain.go`, `sources/telegram/pain_taxonomy.py`, `data/keywords.json` |
| H11 M3 second leg | `sources/telegram/pain.py`, `sources/telegram/prefilter.py` |
| H12 sales card | `internal/crm/telegrambot/lead_card.go`, `sources/telegram/alert_dispatcher.py`, `docs/OUTREACH.md` |
| H14 scoring boosts | `data/keywords.json`, `internal/scoring/engine.go` |
| P0 resolve cache | `sources/telegram/scraper.py`, `sources/telegram/cursor.py` |
| P1 shard | `sources/telegram/scraper.py`, `sources/telegram/config.py` |
| P2 lease pool | `sources/telegram/cursor.py`, `sources/telegram/lease.py`, `scripts/ops/telegram-scrape-lease.sh` |
| P3 dedup | `internal/sink/leadid.go`, `internal/pipeline/processor.go` |
| Cron ops | `scripts/ops/` (pain cron wrapper), disable realtime in compose on VPS |
| Tests | `sources/telegram/test_geo_heuristic.py`, `internal/filter/*_test.go` |

---

## Verification

```bash
# P0: no ResolveUsername when chat_id in store
python3 -m unittest sources.telegram.test_scraper -q

# P2: lease claim disjoint + stale reclaim
python3 -m unittest sources.telegram.test_lease -q

# H1: geo + bookmaker gates
go test ./internal/geo/... ./internal/filter/... ./internal/pipeline/...
python3 -m unittest sources.telegram.test_geo_heuristic sources.telegram.test_prefilter -q

# H9: cron soak (no realtime container)
bash scripts/ops/vps-p0-telegram-soak.sh   # adapt for cron-only

# H10-H11: pain taxonomy + M3 fixtures
python3 -m unittest sources.telegram.test_alert_dispatcher sources.telegram.test_pain -q
go test ./internal/scoring/... ./internal/crm/telegrambot/...

# H12: lead card golden tests (when implemented)
go test ./internal/crm/telegrambot/... -run LeadCard

# Dedup P3
go test ./internal/pipeline/... ./internal/sink/...
```

M2 soak: 48h cron-only; track `pain_alerts`, `parser_stats_accepted_total`, FloodWait lines in scrape logs.

---

## Anti-patterns

| Do not | Why |
|--------|-----|
| Invest in multi-realtime compose before H9 passes | Realtime deprecated |
| 3+ accounts in same private supergroup | Duplicate alerts; farm signal |
| Restart all shards at once | ResolveUsername storm per account |
| Shodan -> auto CRM without pain text | Intel != lead |
| Drop Cyrillic-only messages | Kills UA/WW teams |
| Scrape OTC / closed desks without invite | Access + legal noise |
| Skip P0 and only add shards | Each account still resolves 40 handles on boot |
| `instant_drop` on buyer in vendor support | H5 needs channel_class policy |
| Auto Enterprise tier on one DDoS message | H10 needs volume qualification |
| Invented savings on CRM card | H12 requires price/volume in source text |
| Replace M3 with 60-pt scorer in alerts only | H11 - extend second leg; Go already weighted |
| Kill forum before H13 measurement | Lose Scale/infra_scale recall |

---

## Decision guide

| Situation | Action |
|-----------|--------|
| FloodWait after restarts | **P0** chat_id cache |
| Need export + cron scrape 24/7 | **P0 + P1** (2 sessions: hot + cold) |
| 80+ chats or dynamic discover churn | **P0 + P2** lease pool |
| Low lead quality, RU noise | **H1 + H2** before more chats |
| PWA/hosting buyer voice | **H3 + H5** yaml + vendor_support |
| Outbound domain list | **H6** intel branch |
| Realtime vs cron | **H9** - cron wins unless proven otherwise |
| Alert misses "recommend tracker" / shave | **H11** M3 second leg |
| CRM card not actionable | **H12** unified formatter |
| Wrong product tier in outreach | **H10** bucket classifier |
| Forum vs TG ROI | **H13** 30-day compare |

---

## Open questions

1. Shared `crawler.db` on NFS vs one SQLite per host (lease needs single writer).
2. Cold session: append NDJSON vs shared ingest path.
3. Join policy: implemented (`TELEGRAM_SESSION_ROLE=cold` required for `TELEGRAM_INVITE_JOIN`).
4. H6: Shodan API key budget and review cadence (weekly manual triage).
5. Minimum `pain_alerts/week` to call H2 funnel "green".
6. H1 vs H10: drop CIS shave messages (`1win`) or tag and keep for manual review?
7. H12: card language EN only vs mirror post locale (RU/EN)?

---

## Status

| Item | Status |
|------|--------|
| **Realtime production** | **Off / not pursuing** |
| H1 geo + CIS bookmaker gate | Implemented (`geo.FilterH1`, `h1_geo_block.py`) |
| H2 WW/UA chat pivot | Implemented (auto pool from registry, tg catalog meta harvest, h2_pool gate) |
| H3 PWA provider chats | Implemented (`pwa_pain.go/py`, `pwa_dorks` in discover.icp.json) |
| H4 bulletproof hosting TG | Implemented (`hosting_incident.go/py`, `hosting_dorks`) |
| H5 vendor_support policy | Implemented (`channel_role`, `SellerAuthorProfileForChannel`, vendor emit gate) |
| H6 Shodan/Censys intel | Implemented (`infra-shodan-export.sh`, `infraosint` source, `discover infra-clusters`) |
| H7 OTC desks | Manual only (`docs/H7_OTC_OSINT.md`, `otc-osint-manual.sh`; no auto scrape) |
| H8 M11 payment table | Implemented (`h8_payment.go/py`, FOP/TOV/grivna + crypto-only gate) |
| H9 cron vs realtime | **Hypothesis** - `telegram-pain-cron.sh` + `vps-h9-cron-soak.sh` on VPS |
| H10 pain taxonomy -> tier | Implemented (`internal/classify/bidshard_pain.go`, `pain_taxonomy.py`) |
| H11 M3 second leg expansion | Implemented (`pain.py`, `prefilter.py` Cyrillic/shave/volume) |
| H12 unified sales card | Implemented (`lead_card.go`, `alert_dispatcher.py` H10 fields) |
| H13 TG-first source mix | Implemented (`source_mix.go`, `tg-first-collect.sh`, default source profile) |
| H14 scoring bucket boosts | Implemented (`BidShardPainBoost` in `internal/scoring/bidshard_pain.go`) |
| P0 chat_id cache | Implemented (`cursor.get_chat_id`, `resolve_chat_entity`, persist on scrape) |
| P1 static shard | Implemented (`shard.py`, `TELEGRAM_SHARD*`, `telegram-scrape-shard.sh`) |
| P2 lease pool | Implemented (`cursor.claim_due_chats`, `lease.py`, `telegram-scrape-lease.sh`) |
| P3 message dedup | Implemented (`TelegramMessageHashID`, processor telegram path) |
| Cold-session join policy | Implemented (`join_policy.invite_join_allowed`, `TELEGRAM_SESSION_ROLE`) |

Last updated: 2026-09-12 (H8/H13 wiring, cold join policy, cron ops scripts).
