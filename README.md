# BidShard Lead Miner

A Go lead-generation parser for AdTech and affiliate marketing. It crawls public gray-market signals (forums, supply files, landers, Reddit, Discord, CT, GitHub, reviews), scores intent against `data/keywords.json`, and stores qualified leads for the [BidShard](../bidshard) self-hosted tracker.

> **Policy:** Worldwide affiliate/iGaming markets **excluding Russia and Belarus**. LinkedIn is out of scope.

---

## What works today

Verified locally (no external API keys required):

```bash
go build ./...
go test ./... -count=1
go run ./cmd/parser scan
# or: make build && ./bin/parser scan
```

- **Default source is `stub`** (`PARSER_SOURCE=stub`) — built-in fixtures for pipeline/scoring smoke tests.
- **Unit tests** cover sources via `httptest` fixtures; live crawls are not run in CI.
- **MongoDB** is optional for smoke tests (in-memory stub sink). For production, set `MONGO_URI` or `PARSER_EXPORT_JSON`.

---

## Key features

- **Sources:** forum (STM/BHW/AffiliateFix seeds), supply (`ads.txt` / `sellers.json`), lander (HTML + optional Playwright), Reddit (PullPush), Discord (Bot API), Warrior Forum, Certificate Transparency (`ct`), GitHub issue search, review-site seeds.
- **Telegram:** Python Telethon sidecar (`parser telegram`); not part of `-source=all`.
- **Pipeline:** geo/lang hard-reject → keyword scoring → dedup → optional RDAP/DNS/email enrichment → optional Gemini ICP/geo → MX check → Mongo/JSONL sink.
- **Gemini (optional):** ICP classification, geo verify, cold-path junk analysis, keyword suggestions — requires `GEMINI_API_KEY` + Mongo for cold path.
- **Scoring:** competitor stack detection, source reputation, pilot tagging (`pilot-qualified`).

---

## Prerequisites

| Component | Required for | Notes |
| --- | --- | --- |
| **Go 1.25+** | always | see `go.mod` |
| **MongoDB** | production storage | not bundled in `docker-compose.yaml` |
| **Python 3.11+** | Telegram sidecar only | `pip install -r requirements.txt` |
| **Playwright** | headless landers only | `PARSER_LANDER_HEADLESS=true` |

---

## Installation

```bash
git clone https://github.com/your-repo/bidshard-lead-miner.git
cd bidshard-lead-miner

go mod download

# Telegram sidecar only
pip install -r requirements.txt
# or: make setup   # creates .venv and installs requirements.txt
```

### Configuration

```bash
cp .env.example .env
# edit .env — see sections below
```

---

## Sources and flags

`PARSER_SOURCE` (or `-source`) selects crawlers:

| Value | Crawler |
| --- | --- |
| `stub` | Built-in fixtures (default) |
| `forum` | Forum thread seeds → `data/seeds/forum_threads.csv` |
| `supply` | Domain seeds → `ads.txt` / `sellers.json` |
| `lander` | Lander URL seeds |
| `reddit` | PullPush search (rate-limited; may return 429) |
| `discord` | Bot API (needs `DISCORD_BOT_TOKEN` + `DISCORD_CHANNEL_IDS`) |
| `warrior` | Warrior Forum thread seeds |
| `ct` | crt.sh subdomain search |
| `github` | GitHub issue search (needs `GITHUB_TOKEN`) |
| `reviews` | Review-site seeds |
| `all` | `forum`, `supply`, `lander`, `reddit`, `discord`, `warrior` |

Comma-separated lists work: `-source=forum,supply`.

**Telegram** is separate:

```bash
parser telegram --dry-run
parser scan --source=stub   # default smoke test
```

### Seed files

CSV seeds under `data/seeds/` ship with **example URLs** (many are placeholders). Replace them with real thread/domain/lander URLs before expecting live leads. Lines starting with `#` are ignored.

---

## Storage

| Mode | Env | Behavior |
| --- | --- | --- |
| Stub (dev) | `MONGO_URI` unset, no export | Leads processed but not persisted |
| MongoDB | `MONGO_URI=mongodb://localhost:27017` | Primary production sink |
| JSONL export | `PARSER_EXPORT_JSON=data/export/leads.jsonl` | Append-only file sink (can combine with Mongo) |

