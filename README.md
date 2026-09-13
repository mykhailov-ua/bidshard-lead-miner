# BidShard Lead Intent Processor

Go parser + Python Telethon sidecar for **BidShard** outbound: find media buyers with tracker/pain voice in Telegram supergroups, score and gate leads, write MongoDB / JSONL, notify CRM via webhook and Telegram bot.

**Product focus (2026-09):** MTProto realtime listener + pain alerts are the primary funnel. SERP, forum, Reddit, and tgweb are **discovery or batch** paths, not hot-poll CRM accept. See [docs/ICP.md](docs/ICP.md) and [docs/MILESTONE.md](docs/MILESTONE.md).

**Honest status:** code for M0-M11 is largely in tree; **production yield is still low** (FloodWait on Telethon `ResolveUsername`, forum 403 without residential proxy, PullPush 429 on Reddit archive). Gemini ICP is **off** on VPS (`GEMINI_API_KEY` empty). Soak gates in M2/M9 are not green yet.

| Area | Status |
|------|--------|
| P0 prescan, synthetic contact reject, CRM webhook | Deployed VPS |
| Hot poll `webpain,reviews` only (no SERP/reddit/forum) | Deployed |
| Telegram realtime + M3 pain AND-gate + alert cards | Deployed; 12/40 chats listening (FloodWait) |
| 40 `buyer_supergroup` in yaml | Done; session sharding not implemented |
| History export M5, Reddit offline M7, buyer-discover | CLI + VPS scripts; batch runs fragile |
| Forum fetch M6 | bgworker wired; 403 on datacenter IP |
| Multi-session sharding | Design only: Д[SHARDING_SESSION.md](SHARDING_SESSION.md) |

**Geo policy:** hard-reject RU/BY (`GEO_BLOCK_COUNTRIES`). LinkedIn is not supported.

**Docs:** [docs/CREDENTIALS.md](docs/CREDENTIALS.md) | [docs/DEPLOY.md](docs/DEPLOY.md) | [docs/OPS.md](docs/OPS.md) | [docs/OUTREACH.md](docs/OUTREACH.md) | [HEADLESS.md](HEADLESS.md) (browser signals) | [docs/CRAWL_EGRESS_ANTIFRAUD.md](docs/CRAWL_EGRESS_ANTIFRAUD.md) (proxy + Playwright + CF risk)

---

## Architecture (production target)

```
DISCOVERY (no CRM accept from hot poll)
  SERP / jobboard / buyer-discover --> discovered_* JSON registries
  forum bgworker, tgweb cron (proxy), reddit offline archive

PRIMARY FUNNEL
  Telethon realtime (buyer_supergroup) --> pain alert (M3 AND-gate)
       |                                      |
       +--> IPC msgpack --> Go processor --> Mongo + JSONL
       +--> CRM webhook --> crm-bot --> Telegram lead card

GATES (Gemini off on VPS)
  keyword prescan, telegram buyer voice (M8), MIN_SCORE 70 / telegram 50,
  contact must be telegram:@user | forum:user | email (no @serp:)
```

Accept rule: no lead without a **reachable** contact. Pain alerts require tracker + operational pain (or crypto-gray) and `@username`.

---

## Quick start (local)

Fresh clone is **not** fully runnable without setup.

```bash
cp .env.example .env
docker compose up -d mongo
make build
make venv                    # Telethon sidecar
go run ./cmd/parser config check
```

Optional profiles: `cat config/env/.env.bidshard-icp.example >> .env` (VPS-like ICP), `cat config/env/.env.crm-telegram.example >> .env`.

Telegram MTProto (optional):

```bash
docker compose run --rm -it parser telegram login --qr
docker compose -f docker-compose.telegram-realtime.yaml --profile parser-telegram-realtime up -d
```

Tests: `go test ./...` and `make test-py` (Telethon unit tests; no live MTProto in CI).

---

## Docker stack

Default `docker-compose.yaml`:

| Service | Role |
|---------|------|
| `mongo` | leads, entities, junk |
| `parser` | `parser run` poll loop + bgworker |
| `crm-bot` | webhook inbox, Telegram lead notify |

Optional `docker-compose.telegram-realtime.yaml`:

| Service | Role |
|---------|------|
| `parser-telegram-realtime` | long-running NewMessage listener (`TELEGRAM_REALTIME=1`) |

Parser uses `network_mode: host` on VPS so `127.0.0.1` reaches Mongo and crm-bot. Telethon session: named volume `parser_runtime` (`data/runtime/telethon.session` inside container).

```bash
docker compose build
docker compose up -d
docker compose -f docker-compose.telegram-realtime.yaml --profile parser-telegram-realtime up -d
docker compose logs -f parser
```

VPS deploy: `make vps-deploy-p0`, env merge `bash scripts/ops/vps-apply-p0-env.sh`.

---

## CLI

| Command | Description |
|---------|-------------|
| `parser config check` | validate env, seeds, Mongo ping |
| `parser sources list` | registered sources and prerequisites |
| `parser scan` / `parser run` | single round / poll loop (`PARSER_POLL_SEC`) |
| `parser telegram login` | MTProto session (QR or phone) |
| `parser telegram realtime` | NewMessage listener (sidecar) |
| `parser telegram history-export` | M5 historical pain NDJSON (`--since`, `--relax`) |
| `parser reddit offline-archive` | M7 PullPush/Arctic Shift batch (not hot poll) |
| `parser discover` | SERP / jobboard catalog harvest |
| `parser ingest` | NDJSON or msgpack stdin ingest |
| `crm-bot run` | CRM webhook + optional Telegram notify |

