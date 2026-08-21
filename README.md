# BidShard Lead Intent Processor

Lead collection and scoring from public gray-market sources (forums, Reddit, GitHub, supply/ads.txt, landers, Discord, Telegram). Pipeline: crawl -> normalize -> keyword scoring -> geo/ICP gates -> dedup -> MongoDB (+ optional JSONL export).

**Geo policy:** hard-reject RU/BY (`GEO_BLOCK_COUNTRIES`). LinkedIn is not supported.

**Credentials:** [docs/credentials.md](docs/credentials.md) | **VPS deploy:** [docs/deploy.md](docs/deploy.md) | **Operations (tgweb, proxy, eBPF):** [docs/ops.md](docs/ops.md)

---

## Deployment (Docker)

Stack: `mongo` (bridge, port 27017) + `parser` (`network_mode: host`, `parser run`).

```bash
cp .env.example .env
docker compose build
docker compose up -d
```

Validate config:

```bash
docker compose run --rm parser config check
```

Day-to-day:

```bash
docker compose logs -f parser
docker compose ps
docker compose down
docker compose run --rm parser scan --source=forum,reddit
```

### Minimal `.env`

```env
MONGO_URI=mongodb://127.0.0.1:27017
MONGO_DB=parser
PARSER_MONGO_COLLECTION=leads
PARSER_EXPORT_JSON=/app/data/export/leads.jsonl
PARSER_EXPORT_JSON_FORMAT=auto
PARSER_SOURCE=all
PARSER_POLL_SEC=120
PARSER_OUTPUT=auto
PARSER_LOG_FORMAT=auto
```

---

## Local build (no Docker)

Requirements: Go 1.25+, Python 3.12 (Telegram sidecar), MongoDB.

```bash
go build -o bin/parser ./cmd/parser
cp .env.example .env
./bin/parser config check
./bin/parser scan --source=forum,supply,lander --output=pretty
./bin/parser run --source=forum,reddit
```

CLI:

| Command | Description |
|---------|-------------|
| `parser config check` | validate env, seeds, Mongo ping |
| `parser sources list` | registered sources and prerequisites |
| `parser scan` | single scan round |
| `parser run` | polling loop (`PARSER_POLL_SEC`) |
| `parser telegram` | Telethon sidecar + ingest |
| `parser ingest` | NDJSON/stdin ingest |

---

## HTTP client and anti-bot

- **uTLS:** ClientHello mimics Chrome (TLS fingerprint).
- **Headers:** browser-like defaults on outbound HTTP.
- **Proxy rotation + CF cooldown:** on 403/503 via proxy - ~10m cooldown, failover to next URL in `PARSER_PROXY_LIST`. Profiles: `config/env/`, preflight `make preflight-tgweb`, crawl `make tgweb-crawl`. See [docs/ops.md](docs/ops.md#proxy).
- **eBPF dev probe (Linux):** syscall/sched/net probe for tgweb crawl analysis - [docs/ops.md](docs/ops.md#ebpf-dev-probe-linux), `make bpf-dev`, `sudo make bpf-session-start`.

---

## Sources

| ID | Transport | Auth | Notes |
|----|-----------|------|-------|
| `reddit` | PullPush API | - | subreddits + queries |
| `github` | GitHub Search API | `GITHUB_TOKEN` | issue search |
| `telegram` | MTProto (Telethon) | `TELEGRAM_API_*` | sidecar, yaml config |
| `forum` | HTTP | - | seed CSV, host rate limit |
| `supply` | HTTP | - | ads.txt / sellers.json |
| `lander` | HTTP (+ optional headless) | - | Next.js `__NEXT_DATA__` / RSC flight |
| `discord` | Bot API | `DISCORD_BOT_TOKEN` | channel IDs |
| `warrior`, `reviews`, `ct`, `serp` | HTTP | - | opt-in / seeds |

Default `PARSER_SOURCE=all`: forum, supply, lander, reddit, discord, warrior, serp.

---

## Scoring

Keyword registry: `data/keywords.json` (+ overlays `keywords-gray.json`, locale files).

Priority tiers (examples):

| Tier | Score boost | Tags |
|------|-------------|------|
| Hot intent | +50 | `hot-lead` - voluum/keitaro alternative, migration |
| Pain | +30 | `pain-point` - postback failing, tracker down |
| Scale signals | +20 | `high-roller` - dedicated infra, high event volume |

`pilot-qualified`: score >= threshold or spend tier + stack + pain combo (`PARSER_PILOT_TAG`).

Pipeline gates (before accept): geo filter, hard-reject phrases, keyword prescan, contact extraction, MX (optional), Gemini ICP/geo (optional), dedup (seen cache + Mongo `hash_id`).

---

## Output schema (JSONL / Mongo)

```json
{
  "hash_id": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
  "source": "reddit:affiliatemarketing",
  "author": "media_buyer_mx",
  "contact": { "type": "telegram", "value": "@buyer_mx" },
  "intent_score": 80,
  "tags": ["pilot-qualified", "hot-lead"],
  "matched_keywords": ["voluum alternative"],
  "context_snippet": "...",
  "posted_at": "2026-08-17T10:00:00Z",
  "status": "new"
}
```

| Field | Description |
|-------|-------------|
| `hash_id` | dedup key (contact-derived) |
| `source` | crawler source identifier |
| `intent_score` | weighted keyword score |
| `matched_keywords` | matched phrases |
| `status` | CRM handoff state (`new`, ...) when `PARSER_LEAD_STATUS_ENABLED` |

---

## Operations

**Mongo backup/restore:** `make backup`, `make restore DUMP=...` (see `scripts/ops/`).

**Integration tests:** `make test-integration` (requires `MONGO_URI`, tag `integration`).

**Debug scan logs:**

```bash
docker compose run --rm \
  -e PARSER_LOG_LEVEL=debug \
  parser scan 2>&1 | tee data/export/scan-debug.log
```

---

## Troubleshooting

| Symptom | Check |
|---------|-------|
| Mongo connection error | `parser config check`, `docker compose ps`, `MONGO_URI` (host network -> `127.0.0.1`) |
| Zero raw items | seed URLs (`data/seeds/*.csv`), source errors in logs, HTTP/CF blocks |
| `json export open failed` | permissions on `data/export/` for parser UID (10001) |
| Gemini skipped | `GEMINI_API_KEY`, `PARSER_ICP_CLASSIFY` / `PARSER_GEO_CLASSIFY` |
| Forum `malformed HTTP response` | HTTP/2 vs HTTP/1 - CF/ALPN; proxy or uTLS path |

**Tests:** `go test ./...`
