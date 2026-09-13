# Crawl egress, Playwright, and bot-risk (operational notes)

Consolidates engineering discussion (2026-09): how **residential HTTP proxy**, **HTTP crawl**, and **Playwright** fit together; what "low fraud / bot risk" means for Cloudflare-heavy targets; what is already in tree vs open gaps.

**Related:** [HEADLESS.md](../HEADLESS.md) (signal reference + Playwright profile), [docs/OPS.md](OPS.md) (proxy modes), [docs/DEPLOY.md](DEPLOY.md) (RAM), [LEADS.md](../LEADS.md) (source strategy), [SHARDING_SESSION.md](../SHARDING_SESSION.md), [docs/MILESTONE.md](MILESTONE.md), [docs/POST_MORTEM.md](POST_MORTEM.md).

---

## 1. Product snapshot ("Развиваем" vs repo)

High-level inventory from roadmap docs and git (not a soak certificate).

### Shipped in tree (code + ops scripts)

| Area | Where |
|------|--------|
| Telegram MTProto sidecar, pain AND-gate, alert cards | `sources/telegram/`, M3/M11 |
| Session cache / shard / lease (P0-P2) | `cursor.py`, `shard.py`, cron; see [SHARDING_SESSION.md](../SHARDING_SESSION.md) (README "design only" may be stale) |
| H1 geo/CIS, H5 vendor_support, H13 TG-first mix | `geo`, `channel_role`, source profiles |
| H10-H12/H14 taxonomy, scoring, CRM cards | `pain_taxonomy`, `lead_card`, dispatcher |
| M1 SERP off hot CRM accept | `PARSER_SOURCE` profiles, registries |
| M8 keyword scoring, Gemini optional / defer | processor, `PARSER_GEMINI_DEFER` |
| tgweb / lander HTTP + optional headless defer | `internal/sources/tgweb`, `lander` |
| Playwright desktop Chrome-like profile | `sources/headless/profile.py`, `interact.py`, `fetch.py` |
| Residential proxy scope for CF-heavy sources | `PARSER_PROXY_LIST`, `proxy_defaults.go` |
| Entity heat, CRM webhook, Telegram CRM bot, export | `internal/crm/`, `crm-bot` |
| Ops LLM toggle + raw export when Gemini off | `crm_settings`, `/llm`, `internal/ops/settings.go` |
| buyer-discover, triage, profile-link discover (local) | `scripts/ops/`, `sources/telegram/profile_links` |
| Discord discover (recent) | `internal/sources/discord` |
| Jobboard employer OSINT | `internal/sources/jobboard` (code + tests) |

### Partial / ops-blocked

| Area | Blocker |
|------|---------|
| M2 realtime soak, M9 manual review gates | Not green in milestones |
| Forum M6 on datacenter VPS | 403 without working residential |
| Reddit M7 / PullPush | Rate limits |
| Jobboard E2E ordering | Post-mortem soak gaps |
| H9 cron vs realtime as primary | Hypothesis open |
| LEADS Phase 0 baseline | Scripts exist; metrics often not recorded |
| Residential proxy product quality | Burned pools look like DC to CF |

### Consciously dropped

| Area | Note |
|------|------|
| LinkedIn as lead source | [POST_MORTEM](POST_MORTEM.md); processor rejects linkedin-only contact |
| ESPX LinkedIn import | Failed experiment |
| SERP -> hot CRM accept | Registry / batch only |
| Shodan -> auto CRM | Intel + manual triage |
| Playwright for LinkedIn / open-web SERP | Out of scope |

### Ops model (CRM)

- Hot path: Telegram pain + CRM notify; SSH + Mongo for inspection.
- `crm-bot`: config check, run, version (no legacy tail/db CLI).
- Heat filter on notify: `CRM_TELEGRAM_LEAD_NOTIFY_HEAT_MIN` (webhook posts not dropped by parser heat min when defer-aware).

---

## 2. Playwright: Chrome-like client (browser layer)

Goal: when HTTP+RSC is insufficient for **tgweb / lander**, Chromium should resemble a **desktop Chrome user** opening a public affiliate URL (sections 2-3, 5-7 in [HEADLESS.md](../HEADLESS.md)).

| Component | Role |
|-----------|------|
| `sources/headless/profile.py` | UA + `Sec-Ch-Ua*` aligned with `internal/httpclient/client.go`; viewport 1920x1080; launch without `--enable-automation`, `AutomationControlled` disabled |
| `sources/headless/interact.py` | Post-`goto` dwell, wheel scroll, curved `mouse.move` |
| `sources/headless/fetch.py` | Context, proxy, navigation, interact, `content()` |
| `internal/sources/lander/headless.go` | Go pool; subprocess per fetch |
| `PARSER_LANDER_HEADLESS_DEFER` + drain cron | Keep browsers off 24/7 hot poll |

