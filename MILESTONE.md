# Milestone — bidshard-lead-miner (active backlog)

**Product:** gray-market lead parser for [BidShard](../bidshard) (Starter ICP ~80%, Pro ~15%).  
**Scope:** worldwide **except RU/BY**; LinkedIn **out of scope**.  
**Policy:** [README.md](README.md) — pipeline, ICP, sources catalog.

**Anti-slop:** [.cursor/rules/ai-slop.mdc](.cursor/rules/ai-slop.mdc) (adapted from [bidshard/.cursor/rules/ai-slop.mdc](../bidshard/.cursor/rules/ai-slop.mdc)).  
**Closed P0 archive:** §1 below. **Open work:** §2 (P1), §3 (P2).

**Last updated:** 2026-08-16

---

## 0. Cross-cutting DoD (every milestone)

Applies to all §2–§3 items unless a row explicitly overrides.

### 0.1 Engineering DoD

- [x] Feature behind env flag or source name (no silent behavior change in prod without config).
- [x] `config.Load()` + `.env.example` documented.
- [x] Wired in `internal/app/deps.go` (or source `registry.go`).
- [x] Unit test proves core behavior; test **fails** if implementation removed.
- [x] `go test ./...` green in PR (paste output or CI link).
- [x] No secrets in diff; contacts masked in Info logs.

### 0.2 Anti-slop DoD ([ai-slop.mdc](.cursor/rules/ai-slop.mdc))

- [ ] No `[x]` on milestone row without tests **or** explicit `not verified (needs GEMINI_API_KEY / Docker / network)` note.
- [ ] New HTTP source: `httptest.Server` in `*_test.go`, not live API in unit tests.
- [ ] Gemini feature: uses shared `Client` + `QuotaLimiter`; not raw `http.Client` per call site.
- [ ] Processor change: order preserved — `geo → lang → hard_reject → … → score → dedup → enrich/gemini → mx → upsert`.
- [ ] Mongo fields: `model.Lead` ↔ `sink.LeadDoc` ↔ `ToLeadDoc()` stay in sync.

### 0.3 Verification bundle (baseline)

```bash
go build ./...
go test ./... -count=1
bash scripts/ci/check_parser_slop.sh    # if present; else go vet ./...
```

Optional (integration, not required for every PR):

```bash
MONGO_URI=mongodb://localhost:27017 go test ./internal/sink/... -tags=integration -count=1
GEMINI_API_KEY=... go run ./cmd/parser scan
```

---

## 1. P0 — closed (2026-08-16)

Shipped in-tree. Do **not** re-open without regression ticket.

| ID | Track | Shipped | Key paths |
| :--- | :--- | :--- | :--- |
| P0-1 | CPU geo + lang + hard_reject | [x] | `internal/geo/`, `internal/filter/`, `data/keywords.json` |
| P0-2 | Scoring: spend gate, competitor stack, source rep | [x] | `internal/scoring/spend.go`, `competitor.go`, `source.go` |
| P0-3 | Gemini cold path (junk analyze, report, keyword diff, embed dedup) | [x] | `internal/coldpath/`, `internal/gemini/` |
| P0-4 | Gemini hot path (ICP, geo classify) + rate limits | [x] | `internal/gemini/{icp,geo,limiter,transport}.go` |
| P0-5 | Sources: telegram sidecar, forum, supply, lander, reddit, `all` | [x] | `internal/sources/`, `sources/telegram/` |
| P0-6 | Enrichment: RDAP, DNS CNAME fingerprint, Gravatar, time decay | [x] | `internal/enrich/`, `internal/scoring/decay.go` |
| P0-7 | Discord source (Bot API) | [x] | `internal/sources/discord/` |
| P0-8 | Runtime: worker pool, breaker, dedup, bulk store, slog | [x] | `internal/app/`, `internal/pipeline/`, `internal/sink/` |

**P0 regression gates:**

```bash
go test ./internal/pipeline/... ./internal/gemini/... ./internal/enrich/... -count=1
go test ./internal/sources/... -count=1
```

---

## 2. P1 — quality, CRM, and channel depth

**Parser weight (README):** iGaming FTD / EU-LATAM Starter, small OpenRTB Pro, conversion measurement.

### 2.1 P1-1 — Lead lifecycle in Mongo

