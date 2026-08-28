# Operations: tgweb, proxy, VPS Squid, eBPF

---

## tgweb deploy

Crawl affiliate sites from `discovered_telegram_domains.json`. Empty registry -> no pending domains.

### Quick start

```bash
cp .env.example .env
cat config/env/.env.tgweb.example >> .env
# set GEMINI_API_KEY=... and PARSER_PROXY_LIST=... if needed

make tgweb-seed
make preflight-tgweb
make tgweb-crawl
# targeted:
make tgweb-crawl DOMAINS=blask.com,vietnam.vn
# optional BPF baseline (Linux + sudo; summary under var/bpf-session/tgweb-<ts>/):
make tgweb-crawl-bpf
```

On a datacenter host, add residential proxy:

```bash
cat config/env/.env.residential.example >> .env
cat config/env/.env.vps.example >> .env   # prod seeds + paths
make deploy-preflight                   # vps-preflight + BPF leak gate (Linux + sudo)
make vps-preflight                        # proxy-check + preflight + config check only
make tgweb-crawl-residential
make tgweb-green-accept                   # requires GEMINI_API_KEY; exits 0 when accepted>=1
```

Telethon discover loop (session required):

```bash
docker compose run --rm -it parser telegram login --qr
make tgweb-discover-loop                    # discover -> prune -> crawl
```

### Fill the domain registry

**Dev seed (2 affiliate fixture domains):**

```bash
make tgweb-seed
# or: cp data/runtime/discovered_telegram_domains.json.example data/runtime/discovered_telegram_domains.json
```

Default example: `buylink.pro`, `topxpartners.com` (used in processor/tgweb tests; pass geo + prescan when page content matches affiliate context). Soak script refreshes this fixture by default (`SOAK_DEV_FIXTURE=1`).

**Host Telethon sidecar** (discover / scrape bg jobs):

```bash
make venv
export PARSER_TELETHON_PYTHON=$PWD/.venv/bin/python   # optional; auto-detected when unset
go run ./cmd/parser telegram login --qr
```

**Production discover** - needs `TELEGRAM_API_ID`, `TELEGRAM_API_HASH`, `make venv`, Telethon session:

```bash
docker compose run --rm -it parser telegram login --qr
make tgweb-discover
make tgweb-prune
make tgweb-crawl
```

**Re-crawl specific domains** - domain must exist in registry; clear `crawled_at`:

```bash
go run ./cmd/parser telegram domains reset buylink.pro topxpartners.com
make tgweb-crawl DOMAINS=buylink.pro,topxpartners.com
```

### Docker one-shot

| File | Use |
|------|-----|
| `docker-compose.yaml` | `parser run` 24/7 + mongo |
| `docker-compose.tgweb.yaml` | one-shot `tgweb-crawl`, `tgweb-discover`, `tgweb-prune` |

```bash
docker compose build parser
docker compose -f docker-compose.tgweb.yaml --profile tgweb run --rm tgweb-crawl
docker compose -f docker-compose.tgweb.yaml --profile tgweb run --rm tgweb-crawl --domains buylink.pro,topxpartners.com
```

Rebuild after Go changes - otherwise the container runs a stale binary. `network_mode: host` lets Mongo use `localhost:27017`.

### Make targets

| Target | Action |
|--------|--------|
| `make tgweb-seed` | Copy example registry if missing |
| `make preflight-tgweb` | Config, proxy hints, registry, TLS smoke (proxy), HTTP smoke |
| `make tgweb-discover` | Telethon discover |
| `make tgweb-prune` | Prune invalid hosts from registry |
| `make tgweb-domains-prune` | Host prune via `parser telegram domains prune` (no Docker) |
| `make tgweb-crawl` | Seed + preflight + docker crawl |
| `make tgweb-crawl-bpf` | Same + optional BPF baseline (`PARSER_BPF_BASELINE=1`, Linux + sudo) |
| `make vps-preflight` | Proxy-check + tgweb preflight + `parser config check` |
| `make deploy-preflight` | `vps-preflight` + tgweb crawl under eBPF leak gate (Linux + sudo) |
| `make tgweb-bpf-leak-gate` | BPF leak gate only (`scripts/tgweb/bpf-leak-preflight.sh`) |
| `make tgweb-crawl-residential` | Crawl (requires `PARSER_PROXY_LIST` in `.env`) |
| `make docker-tgweb-crawl` | Docker only (`DOMAINS=...`) |
| `make proxy-check` | Curl through first proxy |

