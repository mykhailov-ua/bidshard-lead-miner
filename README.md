# BidShard Lead Miner v1.2 — Runbook & Operational Guide

High-performance Go lead-generation miner for AdTech, affiliate marketing, and iGaming gray-market intent signals. It crawls public sources (forums, Reddit, GitHub, Trustpilot/G2 reviews, supply files, SERP dorks, Telegram sidecar), scores intent against `data/keywords.json`, and stores qualified leads for the [BidShard](../bidshard) self-hosted tracker.

> **Targeting Policy:** Worldwide affiliate/iGaming markets **excluding Russia (RU) and Belarus (BY)**. LinkedIn is out of scope.

---

## 1. Quick Start & Docker Background Runbook

The primary production mode runs the miner continuously in the background inside a Docker container.

### Step 1: Environment Configuration

Copy `.env.example` and configure your environment variables:

```bash
cp .env.example .env
```

Key variables for production `.env`:

```env
# Selected crawlers (all, or specific list: forum,reddit,github,serp,reviews,supply,ct,warrior)
PARSER_SOURCE=all

# Polling interval in continuous run mode (seconds)
PARSER_POLL_SEC=120

# Output & Logging
PARSER_OUTPUT=ndjson
PARSER_LOG_FORMAT=json

# HTTP Concurrency & Residential Proxies (for uTLS Cloudflare bypass)
PARSER_HTTP_WORKERS=10
PARSER_PROXY_LIST=http://user:pass@proxy1.com:8080,http://user:pass@proxy2.com:8080

# Storage Sinks
MONGO_URI=mongodb://localhost:27017/parser
PARSER_EXPORT_JSON=/app/data/export/leads.jsonl
```

### Step 2: Build & Launch Docker Container

Run the service in the background (detached mode):

```bash
# Build Docker image
docker compose build

# Start miner container in background (uses command: ["run"] with restart: unless-stopped)
docker compose up -d
```

### Step 3: Operational Monitoring & Management

```bash
# Stream live logs
docker compose logs -f parser

# Check container status
docker compose ps

# Stop background execution
docker compose down

# Execute one-shot scan round in temporary container
docker compose run --rm parser scan --source=reddit,github
```

---

## 2. CLI Runbook (Standalone Local Execution)

For local development or manual cron-triggered runs, use the compiled Go CLI `bin/parser` or `go run ./cmd/parser`.

### Build Binary

```bash
go build -o bin/parser ./cmd/parser
```

### Common Operator Commands

```bash
# 1. Validate configuration, Mongo connectivity, and seed CSV paths
./bin/parser config check

# 2. List available source adapters and their operational status
./bin/parser sources list

# 3. Perform a single scan round with console table output (smoke test)
./bin/parser scan once --source=stub --output=table

# 4. Perform a single scan round across all active live sources
./bin/parser scan --source=all

# 5. Run continuous polling loop (background loop without Docker)
./bin/parser run --source=forum,reddit,github
```

---

## 3. Network Architecture & uTLS Cloudflare Bypass

The network layer (`internal/httpclient`) is engineered to bypass Cloudflare Bot Management and WAF protections:

1. **uTLS TLS Fingerprinting (`utls.HelloChrome_Auto`)**:
   * Replaces standard Go `crypto/tls` ClientHello handshake with Chrome 122+ TLS fingerprints and HTTP/2 ALPN (`h2`).
2. **Browser Header Mirroring**:
   * Injects `User-Agent`, `Accept`, `Accept-Language`, `Sec-Ch-Ua`, `Sec-Ch-Ua-Mobile`, `Sec-Ch-Ua-Platform`, `Sec-Fetch-Dest`, `Sec-Fetch-Mode`, `Sec-Fetch-Site`, `Sec-Fetch-User` into every outgoing HTTP request.