**Goal:** measure conversion by source/keyword; sync with `bidshard-leads` CRM outreach.

| | |
| :--- | :--- |
| **Status** | [x] |
| **ARPU** | All tiers — ops efficiency |

**Scope:**

- `model.Lead` + `sink.LeadDoc`: `status`, `status_at`, `outreach_channel`, `pilot_qualified` (bool).
- Enum: `new` → `contacted` → `replied` → `pilot` → `paid` → `dead` | `nurture`.
- `sink.Store`: `UpdateStatus(hash_id, status)` — **does not** break `$setOnInsert` dedup on first insert.
- Optional webhook or NDJSON export hook for `bidshard-leads` consumer.

**DoD:**

- [x] Default `status=new` on upsert; existing leads unchanged on duplicate `hash_id`.
- [x] `PARSER_LEAD_STATUS_ENABLED=true` gates writes (default false until CRM wired).
- [x] Index on `status` + `ts` for digest queries.
- [x] Test: upsert twice → status not overwritten without explicit `UpdateStatus`.
- [x] Test: `UpdateStatus` transitions `new` → `contacted`.
- [ ] `.env.example` + README one paragraph on CRM handoff.

**Anti-slop:**

- Do not claim CRM integration done without documenting handoff format to `bidshard-leads`.
- `[x]` only if integration test or documented manual curl against Mongo.

**Verification:**

```bash
go test ./internal/sink/... -count=1
# MONGO_URI=... go test ./internal/sink/... -tags=integration -run Status
```

---

### 2.2 P1-2 — Adaptive keyword registry

**Goal:** reduce false positives from cold-path `false_negative` feedback without manual JSON edits every week.

| | |
| :--- | :--- |
| **Status** | [x] |
| **ARPU** | Starter — less junk in CRM |

**Scope:**

- Extend `data/suggestions/keywords_pending_*.json` workflow: track `precision` / `junk_rate` per keyword id in Mongo `keyword_stats`.
- Cold path: after N junk reports, auto-`weight` adjust ±5 (floor 5, cap 30) or `enabled: false` when junk_rate > 30% (min 20 samples).
- Human approve file still required before merge to `keywords.json` (no auto-write to prod registry).

**DoD:**

- [x] `keyword_stats` collection + `sink.KeywordStatsStore`.
- [x] `coldpath.Service` records match outcome (accepted vs junk reason) per keyword hit.
- [x] Suggestion JSON includes `suggested_weight`, `evidence_count`, `junk_rate`.
- [x] Test: high junk_rate keyword → suggestion contains `enabled: false` recommendation.
- [x] Test: no direct write to `data/keywords.json` from code path.

**Anti-slop:**

- Do not mark done if only Gemini `BuildKeywordDiff` exists without stats loop (that part is P0).
- Precision must be computed from real counters, not hardcoded in prompt.

**Verification:**

```bash
go test ./internal/coldpath/... ./internal/scoring/... -count=1
```

---

### 2.3 P1-3 — Multilingual keyword overlay

**Goal:** match README geo/lang policy — EN primary; ES/PT/PL/DE/FR pain phrases without false RU/BY pass.

| | |
| :--- | :--- |
| **Status** | [x] |
| **ARPU** | LATAM / EU Starter |

**Scope:**

- `data/keywords-{es,pt,pl,de,fr}.json` overlays (same schema as `keywords.json`).
- `Registry.LoadWithOverlay` loads base + gray + locale file from `KEYWORDS_LOCALE` or auto-detect from snippet (optional, default off).
- `model.Lead`: `lang` field (`en|es|pt|…|unknown`).
- `filter/lang.go`: do not reject short ES/PT buyer phrases with pain keywords.

**DoD:**

- [x] At least **20** intent/pain phrases per locale file (reviewed, not raw MT).
- [x] `KEYWORDS_LOCALE_PATH` or `KEYWORDS_LOCALE=es` env.
- [x] Test: Spanish fixture with `voluum alternativa` scores ≥ Medium.
- [x] Test: long Cyrillic-only still rejected.
- [ ] Gemini ICP prompt lists `lang` in output (optional field).

**Anti-slop:**

- Do not auto-translate entire registry via Gemini in CI — static files only for v1.
- `[x]` requires fixture tests per locale, not one English test.

**Verification:**