Env overrides: `DOMAINS=...`, `TGWEB_CRAWL_MODE=host` for `go run` instead of docker.

### Mode matrix

| Scenario | `PARSER_PROXY_LIST` | `PARSER_LANDER_HEADLESS` | Run |
|----------|---------------------|--------------------------|-----|
| Laptop, CF sites | empty (direct) | `false` | `TGWEB_CRAWL_MODE=host make tgweb-crawl` |
| Server, CF | residential | `false` | `make tgweb-crawl-residential` |
| Simple sites, no CF | VPS or direct | `false` | docker / host |
| Hard CF + JS | residential | `true` (host only) | not Docker MVP |

Headless in default Alpine Docker image is not supported. Use `make docker-headless-build` and `docker-compose.headless.yaml` (Playwright + Chromium), or on the host run `make venv`, then `playwright install chromium` (pip package is already in `.venv`).

### tgweb troubleshooting

| Symptom | Fix |
|---------|-----|
| Registry missing / 0 domains | `make tgweb-seed` or `make tgweb-discover` |
| `emitted=0`, `deferred_retry=N` | `docker compose build parser` |
| Instant hard fail on all URLs | Remove placeholder from `PARSER_PROXY_LIST` |
| Mongo ping failed | `docker compose up -d mongo` |
| `--domains` ignores a host | Add domain to registry JSON first |

Env profiles: `config/env/.env.local.example`, `.env.tgweb.example`, `.env.residential.example`, `.env.vps.example`, `.env.global.example`.

### tgweb prescan mode

| Value | Behavior | When |
| --- | --- | --- |
| `aggressive` (default) | Site LPR bypasses keyword prescan / some hard_reject | Dev crawl tuning |
| `strict` | Require affiliate keyword hits on page text | Prod / geo compliance |

Set `PARSER_TGWEB_PRESCAN_MODE=strict` on VPS (see `config/env/.env.vps.example`).

### Geo compliance registry hygiene

After Telethon discover or SERP harvest, drop RU/BY hosts before crawl:

```bash
make tgweb-domains-prune
# or: go run ./cmd/parser telegram domains prune
make tgweb-prune   # same via Docker profile
```

Manual audit (gitignored runtime files):

- `data/runtime/discovered_telegram_channels.json` - remove CIS-heavy channels (Cyrillic titles, tgstat noise)
- `data/runtime/discovered_telegram_domains.json` - prune removes `.ru`/`.by`/`.su` TLDs automatically

Re-run discover after editing `config/discover.icp.json`: `make tgweb-discover`.

Cross-source entity graph uses forum_user keys, heat tiers, and CRM inbox sort. Run `make entity-heat-report` weekly; repair false merges with `crm-bot db entity-split`.

---

## Global discovery (non-RU bias)

After geo compliance (P0-P2), widen recall outside RU/BY:

### Locale keyword overlays

```env
KEYWORDS_LOCALE=es,pt
```

Loads `data/keywords-es.json` and `data/keywords-pt.json` on top of base + gray registry. Rotate weekly (e.g. `es,pt` then `de,fr`) to cover EU markets without enabling RU locale files.

Profile: `config/env/.env.global.example` (merge into `.env`).

### HTTP source bundle

`PARSER_SOURCE=all` = forum, supply, reddit, discord, serp (**lander opt-in**; not in `all`). Does **not** include tgweb, github, ct, or reviews.

Recommended prod mix (accept-quality Tier 0):

```env
cat config/env/.env.precision.example >> .env
# or manually:
PARSER_SOURCE=forum,supply,reddit,serp,reviews,tgweb
PARSER_ICP_CLASSIFY=true
PARSER_EMBED_PRESCAN=true
PARSER_CHANNEL_TRIAGE=true
PARSER_SOURCE_DISABLE_GOVERNOR=true
```

