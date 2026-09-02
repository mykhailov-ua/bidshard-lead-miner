# VPS deploy playbook (boxed product)

Single path: fresh Ubuntu VPS + residential proxy -> parser 24/7 with leads in Mongo/JSONL.

Related: [CREDENTIALS.md](CREDENTIALS.md), [OPS.md](OPS.md).

---

## 1. Server requirements

| Item | Minimum | Full prod (boxed) |
| --- | --- | --- |
| OS | Ubuntu 22.04 / 24.04 | Ubuntu 24.04 LTS |
| RAM | 2 GB | **4 GB** (Mongo + Telethon bg); **8 GB** if `PARSER_LANDER_HEADLESS=true` on same host |
| vCPU | 1 | **2** (`PARSER_WORKERS=4`, `PARSER_SOURCE_CONCURRENCY=3`) |
| Disk | 20 GB | **40-80 GB** (Mongo, JSONL, `parser_runtime`, backups) |
| Egress | Residential HTTP proxy for forum/tgweb on datacenter IP | same |
| Secrets | `GEMINI_API_KEY`, `TELEGRAM_API_*`, `PARSER_PROXY_LIST` | + optional `DISCORD_*`, `GITHUB_TOKEN` |

Full prod profile: `config/env/.env.vps.example` (`forum,supply,lander,tgweb`, `PARSER_BG_TELEGRAM=1`).

### What to buy (defaults)

**Not RU providers** (Selectel, Timeweb, RuVDS, ...). Provider policies change - confirm stock and checkout terms before paying.

**VPS location:** Netherlands or Finland (EU egress; stable for this stack).