```bash
go test ./internal/scoring/... ./internal/filter/... -count=1
```

---

### 2.4 P1-4 — Domain/email blacklist

**Goal:** hard-reject competitor infra, seed spam, and `support@`/`ads@` domains before scoring.

| | |
| :--- | :--- |
| **Status** | [x] |
| **ARPU** | Ops time |

**Scope:**

- `data/blacklist_domains.txt`, `data/blacklist_emails.txt` (same loader pattern as `disposable_domains.txt`).
- `validate/blacklist.go`: `IsBlacklisted(email, domain)`.
- Processor: after extract, before enrich — silent or junk capture with `reason=blacklist` (configurable).

**DoD:**

- [x] `BLACKLIST_DOMAINS_PATH`, `BLACKLIST_EMAILS_PATH` env.
- [x] Loader at startup in `deps.go` with count log.
- [x] Test: `tracker@voluum.com` pattern blocked if in list.
- [x] Test: legitimate domain not blocked when absent from list.
- [ ] Document seed list maintenance in README.

**Anti-slop:**

- Blacklist is not disposable list duplicate — document difference in README.
- Do not conflate with `hard_reject` phrases (domain vs text).

**Verification:**

```bash
go test ./internal/validate/... ./internal/pipeline/... -count=1
```

---

### 2.5 P1-5 — Forum sources: STM / BHW / AffiliateFix (production seeds)

**Goal:** P0 forum adapter exists but seeds are fixtures — need real EN thread discovery.

| | |
| :--- | :--- |
| **Status** | [x] |
| **ARPU** | Starter — highest intent channel |

**Scope:**

- `data/seeds/forum_threads.csv` — production thread URLs (STM Tracking, BHW CPA, AffiliateFix).
- `forum/adapter.go`: respect `FORUM_BASE_URL` robots, rate limit, `PostedAt` from post date.
- `model.RawItem.PostedAt` populated for time decay.

**DoD:**

- [ ] ≥ **50** seed threads across ≥ **3** hosts (documented in CSV header).
- [ ] `FORUM_HOST_RPS` per-host limiter (default 1/s).
- [ ] Circuit breaker on 403/429 (`internal/breaker`).
- [ ] Test: HTML fixture parse extracts title + body + date.
- [ ] Test: geo reject on RU domain in fixture.
- [ ] `PARSER_SOURCE=forum` emits `forum:{host}/{slug}` with `PostedAt`.

**Anti-slop:**

- Do not mark done with only `testdata/` HTML — production CSV must exist (can be gitignored sample + example).
- Crawler must not hammer login walls — detect and skip with Warn log.

**Verification:**

```bash
go test ./internal/sources/forum/... -count=1
go run ./cmd/parser scan --source=forum --output=table
```

---

### 2.6 P1-6 — Warrior Forum / CPALead source

**Goal:** README P1 channel — US/EU nutra/sweeps pain harvesting.

| | |
| :--- | :--- |
| **Status** | [x] |
| **ARPU** | Starter nutra/sweeps |

**Scope:**

- `internal/sources/warrior/` (or extend forum adapter with host profile).
- Seed CSV + HTML parser for thread list and posts.
- Source tag: `warrior:{slug}`.

**DoD:**

- [x] `WARRIOR_SEED_PATH`, `WARRIOR_HOST_RPS` config.
- [x] Registered in `registry.go` (`warrior`, included in `all` if stable).
- [x] `httptest` HTML golden tests.
- [x] Geo + keyword prescan applied before emit.
- [ ] README row updated from P1 → shipped.

**Anti-slop:**

- Separate package if HTML shape differs from STM — do not fake-parse with broken regex.
- `[x]` requires registry test includes `warrior`.

**Verification:**

```bash
go test ./internal/sources/warrior/... -count=1
```

---

### 2.7 P1-7 — Pilot qualification score + tags

**Goal:** README checklist § «Квалификация лида на пилот» automated as tags, not manual CRM notes.

| | |
| :--- | :--- |
| **Status** | [x] |
| **ARPU** | Starter pilot conversion |

**Scope:**

- `scoring/pilot.go`: `PilotQualified(lead, text) (bool, []string)` — checks spend language, competitor stack, pain, VPS hints, USDT-ok phrases.
- `model.Lead.Tags`: append `pilot-qualified` or `pilot-nurture` + reason codes.
- Optional Gemini verify for borderline (uses existing quota).