Env: see HEADLESS.md section 12 and `.env.example` (`PARSER_HEADLESS_*`).

**Not the same as BidShard tracker antifraud:** parser does not run `antifraud_telemetry.js`; HEADLESS sections 2-7 are the checklist we approximate for **fetch**, not tracker HTTP score.

---

## 3. Residential proxy (network layer)

Residential proxies are **not a separate code path**. They are HTTP(S) URLs in `PARSER_PROXY_LIST` / `PARSER_PROXY_LIST_FILE` whose egress uses ISP/home ASN instead of datacenter VPS IP.

| Topic | Behavior |
|-------|----------|
| Telethon | `TELEGRAM_PROXY_URL` only; **not** `PARSER_PROXY_LIST` |
| Default proxy scope | `forum`, `tgweb`, `lander`, `webpain`, `jobboard`, `serp` (`internal/config/proxy_defaults.go`) |
| Direct by default | `reddit`, `github`, supply seed (unless `PARSER_PROXY_SOURCES` overridden) |
| Transport | Rotating pool, uTLS Chrome ClientHello, browser headers, CF 403/503 cooldown (`internal/httpclient`) |
| Playwright | First comma-separated proxy URL in `sources/headless/fetch.py` (not full Go rotation) |
| Ops pattern | Batch cron (`cf-crawl-cron.sh`, `tgweb-crawl-residential`, `buyer-discover.sh`) to avoid burning residential on every `PARSER_POLL_SEC` |
| VPS Squid | Datacenter egress; **does not** replace residential for Cloudflare igaming ([OPS](OPS.md)) |

**Why it matters:** Cloudflare and many igaming hosts weight **IP reputation** heavily. A perfect User-Agent on a DC VPS still often 403. Residential closes the network gap; it does not fix JS-only challenges by itself.

---

## 4. Two layers together (HTTP + Playwright)

```
                 [ tgweb / lander URL from registry or defer queue ]
                                    |
            +-----------------------+-----------------------+
            |                                               |
       HTTP (primary)                                  Playwright (fallback)
       CrawlClient + uTLS Chrome                      Chromium + profile.py
       ProxyURLsForSource("tgweb"|"lander")           first PARSER_PROXY_LIST URL
            |                                               |
            +-----------------------+-----------------------+
                                    |
                      egress IP (residential if configured)
```

| Signal | HTTP path | Playwright path |
|--------|-----------|-----------------|
| User-Agent, Sec-Ch-Ua, Accept-Language | `applyBrowserHeaders` | `profile.py` + context |
| TLS on the wire | uTLS `HelloChrome_Auto` | Native Chromium (closer to real Chrome, not byte-identical to uTLS) |
| JS / RSC shell | GET + optional RSC merge | Full DOM after JS |
| Post-load behavior | None | `interact.py` when `PARSER_HEADLESS_INTERACTIVE=true` |
| Proxy rotation | Full list per source | First URL only |

**Alignment ops:** use sticky residential session in proxy username; set `PARSER_HEADLESS_TIMEZONE` and `PARSER_HEADLESS_LOCALE` to match proxy country so IP and `Intl` do not contradict.

---

## 5. Cloudflare and "fraud score = 0"

Cloudflare Bot Management does **not** expose a public "fraud score = 0" to this crawler. Externally you observe **pass**, **challenge** (JS / Turnstile), or **403** with `CF-Ray`. In-repo signals:

- `parser_proxy_cf_block_total`
- Proxy cooldown after block responses
- `forum crawl skipped`, tgweb hard-fail logs

### Typical low-risk interactive session (industry model)

| Layer | Clean user |
|-------|------------|
| Network | Residential or home ISP, stable identity, geo consistent with language/TZ |
| TLS / HTTP | Real Chrome stack, current versions |
| Session | Cookies / `cf_clearance` after challenges; return visits |
| JS fingerprint | Real GPU WebGL, plausible `outerWidth`, plugins |
| Behavior | OS-trusted input, varied motion and dwell |
| Rhythm | Irregular timing; not hundreds of cold domains per hour per identity |

### How we compare