**Do not use datacenter VPS IP alone** for forum/tgweb on Cloudflare - set `PARSER_PROXY_LIST` to residential. Optional VPS Squid is datacenter egress only ([Appendix A](#appendix-a-vps-squid-datacenter-egress-only)).

### Recommended stack (no passport)

Picks below are for operators who refuse ID upload. All accept crypto or email-only checkout on standard VPS plans (verified against provider sites, 2026-08).

| Role | Pick | Plan (4 GB prod) | Price | Payment | Why |
| --- | --- | --- | --- | --- | --- |
| **VPS** | [Mynymbox](https://mynymbox.io/netherlands-vps) VPS NL-4 | 2 vCPU, 4 GB, 50 GB, Amsterdam | EUR 16/mo | BTC / XMR / card via BTCPay | No KYC on site; KVM; NL DC matches geo proxy |
| **VPS** (cheaper) | [NetherlandsVPS](https://netherlandsvps.com/) NL-4G-VPS | 4 vCPU, 4 GB, 60 GB NVMe, Amsterdam | $14/mo | BTC / USDT / ETH; no KYC for crypto | Best $/GB for no-ID; email only |
| **VPS** (ToS no-KYC) | [Packetra](https://packetra.com/hosting/cloud-hosting) Cloud VPS #3 | 4 vCPU, 4 GB, 60 GB, Finland | EUR 22.90/mo | BTC / XMR / PayPal | [Terms](https://packetra.com/terms) state no KYC at any stage; 24/7 support |
| **Residential proxy** | [DataImpulse](https://dataimpulse.com) | PAYG $1/GB, traffic never expires | from $5 trial | Card / BTC / ETH / USDT | No KYC on standard use; primary example in this repo |
| **Residential** (crypto only) | [Proxies.VISION](https://proxies.vision) | PAYG $1/GB | from $1 | BTC / USDT / card | Email signup; crypto without KYC |

**Do not use for parser VPS:**

| Provider | Why skip |
| --- | --- |
| [Njalla](https://njal.la/servers/) | Privacy domains/VPN OK; VPS is EUR 45+ for 4 GB, Ceph latency, mixed uptime reports |
| Hetzner, DigitalOcean, Scaleway, Vultr | Frequent passport + selfie on new accounts |
| EQVPS, Servury | No track record; not vetted for 24/7 prod |

**Budget mainstream (conditional KYC):** [Contabo](https://contabo.com) Cloud VPS M (4 vCPU, 8 GB, ~EUR 7/mo) or [OVHcloud](https://www.ovhcloud.com) VPS if you accept risk of [mandatory verification](https://help.contabo.com/en/support/solutions/articles/103000348466-why-do-i-need-to-verify-my-purchase-) (passport + utility bill when antifraud flags the order). Register from home IP, no VPN, real billing address.

### Monthly budget (full prod, rough)

| Item | Cost |
| --- | --- |
| VPS (Mynymbox NL-4 or NetherlandsVPS NL-4G) | EUR 16 or $14 / month |
| Residential 5-15 GB | $5-15 (DataImpulse) |
| Gemini API | $0 on free tier (watch RPM/RPD) |
| **Total** | ~$20-30 / month |

### Budget profile ($30 USD cap)

Hard cap: **VPS + residential <= $30/mo**. Do not run forum/lander/tgweb 24/7 through proxy (that burns 50-150 GB/mo).

| Item | Pick | Cost |
| --- | --- | --- |
| VPS | [NetherlandsVPS](https://netherlandsvps.com/) NL-4G-VPS | $14 |
| Proxy | [DataImpulse](https://dataimpulse.com) top-up | $10-12 (~10 GB/mo cap) |
| **Total** | | **$24-26** |

**Split traffic (required):**

| Mode | Sources | Proxy | Schedule |
| --- | --- | --- | --- |
| 24/7 `parser run` | `reddit,discord,supply` + Telethon bg | **none** (datacenter VPS IP is fine) | `PARSER_POLL_SEC=300` |
| CF crawl cron | `forum,tgweb,lander` | **residential** | 1-2x/day via `scripts/ops/cf-crawl-cron.sh` |

Setup:

```bash
cp .env.example .env
cat config/env/.env.budget.example >> .env
cp config/env/.env.proxy.local.example config/env/.env.proxy.local
# edit .env: GEMINI_API_KEY, TELEGRAM_API_*
# edit config/env/.env.proxy.local: PARSER_PROXY_LIST only

docker compose up -d
# CF crawl (manual or crontab):
bash scripts/ops/cf-crawl-cron.sh
```

Keep `PARSER_PROXY_LIST` out of root `.env` so the 24/7 container does not route Reddit API through residential.

**Expected proxy use:** ~5-10 GB/mo at 2 CF crawls/day on default prod seeds (~$5-10). Remaining budget headroom for extra tgweb runs.

**Realistic volume on $30:** hundreds to low thousands of unique leads/mo (reddit + telegram + periodic forum/tgweb), not 15k without a much larger seed surface and budget.

Profile file: `config/env/.env.budget.example`.

### Fully automated v1 profile

Single env bundle for discover -> triage -> defer -> CRM gate without manual channel lists:

```bash
cp .env.example .env
cat config/env/.env.auto.example >> .env
cat config/env/.env.residential.example >> .env
# set GEMINI_API_KEY, TELEGRAM_API_*, PARSER_PROXY_LIST
go run ./cmd/parser config check
```

Key flags: `PARSER_BG_TELEGRAM=1`, `PARSER_CHANNEL_TRIAGE=true`, `PARSER_GEMINI_DEFER=true`, `PARSER_CRM_WEBHOOK_AFTER_ANALYSIS=true`, `PARSER_PROXY_SOURCES=forum,tgweb,lander` (reddit/serp stay direct while proxy list is set).

Profile file: `config/env/.env.auto.example`.

### Wire residential proxy into `.env`

**DataImpulse** (primary; card or crypto; no passport on standard account):

1. Register at [dataimpulse.com](https://dataimpulse.com) (email only).
2. Top up from $5 (card, BTC, ETH, or USDT).
3. Copy login/password from dashboard -> Proxy Access.
4. Geo/session in username (provider dashboard):

```bash
PARSER_PROXY_LIST=http://user-country-de-session-prod1:PASS@gw.dataimpulse.com:823
```

**Proxies.VISION** (crypto only; email signup, no KYC):

1. Register at [proxies.vision/register](https://proxies.vision/register).
2. Top up from $1 via BTC / USDT / ETH.
3. Copy credentials from dashboard:

```bash
PARSER_PROXY_LIST=http://USER:PASS@connect.proxies.vision:8080
```

Sticky session or country targeting: set in dashboard username format (see provider docs).

**Do not use for residential:** IPRoyal (~$7/GB, card KYC possible on large top-ups), Bright Data / Oxylabs (enterprise, contracts). VPS Squid is datacenter egress only - not a residential replacement.

Verify before 24/7:

```bash
make proxy-check
make deploy-preflight
```

`deploy-preflight` runs `vps-preflight` then tgweb crawl under the eBPF probe (`parser_bpf_fd_delta`, thread drift). On non-Linux hosts BPF is skipped; run on staging VPS before ship. Skip BPF: `PARSER_BPF_PREFLIGHT=0 make deploy-preflight`.

More proxy modes: [OPS.md#proxy](OPS.md#proxy).

---

## 2. Install (ordered)

```bash
git clone <repo> lead-intent-processor
cd lead-intent-processor

cp .env.example .env
cat config/env/.env.vps.example >> .env
cat config/env/.env.tgweb.example >> .env
cat config/env/.env.residential.example >> .env

# Edit .env: GEMINI_API_KEY, TELEGRAM_API_ID/HASH, PARSER_PROXY_LIST
nano .env

mkdir -p data/export data/runtime var
sudo chown -R 10001:10001 data/export data/runtime   # parser UID in Docker image

make build
docker compose build
docker compose up -d mongo
docker compose up -d parser
```

Validate:

```bash
docker compose run --rm parser config check
make vps-preflight
```

Expected: `vps-preflight: ok`, log at `var/vps-preflight.log`.

---

## 3. Accept proof (before 24/7 trust)

```bash
make tgweb-green-accept          # accepted>=1, leads_written>=1
make forum-live-check            # forum raw>0 via proxy (XenForo seeds)
make prod-source-smoke           # supply + lander raw>0
```

Record run dirs under `var/green-accept-*`, `var/forum-live-*`, `var/prod-source-smoke-*`.

Telethon discover (optional bg loop):

```bash
docker compose run --rm -it parser telegram login --qr
make tgweb-discover-loop
```

---

## 4. 24/7 operations

| Task | Command |
| --- | --- |
| Tail logs | `docker compose logs -f parser` |
| Status | `docker compose ps` |
| Restart parser | `docker compose restart parser` |
| One-shot scan | `docker compose run --rm parser scan --source=forum,supply` |
| Backup Mongo | `make backup` |
| Restore | `make restore DUMP=backups/mongo-.../dump.gz` |
| Config drift | `docker compose run --rm parser config check` |

Parser container runs as UID **10001**. Bind mounts `data/export` and named volume `parser_runtime` hold Telethon session + JSONL export across rebuilds.

Healthcheck: `docker compose ps` should show parser **healthy** after Mongo is up (`parser config check` inside container).

---

## 5. Observability and SLA gates (BOX-7)

Enable metrics (set in `config/env/.env.vps.example`):

```bash
PARSER_METRICS_ADDR=:9465
```

Scrape `http://<vps-host>:9465/metrics`. Example alerts: [PROMETHEUS/parser-alerts.example.yml](PROMETHEUS/parser-alerts.example.yml).

Soak / smoke fail flags (`scripts/lib/soak_gate.sh`):

| Flag | Meaning |
| --- | --- |
| `accepted=0` | Pipeline rejects everything (geo, ICP, prescan, missing Gemini) |
| `leads_written=0` | No Mongo/export write (check `MONGO_URI`, `PARSER_EXPORT_JSON`) |
| `raw_total=0` | Crawl blocked (proxy, CF, bad seeds) - WARN in gate, FAIL in source smoke |

**Acceptance soak (Epic J):** after 2h prod-like `parser run`, `make acceptance-soak` runs jq gates on `data/export/leads.jsonl` (pending %, lander junk, CSS contacts, telegram High pain). See [OPS.md#acceptance-soak](OPS.md#acceptance-soak).

**Warm path:** `make warm-path-status` shows pending/DLQ counts and rescan env. Set `GEMINI_MODEL=gemini-3.6-flash` (or unset) in prod `.env` - see `config/env/.env.prod.example`.

Prometheus counters:

- `parser_leads_accepted_total` - accept path
- `parser_leads_written_total` - store upsert success
- `parser_junk_total` - reject reasons

---

## 6. Release checklist (BOX-6)

Before shipping a new image to VPS:

```bash
go test ./...
go build ./...
docker compose build parser
docker compose run --rm parser config check
PARSER_SEED_PROFILE=prod go run ./cmd/parser config check
```

On staging VPS with creds:

```bash
make deploy-preflight
make tgweb-green-accept
make forum-live-check
make prod-source-smoke
make acceptance-soak    # after 2h parser run on staging
```

After deploy:

```bash
docker compose pull   # if using registry tag
docker compose up -d --force-recreate parser
docker compose ps     # parser healthy
make backup
```

---

## 7. Troubleshooting

| Symptom | Fix |
| --- | --- |
| `permission denied` on export | `sudo chown -R 10001:10001 data/export data/runtime` |
| Parser unhealthy | `docker compose logs parser`; run `config check` manually |
| `GEMINI_API_KEY required` | Set key or disable flag (`PARSER_ICP_CLASSIFY_TGWEB=false` only for dev) |
| All crawls `raw=0` | `make proxy-check`; verify residential creds; see Appendix A for datacenter Squid |
| Telethon session lost | Use named volume `parser_runtime`; re-run `telegram login --qr` |

Deep dives: [OPS.md](OPS.md) (tgweb, proxy, BPF).

---

## Appendix A: VPS Squid (datacenter egress only)

Squid on a cheap VPS does **not** replace residential proxy for Cloudflare igaming. Use it for unified datacenter egress or non-CF targets.

```bash
scp -r scripts/vps-proxy user@YOUR_VPS:/tmp/bidshard-proxy
ssh user@YOUR_VPS 'cd /tmp/bidshard-proxy && cp env.example .env.local && sudo ENV_FILE=.env.local ./install-on-vps.sh'
# paste PARSER_PROXY_LIST from /root/bidshard-proxy.credentials into .env
make proxy-check
```

Local Docker Squid smoke (no VPS):

```bash
make vps-proxy-docker
./scripts/vps-proxy/print-env-snippet.sh   # paste into .env
make vps-proxy-down                        # stop
```

Full guide: [OPS.md#vps-squid-optional](OPS.md#vps-squid-optional). Scripts: [scripts/vps-proxy/README.md](../scripts/vps-proxy/README.md).

---

## 9. Optional box features (P3)

| Feature | Env / command |
| --- | --- |
| Telethon MTProto proxy | `TELEGRAM_PROXY_URL=socks5://user:pass@host:1080` (not `PARSER_PROXY_LIST`) |
| Playwright in Docker | `make docker-headless-build`; `PARSER_LANDER_HEADLESS=true` |
| BPF release gate | `PARSER_BPF_GATE=1 make tgweb-crawl-bpf`; `make bpf-release-gate SESSION=var/bpf-session/...` |
| BPF leak probe | `make tgweb-bpf-leak-gate` (or `PARSER_BPF_LEAK_GATE=1 make tgweb-crawl-bpf`); `make bpf-leak-gate SESSION=...`; scrape `:9464` for `parser_bpf_fd_delta` |
| CRM webhook | `PARSER_CRM_WEBHOOK=true`; `PARSER_CRM_WEBHOOK_URL=http://crm-bot:8080/v1/leads`; shared `PARSER_CRM_WEBHOOK_SECRET` / `CRM_WEBHOOK_SECRET`; parser needs `PARSER_LEAD_STATUS_ENABLED=true` for inbox |

### CRM sidecar (docker compose)

```bash
cp .env.example .env
# set CRM_WEBHOOK_SECRET, MONGO_URI; expose UI via Caddy (config/caddy/Caddyfile.example)
docker compose up -d mongo crm-bot
docker compose run --rm crm-bot config check
bash scripts/dev/crm_bot_smoke.sh
```

Compose service `crm-bot` publishes `127.0.0.1:8080` on the host and uses `MONGO_URI=mongodb://mongo:27017` inside the bridge network. Remote CLI: `CRM_API_URL=http://127.0.0.1:8080 ./bin/crm-bot api stats`. Host-local dev: `make build-crm-bot && ./bin/crm-bot run`.

---

## 8. Dev vs prod seeds

Set `PARSER_SEED_PROFILE=prod` (included in `.env.vps.example`).

| Profile | Forum | Supply | Lander |
| --- | --- | --- | --- |
| `dev` (default) | `forum_threads.csv` (fixture) | `domains.csv` | `lander_urls.csv` |
| `prod` | `forum_threads.live.csv` | `domains.prod.csv` | `lander_urls.prod.csv` |