**DoD:**

- [x] Table tests for 8 README checklist rows (pass/fail fixtures).
- [x] Tag visible in `LeadDoc` + NDJSON export.
- [x] `PARSER_PILOT_TAG=true` env (default true).
- [x] Does not auto-reject — only tags (human outreach decides).

**Anti-slop:**

- Tags are not ICP `hot` — document difference in code comment.
- Do not mark `[x]` without all 8 checklist cases as table tests.

**Verification:**

```bash
go test ./internal/scoring/... -run Pilot -count=1
```

---

### 2.8 P1-8 — Operator CLI (cobra subcommands)

**Goal:** replace raw `flag` + 90-line `.env.example` discovery with a discoverable, self-documenting CLI for operators and Docker entrypoints.

| | |
| :--- | :--- |
| **Status** | [x] |
| **ARPU** | Ops — onboarding time |

**Current state (2026-08-16):**

- Single entry: `cmd/parser/main.go` — stdlib `flag` only (9 flags).
- No subcommands, no `--version`, no config validation, no shell completion.
- ~50 runtime knobs live only in env (`PARSER_*`, `GEMINI_*`, `MONGO_*`, …); README is the operator manual.
- Invocation is `go run ./cmd/parser -scan-once -source=stub` or `make run` — not a polished binary UX.

**Target UX:**

```text
parser run [--source stub|forum|…|all] [--output table|ndjson|quiet]
parser scan once …          # alias for current -scan-once
parser telegram run [--dry-run]
parser ingest [--fixture path.ndjson]
parser sources list         # print registry names + env prerequisites
parser config check         # validate .env: Mongo ping, token presence, seed files exist
parser version              # git sha / build time via -ldflags
parser completion bash|zsh|fish
```

Env remains source of truth for secrets and tuning; flags override env for the same keys already bridged today (`-source`, `-output`, `-log-format`).

**Scope:**

- Add `github.com/spf13/cobra` (and optionally `pflag` via cobra).
- Refactor `cmd/parser/main.go` → `cmd/parser/cmd/*.go` (root + subcommands).
- Map existing flags 1:1 first (no behavior change); deprecate bare `-scan-once` with hidden alias for one release.
- `config check` (read-only): load `config.Load()`, ping Mongo if `MONGO_URI` set, verify seed CSV paths, warn on missing optional tokens (Discord/GitHub/Gemini) without printing values.
- `sources list`: delegate to `internal/sources` registry metadata (name, included in `all`, required env keys).
- Embed examples in command `Long` help; `parser --help` shows quick-start block.
- `Makefile` / `docker-compose.yaml` entrypoint → `parser scan once` (or keep backward-compatible argv).
- Build: `go build -ldflags "-X main.version=…"` wired in Makefile `build` target.

**DoD:**

- [x] `go build -o bin/parser ./cmd/parser` produces binary with subcommands above.
- [x] `parser --help` and `parser <cmd> --help` render without running crawlers.
- [x] `parser config check` exits 0 on valid stub config; non-zero with actionable errors (missing seed file, bad `MONGO_URI`).
- [x] `parser version` prints version string (dev build ok: `dev` + `runtime.Version()`).
- [x] All existing README smoke commands have equivalent subcommand form documented in README § Usage.
- [x] Backward compat: `parser -scan-once -source=stub` still works (or documented breaking change + compose update).
- [x] Test: `cobra` command tree test — each registered subcommand has `RunE` and non-empty `Short`.
- [x] `.env.example` cross-links `parser config check` in header comment.

**Anti-slop:**

- Do not duplicate `config.Load()` field list as 40 new flags — subcommands + `config check` only; keep env for secrets.
- `[x]` requires `go test ./cmd/parser/...` with help-tree test, not just cobra scaffold.
- No interactive TUI in v1 — plain stdout/stderr.

**Verification:**

```bash
go test ./cmd/parser/... -count=1
go build -o bin/parser ./cmd/parser
./bin/parser config check
./bin/parser scan once --source=stub --output=table
./bin/parser sources list
```

**Suggested implementation order:**

1. Cobra root + `version` + migrate existing flags to `scan` / `run` / `telegram` / `ingest`.
2. `sources list` (registry introspection).
3. `config check` (Mongo ping + file existence).
4. Shell completion + README/Makefile/docker entrypoint update.

