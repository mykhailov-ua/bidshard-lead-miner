# VPS deploy playbook (boxed product)

Single path: fresh Ubuntu VPS + residential proxy -> parser 24/7 with leads in Mongo/JSONL.

Related: [CREDENTIALS.md](CREDENTIALS.md), [OPS.md](OPS.md).

---

## 1. Server requirements

| Item | Minimum |
| --- | --- |
| OS | Ubuntu 22.04 / 24.04 |
| RAM | 2 GB (4 GB with Mongo + Telethon) |
| Disk | 20 GB |
| Egress | Residential HTTP proxy for forum/tgweb on datacenter IP |
| Secrets | `GEMINI_API_KEY`, `TELEGRAM_API_*`, `PARSER_PROXY_LIST` |

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

Scrape `http://<vps-host>:9465/metrics`. Example alerts: [prometheus/parser-alerts.example.yml](prometheus/parser-alerts.example.yml).

Soak / smoke fail flags (`scripts/lib/soak_gate.sh`):

| Flag | Meaning |
| --- | --- |
| `accepted=0` | Pipeline rejects everything (geo, ICP, prescan, missing Gemini) |
| `leads_written=0` | No Mongo/export write (check `MONGO_URI`, `PARSER_EXPORT_JSON`) |
| `raw_total=0` | Crawl blocked (proxy, CF, bad seeds) - WARN in gate, FAIL in source smoke |

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
make vps-preflight
make tgweb-green-accept
make forum-live-check
make prod-source-smoke
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
| BPF leak probe | `PARSER_BPF_LEAK_GATE=1` + `make bpf-leak-gate SESSION=...`; scrape `:9464` for `parser_bpf_fd_delta` |
| CRM webhook | `PARSER_CRM_WEBHOOK_URL=https://crm.example/hook`; optional `PARSER_CRM_WEBHOOK_SECRET` |

---

## 8. Dev vs prod seeds

Set `PARSER_SEED_PROFILE=prod` (included in `.env.vps.example`).

| Profile | Forum | Supply | Lander |
| --- | --- | --- | --- |
| `dev` (default) | `forum_threads.csv` (fixture) | `domains.csv` | `lander_urls.csv` |
| `prod` | `forum_threads.live.csv` | `domains.prod.csv` | `lander_urls.prod.csv` |