Do **not** add `github` to `PARSER_SOURCE` without the Tier1 pain gate (CKAN/`keitaroinc` false positives). `GITHUB_SEARCH_QUERIES` is fine for discovery only.

Add `ct` / `reviews` when seeds are populated. Add `lander` only when LPR-only Tier1 gate is deployed.

### Accept-quality Tier 1 (code gates)

Hot path in `internal/pipeline/processor.go` after contact extract:

| Gate | Blocks |
| --- | --- |
| `extract.FilterJunkContacts` | CSS `@media`, `@github` false positives |
| `filter.LanderRequiresEmailOrSkype` | lander with telegram-only / CSS contacts |
| `filter.GitHubRequiresPainContext` | CKAN/vendor issues without tracker pain |
| `filter.TelegramChannelSelfBroadcast` | channel posts where contact == channel handle |
| `filter.TelegramInviteWithoutBuyerIntent` | invite-channel promos without pain/intent |
| `scoring.PhraseMatches` word boundaries | `keitaro` inside `keitaroinc` no longer scores |
| defer `pilot-qualified` | only warm engage sets `pilot_qualified` |

### Accept-quality Tier 2 (ranking + CRM sort)

| Mechanism | Behavior |
| --- | --- |
| `HasBuyerIntentSignal` | displacement boost only on first-person / question intent |
| `StaticSourceBoost` | penalties for `telegram:invite`, `lander`, `github`, news channels |
| `normalizeSourceKey` | aggregates `telegram:invite:*` junk stats |
| `ComputeEngagePriority` | +8 reddit, +6 forum; CRM inbox sorts by `engage_priority` |
| `CRM_ENGAGE_PRIORITY_MIN` | default 70 for `crm-bot api list` inbox and `entity inbox` |

### Accept-quality Tier 3 (source mix + routing)

| Mechanism | Behavior |
| --- | --- |
| `PARSER_SOURCE_PRIORITY` | coordinator collects reddit/forum before lander/github |
| `PARSER_LANDER_OUTREACH` | default false: `lander:*` feeds registry intel only, not CRM |
| `PARSER_INTENT_CLASSIFY` | Gemini buyer_search gate on github, lander (outreach), webpain, TG channels |
| `filter.TelegramChannelBroadcastReject` | channel broadcast without buyer pain -> junk |
| `sourcedisable.MinRawForFamily` | governor disables lander/github/invite at 40 raw (vs 100 default) |
| `crm-bot db purge` | clean legacy junk: `--source-prefix lander: --score-max 50 --yes` |

### Webpain P1 (crawl parity)

| Mechanism | Behavior |
| --- | --- |
| `lander.TextForContactExtract` | webpain uses same HTML/RSC extract as lander/tgweb |
| `PickPageLPR` | on-domain email or skype required at crawl emit |
| `webpain.FilterPipelineContacts` | processor collapses to one LPR, drops role/CSS contacts |
| `CrawlHTML` | preserved for stack/debug on accepted webpain leads |

### Lander P2 (extraction hygiene)

| Mechanism | Behavior |
| --- | --- |
| `skipExtractString` | `flattenJSON` / RSC skip CDN URLs, asset paths, base64, webpack chunk IDs |
| `inlineStyleRe` | strip `style="..."` before visible body parse (no CSS `@media` leakage) |
| contact-region prefer | footer/mailto/skype found -> skip full 50k body dump in `ExtractStaticLandingText` |

### Supply P3 (ads.txt hygiene)

| Mechanism | Behavior |
| --- | --- |
| `validate.AcceptEmail` | `collectContacts` and `CONTACT=` directives reject role/disposable mail |
| `BuildSnippet` samples | up to 3 lines: `CONTACT=`, partner DIRECT row, seller email |
| `AcceptCascadePartnerDomain` | SSP/CDN partners (rubicon, pubmatic, openx, ...) skip tgweb fan-out |