---

## 3. P2 — scale, discovery, and expensive crawlers

**Parser weight:** lower ARPU segments, proactive domain intel, optional headless.

### 3.1 P2-1 — Certificate Transparency (crt.sh)

**Goal:** discover `track.*`, `click.*`, `go.*` subdomains → feed supply/lander seeds.

| | |
| :--- | :--- |
| **Status** | [x] |
| **ARPU** | Pro / supply-side leads |

**Scope:**

- `internal/sources/ct/crawler.go` — query crt.sh JSON API, dedup domains, geo-filter TLD.
- Output: append to `data/seeds/ct_domains.csv` or emit directly as `ct:{domain}` raw items.
- Rate limit: ≤ 1 req / 2s (crt.sh etiquette).

**DoD:**

- [x] `CT_QUERIES=track,click,go` env (percent-encoded query).
- [x] `CT_MAX_RESULTS` cap per run.
- [x] RDAP geo reject before emit (reuse `enrich`).
- [x] `httptest` for JSON parse path.
- [x] Registered as `ct` source; **not** in default `all` until stable.

**Anti-slop:**

- Do not scrape crt.sh in unit tests — mock JSON only.
- `[x]` requires documented seed growth metric (domains emitted per run) in PR description.

**Verification:**

```bash
go test ./internal/sources/ct/... -count=1
```

---

### 3.2 P2-2 — Trustpilot / G2 / Capterra pain harvest

**Goal:** 1–2★ reviews on Voluum, Keitaro, RedTrack → intent snippets.

| | |
| :--- | :--- |
| **Status** | [x] |
| **ARPU** | Starter refugees |

**Scope:**

- `internal/sources/reviews/` — HTML or official API where licensed.
- Parse: rating ≤ 2, text, date, reviewer country if visible.
- Source: `reviews:trustpilot:{product}`.

**DoD:**

- [ ] Legal/ToS note in README (robots, rate limits; no login bypass).
- [x] Fixture HTML tests per site.
- [x] `PostedAt` from review date; time decay applies.
- [x] Geo reject on RU/BY reviewer country in text.
- [x] Opt-in source only (`PARSER_SOURCE=reviews`).

**Anti-slop:**

- Do not claim API access without key in config.
- Mock HTML must include negative review body — header-only page is not done ([ai-slop.mdc](.cursor/rules/ai-slop.mdc) header-only export pattern).

**Verification:**

```bash
go test ./internal/sources/reviews/... -count=1
```

---

### 3.3 P2-3 — Playwright headless (Next.js App Router)

**Goal:** README § «Уровень 3» — lander crawl when `__NEXT_DATA__` and RSC empty.

| | |
| :--- | :--- |
| **Status** | [x] |
| **ARPU** | lander / competitor stack |

**Scope:**

- Enable `internal/sources/lander/headless.go` — Playwright pool (max 2 browsers).
- Semaphore separate from HTTP workers; `PARSER_LANDER_HEADLESS=true`.
- Extract: visible text + script src for competitor stack.

**DoD:**

- [ ] `playwright` documented in README (install steps, not in go.mod).
- [x] Sidecar or `exec` with timeout; parser does not block whole pool > 30s/page.
- [x] Fallback: L1 `__NEXT_DATA__` still preferred (no headless if L1 ok).
- [x] Test: mock headless response injected via interface (no real browser in CI).
- [x] Memory cap documented (≤ 512MB extra per browser).

**Anti-slop:**

- `[x]` cannot rely on CI running Chromium — interface test required.
- Do not mark lander DoD done if only stub `headless disabled` removed without pool.

**Verification:**

```bash
go test ./internal/sources/lander/... -count=1
# manual: PARSER_LANDER_HEADLESS=true go run ./cmd/parser scan --source=lander
```

---

### 3.4 P2-4 — GitHub discovery expansion

**Goal:** README §C — issues/gists with tracker migration pain (EN only).

| | |
| :--- | :--- |
| **Status** | [ ] |
| **ARPU** | Pro technical |

**Scope:**

- `internal/sources/github/crawler.go` — GitHub REST search (authenticated token).
- Queries: `voluum alternative`, `self-hosted tracker`, `keitaro docker`.
- Emit: `github:issue/{repo}/{number}` with contact from profile email if public.