3. **Automated Cloudflare Cooldown & Proxy Rotation**:
   * If a proxy node encounters a Cloudflare `403 Forbidden` or `503 Service Unavailable` block (`CF-Ray` or `Server: cloudflare`), it enters a **10-minute cooldown**.
   * Incoming requests dynamically route to alternative proxies in the pool via lock-free atomic round-robin.

---

## 4. Source Execution Matrix & Profile Configuration

| Source | Resistance | Profile | Strategy & Limits |
| :--- | :--- | :--- | :--- |
| `reddit` | Low | High-Priority | PullPush & Arctic Shift search across `r/affiliatemarketing`, `r/media_buying`, `r/adops`, `r/PPC`. |
| `github` | Low | High-Priority | GitHub REST Search API for Issues & Discussions (`"tracker alternative"`, `"openrtb"`, `"clickhouse tracker"`, `"voluum api"`, `"keitaro api"`). |
| `telegram` | Low | High-Priority | Python Telethon sidecar (`parser telegram`) for CPA & arbitrage chats. |
| `forum` | High (WAF) | Heavy Scrape | XenForo/vBulletin spidering with uTLS transport, residential proxies, and **0.5 RPS per host** rate limit. |
| `reviews` | High (WAF) | Heavy Scrape | Trustpilot/G2 1–2★ review harvesting for competitor refugees (Voluum, Keitaro, RedTrack, Binom, FunnelFlux). |
| `supply` | Low | Intel | Scans `/ads.txt`, `/app-ads.txt`, `/sellers.json` extracting `CONTACT=` directives & publisher emails. |
| `serp` | Medium | Discovery | Google/Bing/DDG dork execution for tracker pain queries. |

---

## 5. Scoring Dictionary & Intent Qualification

Lead intent is evaluated using a 3-tier dictionary system (`data/keywords.json`):

* **Tier 1: Critical Intent (+50 score / Tag: `hot-lead`)**
  * `voluum alternative`, `keitaro alternative`, `self-hosted tracker alternative`, `tracker migration`, `switch from voluum`, `voluum pricing too high`, `looking for high volume tracker`.
* **Tier 2: Technical Pain & Financial Loss (+30 score / Tag: `pain-point`)**
  * `postback failing`, `click loss`, `tracking lost events`, `redirect delay`, `slow click redirection`, `overage charges`, `event limits reached`, `tracker down`, `cloaking detection`.
* **Tier 3: Infrastructure High-Rollers (+20 score / Tag: `high-roller`)**
  * `dedicated server tracker`, `clickhouse for ad tracking`, `10m events per day`, `50k rps`, `usdt payment tracker`, `media buying team setup`, `custom openrtb endpoint`.

### Pilot Qualification Tagging

Leads scoring **$\ge 50$** or satisfying 3+ independent pilot signals (budget, competitor stack, pain, VPS, USDT, buyer role, volume, migration intent) automatically receive the **`pilot-qualified`** tag.

---

## 6. Output Data Format (JSONL & MongoDB)

Leads are stored in MongoDB (`leads` collection) and append-only NDJSON file exports (`PARSER_EXPORT_JSON`):

```json
{
  "hash_id": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
  "source": "reddit:affiliatemarketing",
  "author": "media_buyer_mx",
  "contact": {
    "type": "telegram",
    "value": "@buyer_mx"
  },
  "intent_score": 80,
  "tags": ["pilot-qualified", "hot-lead", "pain-point"],
  "matched_keywords": ["voluum alternative", "redirect delay"],
  "context_snippet": "Keitaro postback died again on FTD. Looking for voluum alternative with flat fee pricing...",
  "posted_at": "2026-08-17T10:00:00Z",
  "status": "new"
}
```

---

## 7. Troubleshooting & Verification

```bash
# Run unit tests across workspace
go test ./... -count=1

# Run hot-path benchmarks
go test -v ./internal/scoring -bench=BenchmarkKeywordExpander -benchmem

# Check Docker container health
docker inspect --format='{{.State.Health.Status}}' parser
```