---

## Sources

### Hot poll (`PARSER_SOURCE`)

**Production VPS:** `webpain,reviews` only. SERP, reddit, forum, github are **out** of the 24/7 poll (SEO noise, 429, CF 403).

| ID | Hot poll | Discovery / batch |
|----|----------|-------------------|
| `telegram` | via realtime container (not `PARSER_SOURCE`) | discover, history-export, channel_search |
| `serp` | no | bgworker `serp_telegram_catalog`, `serp_forum_threads` |
| `forum` | no | bgworker `forum_crawl` (needs residential proxy) |
| `reddit` | no (PullPush 429) | `parser reddit offline-archive` |
| `webpain`, `reviews` | yes (low volume) | - |
| `tgweb`, `lander`, `supply`, `jobboard` | opt-in / cron | `make buyer-discover`, CF crawl scripts |

Registry: `config/sources.telegram.yaml` (40 `buyer_supergroup`), `data/runtime/discovered_*.json`.

---

## Scoring and gates

Keyword registry: `data/keywords.json` (+ crypto-gray phrases for M11).

Without Gemini (current VPS):

- **M3 (Python):** pain alerts = tracker AND operational pain (+ crypto-gray path); needs `@username`.
- **M8 (Go):** `telegram_no_buyer_voice` reject; `PARSER_ACCEPT_MIN_SCORE=70`, `PARSER_TELEGRAM_ACCEPT_MIN_SCORE=50`.
- Prescan, synthetic `@serp:` reject, spend/newbie gates, contact extraction.

With Gemini (optional, `GEMINI_API_KEY`): ICP/geo classify on warm path or inline; `PARSER_GEMINI_DEFER` queues async analysis. **Not verified on prod** while quota/key empty.

---

## Ops commands (Makefile)

| Target | Purpose |
|--------|---------|
| `make vps-deploy-p0` | rsync + docker build on VPS |
| `make vps-apply-p0-env` | merge ICP env keys (via script) |
| `make vps-telegram-realtime-soak` | M2 listener / pain / export gate report |
| `make vps-history-export ARGS="--since 2025-03-01 --detach"` | M5 batch on VPS |
| `make vps-reddit-offline-archive ARGS="--detach"` | M7 Reddit batch on VPS |
| `make vps-buyer-discover` | SERP + CF crawl discovery (detached) |
| `make buyer-discover` | local discovery cron job |
| `make acceptance-soak` | JSONL export quality gates |
| `make warm-path-status` | deferred Gemini queue snapshot |

Soak / metrics: `bash scripts/ops/icp-soak-report.sh --vps`, `bash scripts/ops/vps-status.sh`.

---

## Output (JSONL / Mongo)

Leads keyed by `hash_id` (contact-derived; see [SHARDING_SESSION.md](SHARDING_SESSION.md) for telegram message-level dedup plan).

```json
{
  "hash_id": "...",
  "source": "telegram:@mediabuyingandselling",
  "score": 32,
  "priority": "Medium",
  "contacts": [{"type": "telegram", "value": "@buyer1"}],
  "matched": ["keitaro(+12)"],
  "snippet": "keitaro postback failing again",
  "status": "new"
}
```

CRM webhook posts to `crm-bot` when `PARSER_CRM_WEBHOOK=true` and secrets synced (`PARSER_CRM_WEBHOOK_SECRET` = `CRM_WEBHOOK_SECRET`).

---

## HTTP client and anti-bot

- uTLS Chrome fingerprint, browser-like headers.
- Proxy rotation + cooldown (`PARSER_PROXY_LIST` / `PARSER_PROXY_LIST_FILE`).
- Optional eBPF dev probe for tgweb: `make bpf-dev`, [docs/OPS.md](docs/OPS.md).

---

## Troubleshooting

| Symptom | Likely cause |
|---------|----------------|
| `realtime skip chat ... ResolveUsername` / FloodWait | Too many username resolves; wait hours or implement P0 `chat_id` cache ([SHARDING_SESSION.md](SHARDING_SESSION.md)) |
| `listening channels=12` not 40 | FloodWait skipped chats on startup |
| `pain alert sent` = 0 | Strict M3 gate; quiet chats; or not in listened set |
| Hot poll `raw=0` | webpain DDG/proxy failures; expected low volume |
| Forum bgworker 0 raw | CF 403 without residential proxy |
| Reddit archive fails | PullPush 429 from VPS IP |
| `database is locked` | Two processes on same `telethon.session` |
| Gemini / all `pending` | Key empty or defer on; warm path backlog |
| CRM webhook 401 | `PARSER_CRM_WEBHOOK_SECRET` mismatch |

Telethon session lock: one MTProto job at a time per session file. Stop realtime before `make vps-history-export`.

---

## Development

```bash
go test ./...
make test-py
bash scripts/ci/check_parser_slop.sh   # if present
make backup / make restore             # Mongo
```

Commit policy and crawl budgets: `.cursor/rules/core.mdc` (agent); human planning may use local `BACKLOG.md` (gitignored).

---

## Related repositories

Product: [BidShard](https://bidshard.com/) (self-hosted tracker + antifraud stack). This repo is **sources-only** lead discovery for that ICP, not the tracker product itself.