**DoD:**

- [ ] `GITHUB_TOKEN`, `GITHUB_SEARCH_QUERIES` env.
- [ ] Rate limit handler (403 + `Retry-After`).
- [ ] `httptest` for search response parse.
- [ ] Respect GitHub ToS — no email scraping from private events.

**Anti-slop:**

- Token required — document in `.env.example`; skip test if absent.
- Do not duplicate `bidshard-leads` parser — extend queries only.

**Verification:**

```bash
go test ./internal/sources/github/... -count=1
```

---

### 3.5 P2-5 — Facebook Groups (optional, high risk)

**Goal:** README P2 — US/LATAM groups; **ToS risk**.

| | |
| :--- | :--- |
| **Status** | [ ] |
| **ARPU** | Starter (experimental) |

**Scope:**

- Explicit **opt-in** `PARSER_SOURCE=facebook` + legal warning in README.
- Prefer manual CSV ingest of post exports over automated scraping if ToS blocks.

**DoD:**

- [ ] README § legal + «experimental» banner.
- [ ] If automated: separate cookie/session config, not in default `all`.
- [ ] Geo + keyword pipeline identical to other sources.
- [ ] Product owner sign-off checkbox in this file before `[x]`.

**Anti-slop:**

- Default status stays `[ ]` until legal sign-off line checked.
- Agent must not implement credential storage in repo.

---

### 3.6 P2-6 — Observability: Prometheus metrics

**Goal:** operator visibility — accepted/junk/latency per source, Gemini quota.

| | |
| :--- | :--- |
| **Status** | [x] |
| **ARPU** | Ops |

**Scope:**

- `internal/metrics/prometheus.go` — counters: `parser_leads_accepted_total{source,priority}`, `parser_junk_total{reason}`, `parser_gemini_wait_seconds`.
- HTTP `:9090/metrics` optional (`PARSER_METRICS_ADDR`).

**DoD:**

- [x] Metrics registered without panic on duplicate import.
- [x] Test: scrape handler returns 200 with expected metric names.
- [ ] Document in README (not ARCHITECTURE duplicate).

**Anti-slop:**

- Counters must increment in real code paths — test asserts delta after `Process()`.

**Verification:**

```bash
go test ./internal/metrics/... -count=1
```

---

## 4. Explicit CUT list (never ship)

| Item | Reason |
| :--- | :--- |
| LinkedIn scraping / outreach | README — closed channel |
| VK / RU-only social | Geo policy |
| White AdTech (GAM, Prebid publisher) | Anti-ICP |
| Auto-send outreach from parser | belongs in `bidshard-leads` |
| Auto-merge keywords without human approve | false positive risk |
| Stored Telegram session in git | secrets |

---

## 5. Milestone status summary

| Tier | Open | Closed |
| :--- | ---: | ---: |
| P0 | 0 | 8 |
| P1 | 0 | 8 |
| P2 | 2 | 5 |

**Suggested order:** P1-4 (blacklist) → P1-7 (pilot tags) → P1-3 (locale) → P1-1 (lifecycle) → P1-5 (forums) → P1-2 (adaptive keywords) → P2-1 (CT) → P2-3 (headless).

---

## 6. Verification matrix (release candidate)

Before calling parser «GA for gray pilot»:

```bash
# Baseline
go test ./... -count=1

# Sources smoke (no API keys)
go run ./cmd/parser scan --output=table

# With Mongo
MONGO_URI=mongodb://localhost:27017 go run ./cmd/parser scan

# Telegram dry-run
go run ./cmd/parser telegram --dry-run

# Gemini (optional)
GEMINI_API_KEY=... PARSER_ICP_CLASSIFY=true go run ./cmd/parser scan
```

| Gate | Command | Needs |
| :--- | :--- | :--- |
| Unit | `go test ./...` | — |
| Slop | [.cursor/rules/ai-slop.mdc](.cursor/rules/ai-slop.mdc) checklist | reviewer |
| Mongo integration | `go test ./internal/sink/... -tags=integration` | Docker |
| Discord | `go test ./internal/sources/discord/...` | — |
| Gemini live | manual stub + API key | network |

**Anti-slop:** «GA» checkbox in README only when §6 table run with pasted outputs in PR or release notes.