Mongo connect uses a **5s timeout**; if Mongo is down, the parser logs a warning and falls back to stub/export sinks.

Start Mongo separately (not included in compose):

```bash
docker run -d --name parser-mongo -p 27017:27017 mongo:7
```

---

## Credentials

### Telegram (MTProto)

1. [my.telegram.org](https://my.telegram.org) → API Development tools → create app.
2. Set `TELEGRAM_API_ID`, `TELEGRAM_API_HASH` in `.env`.
3. Create session file:

```bash
python3 -c 'from telethon import TelegramClient; client = TelegramClient("data/telethon.session", int(input("API_ID: ")), input("API_HASH: ")); client.start(); print("Session saved to data/telethon.session")'
```

4. Edit channel list: `config/sources.telegram.yaml`.

### Google Gemini

1. [Google AI Studio](https://aistudio.google.com/) → API key → `GEMINI_API_KEY`.
2. `PARSER_ICP_CLASSIFY` / `PARSER_GEO_CLASSIFY` default to `true` but are no-ops without a key.
3. Cold path (junk analyze, keyword diff) also needs Mongo.

### Discord

1. [Discord Developer Portal](https://discord.com/developers/applications) → Bot token → `DISCORD_BOT_TOKEN`.
2. Enable **Message Content Intent**.
3. Set `DISCORD_CHANNEL_IDS` (comma-separated).

### GitHub

1. Personal access token (classic) with `public_repo` → `GITHUB_TOKEN`.
2. Optional: `GITHUB_SEARCH_QUERIES=voluum alternative;self-hosted tracker`.

---

## Usage

Show commands:

```bash
parser help
```

Smoke test (no network):

```bash
parser scan
# or: go run ./cmd/parser scan
```

Single scan with live sources (needs seeds + network + tokens as applicable):

```bash
parser scan --source=all
```

Polling loop:

```bash
parser run --source=forum,supply
```

Validate configuration:

```bash
parser config check
```

List sources:

```bash
parser sources list
```

### Docker

`docker-compose.yaml` runs **only the parser** (`network_mode: host`). Mongo must run on the host (or update `MONGO_URI`).

```bash
make docker-build
make docker-up
# one-shot:
make docker-run-once
# with Telegram dry-run:
docker compose run --rm parser telegram --dry-run
```

### Headless landers (optional)

Playwright is not a Go dependency. Install separately, then:

```bash
pip install playwright
playwright install chromium
PARSER_LANDER_HEADLESS=true parser scan --source=lander
```

---

## Core concepts

### Geo and language

- Hard-reject: `GEO_BLOCK_COUNTRIES` (default `RU,BY`), phone codes, `.ru`/`.by` domains.
- Locale overlays: `KEYWORDS_LOCALE=es` or `KEYWORDS_LOCALE_PATH=data/keywords-es.json` (also `pt`, `pl`, `de`, `fr`).

### Lead quality

Scoring uses `data/keywords.json` (+ gray overlay):

- **High (≥35):** competitor pain, stack needs (e.g. "Voluum too expensive").
- **Pilot:** `pilot-qualified` tag when pain + contact signals match (`PARSER_PILOT_TAG=true`).

### CRM handoff

Lead status fields (`new`, `contacted`, `replied`, …) exist in the model but writes are gated by `PARSER_LEAD_STATUS_ENABLED=false` (default). Enable when wiring to `bidshard-leads` CRM.

---

## Architecture

| Path | Role |
| --- | --- |
| `cmd/parser` | CLI entry, scan loop, Telegram sidecar pipe |
| `internal/sources` | Source adapters |
| `internal/pipeline` | Worker pool, geo → score → dedup → sink |
| `internal/sink` | MongoDB, JSONL, bulk writer |
| `internal/gemini` | AI client, quotas, ICP/geo |
| `internal/coldpath` | Junk queue, analyze/report loops |
| `sources/telegram` | Telethon scraper (Python) |

---

## Development

```bash
make test
make test-py          # Telegram Python unit tests
bash scripts/ci/check_parser_slop.sh
```

Backlog and verification gates: [MILESTONE.md](MILESTONE.md).