### Gemini discover diff

Suggest new global dorks from junk/accept stats:

```env
GEMINI_DISCOVER_DIFF_EVERY=5
GEMINI_DISCOVER_DIFF_DIR=data/suggestions
GEMINI_API_KEY=...
```

After each cold-path cycle, review files under `data/suggestions/`. Merge approved `telegram_search` / `serp_dorks` lines into `config/discover.icp.json` manually (no auto-merge). Then `make tgweb-discover`.

### Weekly ops checklist

Run once per week on a host with Mongo + optional Gemini:

1. `make entity-heat-report` - entity heat negative selection + semantic_cluster collisions
2. `parser suggestions list` - review pending keyword/discover/pain_vocab diffs
3. `parser suggestions preview --file ...` then `parser suggestions apply --file ...` after backup
4. `parser cold report` - junk aggregate report (or wait for scheduled cold-path cycle)
5. `crm-bot api stats` - inbox backlog; mark spam in CRM to feed `webhook_feedback` audit
6. Check `data/suggestions/dork_rank_*.json` and `webhook_audit_*.json` when present

### Entity heat report (weekly soak)

Negative-selection metrics for cross-source entity graph quality:

```bash
make entity-heat-report
# or: bash scripts/dev/entity_heat_report.sh
```

Writes `data/suggestions/entity_heat_YYYYMMDD.txt` (hot + low confidence, under-hot cold tiers, forum_user hit rate) and `entity_pain_YYYYMMDD.txt` (top `unified_pain` strings for keyword feedback into `data/keywords.json`).

False merge repair: `crm-bot db entity-split --id ENTITY_ID --hash HASH --yes` (on-server, Mongo reachable).

### Proxy egress for global SERP/tgweb