| Signal | Clean user | Us (HTTP) | Us (Playwright) |
|--------|------------|-----------|-----------------|
| IP / ASN | Home | OK if real residential in `.env` | Same (first proxy) |
| TLS | Chrome | uTLS Chrome (close) | Chromium native |
| JS challenge | Browser solves | **No JS engine** | Sometimes soft checks only; **no** CAPTCHA/Turnstile solver |
| WebGL | GPU | N/A | Often SwiftShader in Docker |
| Automation traces | Low | N/A | Reduced vs default bot; CDP/headless gaps remain |
| Cookies / identity | Persistent profile | Minimal | **New context per fetch** (no `storage_state` yet) |
| Crawl intent | Browse | Extract contacts, batch | Same |

---

## 6. How much can bot-risk be minimized?

**Realistic goal:** lower 403 rate and raise contact yield per GB proxy; **not** a guaranteed "score zero" on all CF properties.

### Ceiling (cannot fully automate away)

1. Mass registry crawl + contact extract is not organic browsing.
2. Forum-heavy paths stay on **HTTP**; hard CF + Turnstile blocks without browser solve flow.
3. Each Playwright fetch is a **fresh context** (no shared `cf_clearance` across URLs today).
4. Playwright input is not OS-level `isTrusted` (HEADLESS section 6).
5. Burned or datacenter "residential" gateways behave like bad DC IP.

### ROI by lever (qualitative, CF-heavy igaming)

| Lever | Effect | Status |
|-------|--------|--------|
| Residential sticky egress | Largest | Ops: `.env`, `tgweb-crawl-residential`, `cf-crawl-cron` |
| Batch crawl, host limiters, proxy daily cap | High | Cron vs 24/7 poll; `limit.HostLimiters`; config caps |
| IP geo vs locale/timezone | Medium | Env on Playwright; manual per provider |
| Chrome profile + interact | Medium | Implemented in `sources/headless/*` |
| System Chrome / headed + xvfb | Medium for GL | `PARSER_HEADLESS_CHANNEL`, `PARSER_HEADLESS_HEADED` |
| More stealth patches | Low / fragile | Not product direction |
| CAPTCHA / Turnstile solve | Separate product | Out of scope |

### KPIs to use instead of a fictional score

- Ratio HTTP 200 vs 403 on tgweb/forum runs
- `parser_proxy_cf_block_total{source=...}`
- Contacts per domain after residential crawl
- Headless queue depth (`parser_headless_*`, `parser auto status`)

---

## 7. Gaps and plausible next code

| Idea | Why | Status |
|------|-----|--------|
| `storage_state` per sticky proxy session | Reuse cookies and `cf_clearance` on same host | **Done (P0):** `data/runtime/browser_profiles/proxy_N/storage_state.json` |
| Playwright proxy index matches Go HTTP pool | Same persona on wire | **Done (P0):** `LastProxyIndex`, `PARSER_HEADLESS_PROXY_INDEX`, queue `proxy_index` |
### Home ISP bridge (ops)

Reverse SSH + home Squid: [scripts/home-egress/README.md](../scripts/home-egress/README.md). Same `PARSER_PROXY_LIST` path; egress ASN is your ISP, not datacenter VPS.

### P2 (done)

| Item | Implementation |
|------|----------------|
| CF 403/503 + CF-Ray -> headless queue | `page_fetch` `cf_http_block`, `forum/fetch.go` enqueue when `PARSER_LANDER_HEADLESS_DEFER` |
| Daily cap per proxy persona | `PARSER_HEADLESS_MAX_URLS_PER_PROXY_DAY` (default 40), `headless_persona_budget.go` |

### P1 (done)

| Item | Implementation |
|------|----------------|
| Auto locale/timezone from proxy username | `sources/headless/geo_from_proxy.py`, `profile.py`, `geo_env.py`, `headless_proxy_geo.sh` |
| Chrome channel + xvfb for drain | `Dockerfile.playwright` (`playwright install chrome`, xvfb), `headless-crawl-cron.sh`, `headless-xvfb.sh` |

---

## 8. Quick ops checklist

1. `PARSER_PROXY_LIST` residential on VPS for tgweb/forum cron; not required for Telethon discover.
2. `PARSER_LANDER_HEADLESS=false` on 24/7 parser; `PARSER_LANDER_HEADLESS_DEFER=true` + `scripts/ops/headless-crawl-cron.sh`.
3. Match `PARSER_HEADLESS_TIMEZONE` / `PARSER_HEADLESS_LOCALE` to sticky proxy region.
4. Stubborn landers: `PARSER_HEADLESS_CHANNEL=chrome` or headed + `xvfb-run`.
5. Read [HEADLESS.md](../HEADLESS.md) sections 2-7 before changing launch flags or adding "stealth" scripts.
