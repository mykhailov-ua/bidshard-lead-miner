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