On datacenter VPS, use residential proxy with geo session in username (provider-specific). Provider picks: [DEPLOY.md#recommended-stack-no-passport](DEPLOY.md#recommended-stack-no-passport).

```env
PARSER_PROXY_LIST=http://user-country-us-session-abc:pass@gw.dataimpulse.com:823
```

Verify before crawl: `make preflight-tgweb`. See [config/env/.env.residential.example](../config/env/.env.residential.example).

---

## Proxy

Set HTTP proxies in one variable: **`PARSER_PROXY_LIST`**.

Chain: `PARSER_PROXY_LIST` -> `internal/httpclient` (uTLS + Chrome fingerprint) -> lander/tgweb fetch.

### Pick a mode

| Mode | Cloudflare (igaming) | Use when |
|------|----------------------|----------|
| Empty `PARSER_PROXY_LIST` (direct) | Often works | Dev on home IP |
| Residential (DataImpulse, Proxies.VISION, IPRoyal, ...) | Primary path | Parser on server/cloud |
| Your VPS Squid | Poor | Non-CF egress isolation only - see [VPS Squid](#vps-squid-optional) |

Cloudflare treats home IPs differently from datacenter IPs. VPS Squid is still a datacenter IP.

### Format

```bash
PARSER_PROXY_LIST=http://user:pass@host:port
# DataImpulse (~$1/GB PAYG):
PARSER_PROXY_LIST=http://USER:PASS@gw.dataimpulse.com:823
# Proxies.VISION ($1/GB, crypto or card; host/port from dashboard):
PARSER_PROXY_LIST=http://USER:PASS@connect.proxies.vision:8080
# IPRoyal:
PARSER_PROXY_LIST=http://USER:PASS@geo.iproyal.com:12321
```

Geo/session often goes in the **username** (provider dashboard). Multiple URLs -> round-robin. Per-proxy defaults: `PARSER_PROXY_RPS=0.5`, `PARSER_PROXY_BURST=1`, `PARSER_PROXY_COOLDOWN=10m`. On 403/503/429 block -> cooldown per URL; pool waits instead of reusing cooled endpoints.

Residential signup without passport: DataImpulse (card or crypto) or Proxies.VISION (crypto). See [DEPLOY.md#recommended-stack-no-passport](DEPLOY.md#recommended-stack-no-passport).

### Verify

```bash
make preflight-tgweb
make proxy-check
```

### Limits

| Case | Residential | VPS Squid | Direct (home) |
|------|-------------|-----------|---------------|
| Open HTML (blask.com) | OK | OK | OK |
| CF challenge | Often OK | Fail | Often OK |
| Placeholder URL in `.env` | Instant fail | Instant fail | - |

Do not put fake proxy URLs in `.env` - crawl hard-fails immediately.

---

## VPS Squid (optional)

Datacenter Squid does **not** replace residential for Cloudflare igaming. Use it for simple 403s, unified egress, or testing `PARSER_PROXY_LIST`.

### Requirements

| | |
|---|---|
| VPS | Spare Ubuntu host on parser VPS or cheapest NL plan (~EUR 16/mo) |
| OS | Ubuntu 22.04 / 24.04 |
| Port | **3128** (Squid) |

### Install on VPS

```bash
scp -r scripts/vps-proxy user@YOUR_VPS_IP:/tmp/bidshard-proxy
ssh user@YOUR_VPS_IP
cd /tmp/bidshard-proxy
cp env.example .env.local # set PROXY_USER, PROXY_PASS, VPS_PUBLIC_IP
chmod +x *.sh
sudo ENV_FILE=.env.local ./install-on-vps.sh
```

Lock down firewall - allow 3128 only from your parser host IP:

```bash
sudo ufw allow from YOUR_PARSER_IP to any port 3128 proto tcp
sudo ufw allow OpenSSH && sudo ufw enable
```

On the parser host:

```bash
ssh user@YOUR_VPS_IP 'sudo grep PARSER_PROXY_LIST /root/bidshard-proxy.credentials'
# paste into .env:
PARSER_PROXY_LIST=http://parser:YOUR_PASS@YOUR_VPS_IP:3128
make proxy-check
```

Rotate multiple VPS URLs: comma-separate in `PARSER_PROXY_LIST`.

### Local Docker Squid (test without VPS)

```bash
cp scripts/vps-proxy/env.example scripts/vps-proxy/.env.local
./scripts/vps-proxy/setup-docker-proxy.sh
./scripts/vps-proxy/print-env-snippet.sh # paste into .env
make proxy-check
# stop: docker compose -f scripts/vps-proxy/docker-compose.proxy.yaml down
```

Scripts: `scripts/vps-proxy/install-on-vps.sh`, `setup-docker-proxy.sh`, `check-proxy.sh`, `squid.conf`.

---

## eBPF dev probe (Linux)

Syscall/sched/net probe for tgweb crawl and parser pipeline analysis. **Dev only** - Linux 5.8+, root/CAP_BPF, clang or Docker build.

```bash
make bpf-dev
sudo make bpf-session-start

# run crawl in another terminal
docker compose run --rm parser telegram web --domains blask.com

sudo make bpf-session-stop
# -> var/bpf-session/<ts>/bpf-report.md
```

| Variable | Default | Meaning |
|----------|---------|---------|
| `PARSER_BPF_NATIVE` | `1` in session | Track parser/mongo/telegram via pgrep |
| `PARSER_BPF_TARGETS` | `parser,mongo,telegram` | Roles to resolve |
| `PARSER_BPF_SAMPLE_RATE` | `1` | Syscall sample rate (try 10 on laptop) |
| `PARSER_BPF_SLOW_US` | `10000` | Slow syscall threshold (us) |
| `PARSER_BPF_DUMP_INTERVAL` | `30` | Dump `summary.json` interval (s) |
| `PARSER_BPF_METRICS_ADDR` | `:9464` | Prometheus metrics |

Artifacts under `var/bpf-session/<utc>/`: `bpf/maps/summary.json`, `bpf-report.md`, `bpf/collector.log`.

Build deps: `clang llvm libbpf-dev linux-libc-dev bpftool` - or `make bpf-dev` builds `.o` in a container.

Targets: `parser`, `mongo`, `telegram` (see `scripts/perf/bpf_resolve_targets.sh`). Env prefix: `PARSER_BPF_*`.
