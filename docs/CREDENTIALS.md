# Credentials and environment

Copy `.env.example` to `.env`. Validate with `docker compose run --rm parser config check`.

---

## Sources without API keys

| Source | Needs | API key |
|--------|-------|---------|
| `forum` | `data/seeds/forum_threads.csv` | - |
| `supply` | `data/seeds/domains.csv` | - |
| `lander` | `data/seeds/lander_urls.csv` | - |
| `warrior` | `data/seeds/warrior_threads.csv` | - |
| `reddit` | [PullPush API](https://api.pullpush.io/reddit/search/submission/) | - |
| `reviews`, `ct`, `serp` | seed files / public HTTP | - |

Local Mongo (docker-compose): `MONGO_URI=mongodb://127.0.0.1:27017`.

---

## `GEMINI_API_KEY`

Powers ICP/geo classification, cold-path junk analysis, embeddings.

| Variable | Notes |
|----------|-------|
| `GEMINI_API_KEY` | From [Google AI Studio](https://aistudio.google.com/api-keys) |
| `PARSER_ICP_CLASSIFY` | Set `true` after adding the key |
| `PARSER_GEO_CLASSIFY` | Set `true` after adding the key |
| `PARSER_GEMINI_SYNC_GEO` | Set `true` when `PARSER_GEMINI_DEFER=true` and CRM webhook is on (geo before Mongo write) |

### Prod geo compliance (RU/BY + CRM)

When `PARSER_CRM_WEBHOOK=true`, configure geo before leads reach sales. One of:

**Option A - defer + after-analysis webhook (recommended, saves inline Gemini RPM):**

```env
GEMINI_API_KEY=<key>
PARSER_GEO_CLASSIFY=true
PARSER_GEMINI_DEFER=true
PARSER_CRM_WEBHOOK_AFTER_ANALYSIS=true
PARSER_LEAD_STATUS_ENABLED=true
GEO_BLOCK_COUNTRIES=RU,BY
```

Merge full profile: `cat config/env/.env.prod.example >> .env`

**Option B - inline sync geo on accept:**

```env
GEMINI_API_KEY=<key>
PARSER_GEO_CLASSIFY=true
PARSER_GEMINI_SYNC_GEO=true
GEO_BLOCK_COUNTRIES=RU,BY
PARSER_ENRICH_RDAP=true
```

`parser config check` errors on prod when CRM webhook lacks a geo gate. Unset env keys get safe defaults via `applyComplianceDefaults` when `PARSER_CRM_WEBHOOK=true`.

CRM inbox: `crm-bot api list --status new` hides `analysis_status=pending` and parser reject statuses. Pass `--all` to include deferred leads.

1. Create an API key in AI Studio.
2. Add `GEMINI_API_KEY=<key>` to `.env`.
3. Restrict the key to Generative Language API in GCP (recommended).

Refs: [get started](https://ai.google.dev/gemini-api/docs/get-started), [rate limits](https://ai.google.dev/gemini-api/docs/rate-limits).

Free tier for `gemini-2.0-flash` is roughly 15 RPM / 1500 RPD - check current limits.

---

## `TELEGRAM_API_ID`, `TELEGRAM_API_HASH`

MTProto sidecar (`parser telegram`), channels from `config/sources.telegram.yaml`.

| Variable | Source |
|----------|--------|
| `TELEGRAM_API_ID` | Integer from [my.telegram.org](https://my.telegram.org) |
| `TELEGRAM_API_HASH` | 32-char hex |
| `TELEGRAM_CONFIG_PATH` | Default: `config/sources.telegram.yaml` |

1. Log in at [my.telegram.org](https://my.telegram.org).
2. Open [API development tools](https://my.telegram.org/apps) and register an app.
3. Map `api_id` -> `TELEGRAM_API_ID`, `api_hash` -> `TELEGRAM_API_HASH`.

Do not copy DC IPs or RSA keys from the MTProto servers block - Telethon picks production endpoints. Use `TELEGRAM_USE_TEST_DC=true` only for sandbox (`149.154.167.40:443`).

After credentials, create a session:

```bash
docker compose run --rm -it parser telegram login --qr
```

Session file: `data/runtime/telethon.session` (path in yaml).

---

## `DISCORD_BOT_TOKEN`, `DISCORD_CHANNEL_IDS`

Source `discord` - read channel messages.

| Variable | Description |
|----------|-------------|
| `DISCORD_BOT_TOKEN` | Bot token |
| `DISCORD_CHANNEL_IDS` | Comma-separated channel IDs |
| `DISCORD_MAX_MESSAGES` | Optional, default 50 |

**Token:** [Developer Portal](https://discord.com/developers/applications) -> Bot -> Reset Token (copy once).

Enable **Message Content Intent** (privileged) to read message bodies.

**Install bot:** OAuth2 URL Generator -> scope `bot` -> View Channels + Read Message History -> authorize on your guild.

**Channel ID:** Settings -> Advanced -> Developer Mode -> right-click channel -> Copy Channel ID -> `DISCORD_CHANNEL_IDS=id1,id2`.

---

## `GITHUB_TOKEN`

Source `github` - GitHub Search API (`GITHUB_SEARCH_QUERIES`).

**Fine-grained PAT (preferred):** [Fine-grained tokens](https://github.com/settings/personal-access-tokens) -> public repos -> Metadata read.

**Classic PAT (fallback):** [Tokens](https://github.com/settings/tokens) -> scope `public_repo`.

---

## `MONGO_URI`

Connection string, not an API key.

```env
MONGO_URI=mongodb://127.0.0.1:27017
MONGO_DB=parser
PARSER_MONGO_COLLECTION=leads
```

Atlas: [register](https://www.mongodb.com/cloud/atlas/register) -> Connect -> Drivers -> paste `mongodb+srv://...` URI.

---

## CRM sidecar (`crm-bot`)

HTTP sidecar + CLI. No web UI, no Telegram Bot API. Parser webhook: `POST /v1/leads` with Bearer auth.

| Variable | Purpose |
|----------|---------|
| `CRM_WEBHOOK_ADDR` | Listen addr (default `127.0.0.1:8080`; bind localhost in prod) |
| `CRM_WEBHOOK_SECRET` | Bearer token for parser only (`PARSER_CRM_WEBHOOK_SECRET`) |
| `CRM_API_URL` | Remote CLI base URL (`https://crm.example.com`) |
| `CRM_API_USER` / `CRM_API_PASSWORD` | Caddy basicauth for `crm-bot api` |
| `PARSER_LEAD_STATUS_ENABLED` | Must be `true` on parser so leads get `status: new` |

Sales access: **Caddy** on 443 with `basicauth`, reverse proxy to `127.0.0.1:8080`. See `config/caddy/Caddyfile.example`.

```bash
make build-crm-bot
./bin/crm-bot config check
./bin/crm-bot run
```

### Remote CLI (laptop -> VPS)

```bash
export CRM_API_URL=https://crm.example.com
export CRM_API_USER=sales
export CRM_API_PASSWORD=...
./bin/crm-bot api stats
./bin/crm-bot api list --status new --limit 20
./bin/crm-bot api set-status --hash <hash_id> --status contacted
./bin/crm-bot api purge --status spam --yes
```

### On-server Mongo CLI (SSH on VPS)

```bash
./bin/crm-bot db stats
./bin/crm-bot db delete --hash <hash_id> --yes
./bin/crm-bot db purge --status new --score-max 30 --yes
./bin/crm-bot db set-status --hash <hash_id> --status spam
```

HTTP admin API (same operations, Caddy basicauth in prod):

| Method | Path |
|--------|------|
| GET | `/v1/admin/stats` |
| GET | `/v1/admin/leads?status=new&limit=50` |
| GET | `/v1/admin/leads/get?hash_id=...` |
| PATCH | `/v1/admin/leads` |
| DELETE | `/v1/admin/leads?hash_id=...` |
| POST | `/v1/admin/leads/purge` |

---

## Reddit (PullPush)

No API key. Tune:

```env
REDDIT_SUBREDDITS=affiliatemarketing,media_buying,juststart
REDDIT_QUERIES=voluum alternative;postback failing;click id not found;tracker numbers don't match;tracker too expensive
REDDIT_MAX_RESULTS=25
```

PullPush may rate-limit without registration: [pullpush.io](https://api.pullpush.io/).

---

## Other variables

| Variable | Purpose |
|----------|---------|
| `PARSER_PROXY_LIST` | Comma-separated HTTP proxies - see [OPS.md](OPS.md#proxy) |
| `PARSER_MX_CHECK` | MX validation for email leads |
| `PARSER_LEAD_STATUS_ENABLED` | Set `true` before CRM inbox - parser writes `status: new` on accept |
| `PARSER_ENRICH_RDAP` / `DNS` / `EMAIL` | RDAP/DNS enrichment |
| `SUPPLY_BASE_URL` | HTTP rewrite base for supply crawler (tests) |
| `LANDER_BASE_URL` | HTTP rewrite base for lander crawler (tests) |

---

## Preflight for `PARSER_SOURCE=all`

- `MONGO_URI`, `MONGO_DB`
- `PARSER_EXPORT_JSON` (optional)
- `GEMINI_API_KEY` + `PARSER_ICP_CLASSIFY` + `PARSER_GEO_CLASSIFY`
- `DISCORD_BOT_TOKEN` + `DISCORD_CHANNEL_IDS` + bot on guild
- `GITHUB_TOKEN` if `github` is enabled
- `TELEGRAM_API_ID` + `TELEGRAM_API_HASH` + session file
- Real URLs in `data/seeds/*.csv` (not `*.example` placeholders)

```bash
cp .env.example .env
docker compose run --rm parser config check
docker compose run --rm parser scan --source=forum,reddit --output=pretty
```

---

## Security

- Keep `.env` out of git.
- Revoke and rotate any leaked token in the provider portal.
- Production logs mask contacts (`MaskedContact`); do not log full email/telegram at Info.
