# LEADS - ICP, sources, geo, segments (BidShard outbound)

Operational reference for lead discovery and qualification. Product ICP rules live in [docs/ICP.md](docs/ICP.md); this doc captures market research and parser/source strategy from 2026-03.

---

## 1. Core thesis

**We sell a tracker/TDS subscription, not vertical access.**

| Care about | Do not care about |
|------------|-------------------|
| Willingness to pay in USDT for SaaS/infra | Vertical (nutra, igaming, crypto, dating) |
| Recurring spend on tracker + cloak + proxy stack | Nationality / Russian language alone |
| Self-hosted or migrating off Voluum/Keitaro/Binom | CPA network as B2B buyer |

**ICP in one line:** performance operator who routes paid traffic through redirect funnels, depends on S2S postback, and pays the gray infra stack in USDT/crypto.

---

## 2. Who needs a tracker (beyond "media buying team")

TDS = cheap geo/device split. Tracker = postback + multi-offer rotation + bot filter + cost sync. UTM is enough only for "ad -> lander -> form" with one offer.

### Tier 1 - high fit + USDT stack

| Segment | Traffic | Why tracker | Typical pain |
|---------|---------|-------------|--------------|
| **Media buying teams** (3+ buyers) | Meta, Google, native, push, TG Ads | Multi-seat, P&L per buyer, postback to network | postback mismatch, Voluum overage, migration |
| **Push / pop / native arbitrage** | RichAds, Propeller, Taboola, MGID | Millions of clicks, redirect speed, flat fee vs events | Binom MySQL/502, manual cost sync, Voluum pricing |
| **Search arbitrage / RSOC** | Tonic, System1, Ads.com + paid traffic | Multi-hop flows, revenue sync to ad platform | No click-level revenue, AFD -> RSOC migration |
| **Solo power buyer** | Same as above, 1 person + VPS | Binom flat license + own server | subid not found, crash at 3am |
| **PWA / WebView / app-install** | KClient PHP, in-app webview | subid through session/install | "Click for subid not found" |
| **Telegram Ads + Mini App funnels** | TG Ads -> bot -> WebView -> FTD | start_param attribution, deposit postback | New stack, few templates |

### Tier 2 - pays, different persona

| Segment | Notes | USDT likelihood |
|---------|-------|-----------------|
| **Performance agency** (multi-client) | RedTrack/Voluum Agency tier, white-label | Medium (EU agencies on cards) |
| **Pay-per-call** | Voluum/RedTrack + Ringba; insurance/Medicare | Lower |
| **Lead gen** (forms, multi-step) | ClickFlare-style funnels | Medium |
| **SEO / parasite / doorway** | Cloaking, rotation, crawler block | High (same gray stack) |
| **Sub-affiliate / mini-network** | Publisher panel, smart links | High |

### Tier 3 - weak fit for BidShard

| Segment | Why skip |
|---------|----------|
| **CPA network entity** | Own tracking (Everflow, Affise); marketplace not tracker license |
| **In-house iGaming operator (product)** | Scaleo, Intelitics, InTarget - operator analytics |
| **DTC e-commerce** | Shopify/RedTrack language, not arbitrage |
| **Course sellers / newbie affiliates** | TDS or UTM enough |
| **Account/card sellers, pump channels** | instant_drop |

**Exception:** in-house **acquisition team** at a gray operator (own casino product) often runs the same Keitaro+cloak stack as affiliates. Find via hiring posts ("in-house media buyer"), not network AM promos.

---

## 3. USDT and payment reality

USDT is not "crypto vertical only" - it is the **default settlement layer** for gray affiliate infra globally.

### What gets paid in USDT (typical stack)

```
USDT / crypto stack
  VPS / hosting (Vultr crypto, Friendhosting, AlexHost, offshore hosts)
  Residential / mobile proxies
  Anti-detect, warmed accounts
  Cloakers (HideClick, Adspect, Cloaking.House)
  Virtual cards for ads + SaaS (Kripicard, Spendge, Spend.net)
  Affiliate network payouts (TRC20 weekly)
  Tracker hosting (often NOT the Keitaro/Binom license itself)
```

Keitaro/Binom **licenses** are mostly card/PayPal. Buyers still fit USDT ICP when the **whole ops stack** is crypto-funded.

### USDT-ready geos (expand beyond CIS/EU)

| Region | Why | Where to look |
|--------|-----|---------------|
| **LATAM** (BR, MX, AR, CO, CL) | Large igaming/performance market, Portuguese/Spanish TG | arbitragem BR, compra de midia, @affiliate_latam |
| **SEA** (VN, ID, PH, TH) | Low CPM, COD, Telegram-heavy crypto/fintech | Involve Asia, MasOffer, komunitas affiliate |
| **India** | English creatives, display/native buyers | @affiliate_marketing_india, @trafficvaultt |
| **Africa** (NG, KE, GH, ZA) | iGaming growth, USDT payouts from networks | @affiliatepartnersafrica, network AM threads |
| **Central Asia** (KZ, UZ) | CIS-language COD, Telegram as ad layer | Almaty teams (CPA.RIP), UZ COD arbitrage |
| **Global EN** | @chat_affiliates, @aff_secret, STM/AffiliateFix |

---

## 4. Geo and compliance (who to cut vs keep)

**Not:** Russian language or ethnicity.  
**Yes:** Legal entity and physical ops in RU/BY without offshore counterparty.

### Three axes

| Axis | Policy |
|------|--------|
| **Language** (Cyrillic, RU text) | Never reject |
| **Legal entity** (contract counterparty) | Reject RU/BY registration (OOO, INN, .ru corp email) |
| **Physical presence** (where ops run) | Reject proven RU/BY ops (Sber, Mir, SBp, Moscow office, +7 as only work contact) |

### Safe harbor (do not reject)

- Cyprus Ltd, Curacao (GCB), Malta, Isle of Man, UAE DMCC
- USDT-only settlement, Wise/Revolut, international corp domain (.com/.io)
- Explicit offshore: Limassol, Amsterdam, Dubai, Tbilisi, Belgrade
- Russian-speaking team + CY/CW entity + USDT = **accept** (verify entity at onboarding)

### Hard reject signals

- INN/OGRN, OOO/AO, RU legal address
- Sber, Tinkoff, Mir, SBP, ruble invoicing
- reg.ru, beget, timeweb as primary infra
- Europe/Moscow timezone + "office in Russia" as registration context

### Gray zone -> `geo_review`, not auto-reject

- Russian text only, no legal entity named
- Personal mail.ru / +7 without corp context
- "Ex-Moscow team, now remote"

### Parser implication

- `GEO_BLOCK_COUNTRIES=RU,BY` stays for **jurisdiction/presence**, not language.
- Fix target: do not reject on `person_country=RU` alone if `company_country` is CY/CW/Malta and USDT/international banking present (entity-first policy).
- KYC/sanctions: onboarding step (KYB, OFAC on company), not parser-automated.

---

## 5. CPA networks - intel, not buyers

| Role | Treatment |
|------|-----------|
| **CPA network as customer** | No - marketplace, different product |
| **CPA network TG channel** | Yes - scrape for teams inside |
| **Affiliate/webmaster in network chat** | Yes - ICP if buyer voice + tracker pain |
| **Network AM promo** | Drop (`supply` role, `RejectAffiliateNetworkSupply`) |

### Channel roles in `config/sources.telegram.yaml`

| Role | Meaning |
|------|---------|
| `buyer_supergroup` | Buyer communities, HR/recruiting |
| `supply` | CPA network official channels - intel only |
| `vendor_support` | Hosting/PWA vendor chats - buyer pain only |

### CPA network seeds (supply)

- `@drcashglobal`, `@drcash`
- `@affiliatepartnersafrica`

Emit policy (`sources/telegram/cpa_network_intel.py`): allow hiring media buyer, tracker pain, buyer questions; drop pure AM/network promos.

### HR / recruiting (team OSINT)

Vacancy posts reveal **employer teams**, not candidates.

- `@affy_hr`, `@mediabuyers_lenkep`, `@recruiting_affiliates`

Use for company name, budget signals, stack mentions - same as Djinni/Hirify OSINT in [docs/ICP.md](docs/ICP.md).

---

## 6. Telegram and Discord sources (in repo)

### Manual TG seeds (`config/sources.telegram.yaml`)

**Global buyer**

- `@chat_affiliates` - large affiliate chat (~36k)
- `@aff_secret` / affiliate marketing magazine
- `@mediabuyingandselling`, `@affiliategroupchat`

**HR / recruiting**

- `@affy_hr`, `@recruiting_affiliates`, `@mediabuyers_lenkep`

**LATAM**

- `@affiliate_latam`, `@affiliate_latam_en`

**India / SEA**

- `@affiliate_marketing_india`, `@trafficvaultt` (India display/native)
- `@komunitasaffiliate` (Indonesia)

**CPA network intel (supply)**

- `@affiliatepartnersafrica`, `@drcashglobal`, `@drcash`

### Discovery queries

- `config/discover.icp.json` - telegram_search, serp_dorks, tg_catalog_dorks (LATAM, SEA, Africa, India, Central Asia, USDT, recruiting, CPA network channels)
- Run: `make buyer-discover` or `bash scripts/ops/buyer-discover.sh`
- Then: `bash scripts/ops/triage-telegram-registry.sh`

### Keyword overlays

- `data/keywords-pt.json`, `keywords-es.json` - LATAM
- `data/keywords-id.json`, `keywords-vi.json` - Indonesia, Vietnam
- `data/keywords-gray.json` - USDT / crypto-gray signals
- Env: `KEYWORDS_LOCALE=es,pt,id,vi`

### Discord seeds

- `config/discord_seed_invites.txt` - voluum, keitaro, binom, afflift, media-buying, igaming, etc.
- Discover: `bash scripts/ops/discord-discover.sh`

### channel_search pain terms (in-channel)

postback failing, voluum pricing, keitaro alternative, usdt payment tracker, hiring media buyer, cod media buying, arbitragem tracker, ...

---

## 7. Pain signals (discovery and scoring)

### Universal (all segments)

- postback failing / timeout / not firing
- subid / click id not found
- stats don't match network / revenue mismatch
- voluum too expensive / event overage / redtrack overage
- tracker migration / leaving voluum
- self-hosted tracker / 502 bad gateway / mysql deadlock

### Segment-specific

| Segment | Extra signals |
|---------|---------------|
| Push/pop | redirect speed, bot traffic, blacklist rotation |
| RSOC | revenue attribution, tonic/system1 integration |
| PWA/WebView | webview stuck, subid session, KClient |
| TG Mini App | start_param, FTD not tracked, telegram ads |
| Pay-per-call | ringba, retreaver, voluum postback |
| USDT buyer | trc20 payout, crypto card, capitalist, weekly usdt |

### Anti-signals (drop)

- Seller spam (accounts, cards, farms)
- CPA network AM outreach without buyer voice
- Job/tutorial noise without tracker context
- Programmatic / SSP / brand AdTech

---

## 8. GTM priority

| Priority | Segment | Why |
|----------|---------|-----|
| **P0** | Media buying teams (3+) | Highest LTV, core ICP |
| **P0** | Push/pop/native shops | Max tracker pain + volume |
| **P1** | Solo power buyer + DevOps | Fast decision, one contact |
| **P1** | TG Ads / Mini App acquisition | Growing, less competition |
| **P1** | RSOC / search arbitrage | Migration wave, pricing pain |
| **P2** | Performance agencies | Longer cycle, multi-seat |
| **P2** | PWA/WebView operators | Niche but acute postback pain |
| **P3** | Pay-per-call | Ringba stack, weaker fit |
| **Skip** | CPA networks, enterprise operators | Wrong buyer |

---

## 9. Where media buyers actually live (not LinkedIn)

### Non-forum surfaces

| Surface | Use |
|---------|-----|
| TG HR channels | @affy_hr, @mediabuyers_lenkep - team OSINT |
| TG buyer supergroups | @chat_affiliates, @aff_secret, regional seeds |
| CPA network channels | Find affiliates and hiring teams (supply role) |
| Affiliate World / SiGMA | LATAM track, conferences |
| Spy/backlink on competitor landers | Operator domains |
| Regional SERP / tgstat | PT, ES, EN-PH, VI, ID dorks in discover.icp.json |
| Job boards | Djinni, AffGate, jobs.dou.ua - **employer** name only |
| Reddit | `r/affiliatemarketing`, `r/media_buying`, `r/PPC` - separate `reddit` source |

### Forum crawl tiers (`internal/sources/forum/hosts.go` + `discover.icp.json` serp_dorks)

| Tier | Host | Parser | GTM marketplace | Notes |
|------|------|--------|-----------------|-------|
| P0 | affiliatefix.com | yes | vendor section | Core tracker/postback pain + t.me |
| P0 | stmforum.com (AW Forum) | yes | paid vendor | High-level media buying |
| P0 | afflift.com | yes | partner section | Push/pop, CPA, tracker |
| P0 | blackhatworld.com | yes | Jr.VIP / marketplace | Media buying, CAPI, teams |
| P0 | gpwa.org | yes | partner sections | iGaming affiliates |
| P0 | forobeta.com | yes | Negocios | LATAM (ES) |
| P1 | wickedfire.com | yes | careful | CPA/media buying; low volume |
| P1 | webmasterworld.com | yes | no in general threads | PPC veterans, tracker pain |
| P1 | topgold.forum | yes | Buy & Sell | Traffic/crypto overlap |
| P1 | fb-killa.pro | yes | partner section | CIS FB/gambling; **geo gate on accept** |
| P1 | gfbforum.live | yes | loyal mods | CIS arbitrage teams |
| P1 | cpamafia.pro | yes | services branch | CPA/media buying tools |
| P1 | addset.ru | yes | admin approval | FB/Google arbitrage |
| P1 | offshorecorptalk.com | intel | marketplace | Entity OSINT (owner/payments), not volume leads |
| wired | cpaelites.com, digitalpoint.com, wjunction.com, iamaffiliate.com | yes | varies | Long-tail; lower priority than P0 |
| skip | hackforums.net, blackhatprotools.info | no | - | Fraud/script crowd, wrong ICP |
| skip | beermoneyforum.com, warriorforum.com | low | strict | Micro-earners / declining quality |
| GTM only | producthunt.com, betalist.com, pitchwall.co | no | launch | SaaS showcase, not forum crawl |

---

## 10. Pre-sale checklist (USDT subscription)

Before first invoice:

1. Invoice entity is not RU/BY (registry or site + LinkedIn company)
2. No INN/OOO/RU bank in comms
3. USDT wallet / payment path agreed
4. OFAC screen on company name (manual or provider)
5. If `geo_review`: ask "Which legal entity for the invoice?" - one question clears most gray cases

---

## 11. Repo wiring (quick ref)

| File | Purpose |
|------|---------|
| [docs/ICP.md](docs/ICP.md) | Canonical yes/no ICP |
| [config/discover.icp.json](config/discover.icp.json) | SERP + Telethon discovery queries |
| [internal/sources/forum/hosts.go](internal/sources/forum/hosts.go) | Forum crawl host allowlist (P0/P1 tiers in section 9) |
| [config/sources.telegram.yaml](config/sources.telegram.yaml) | Pool, manual seeds, channel_search |
| [config/discord_seed_invites.txt](config/discord_seed_invites.txt) | Discord guild seeds |
| [data/keywords.json](data/keywords.json) + gray + locale overlays | Message scoring |
| [sources/telegram/cpa_network_intel.py](sources/telegram/cpa_network_intel.py) | CPA channel emit policy |
| [sources/telegram/geo_heuristic.py](sources/telegram/geo_heuristic.py) | RU/BY infra hard-stop at ingress |
| [internal/gemini/geo.go](internal/gemini/geo.go) | Warm-path geo classify |
| [internal/filter/supply_profile.go](internal/filter/supply_profile.go) | Network AM promo drop |

---

## 12. Expansion roadmap (ordered tasks)

Execution model: **DISCOVER -> TRIAGE -> SCRAPE -> SCORE -> QUALIFY -> CRM**. One phase at a time; measure before adding the next batch of dorks.

Priority key: **P0** = do first / blocks scale, **P1** = high ROI this month, **P2** = next wave, **P3** = optional.

---

### Phase 0 - Baseline (week 1)

| # | Pri | Task | Deliverable | Owner |
|---|-----|------|-------------|-------|
| 0.1 | P0 | Deploy current config to VPS (discover, seeds, staggered cron, locale overlays) | `KEYWORDS_LOCALE=es,pt,id,vi` in `.env`; cron :07/:37 stable | ops |
| 0.2 | P0 | Run full discover + triage once | `buyer-discover.sh` + `triage-telegram-registry.sh`; registry snapshot saved | ops |
| 0.3 | P0 | Record baseline metrics (3-5 scrape cycles) | Counts: registry size, pool active, TG emitted, accepted, top reject reasons | ops |
| 0.4 | P1 | Document baseline in this file (one line per metric) | Subsection under Phase 0 with date + numbers | ops |

**Baseline snapshot (2026-09-12T20:37Z, deploy + `buyer-discover-fast.sh` + 1 TG cron shard):**

| Metric | Value | Notes |
|--------|-------|-------|
| `registry_channels` | 318 | After triage (340 raw, 22 dropped) |
| `pool_enabled` / `pool_disabled` | 147 / 0 | In `parser_runtime` volume (`crawler.db`) |
| `export_lines` | 262 | JSONL export total |
| `export_telegram_lines` | 77 | TG-sourced lines in export |
| `mongo_leads_total` | 0 | Mongo count query returned 0 (verify URI/collection) |
| `mongo_telegram_today` | 0 | Same |
| `keywords_locale` | es,pt,id,vi | P0 env applied |
| cron | :07 / :37 UTC | 2 hot shards + cold discover 03:15 |
| scrape shard 0 | raw=2131 accepted=30 | One manual cron run |
| scrape shard 1 | failed | `telethon.session.1` not authorized on VPS |
| prometheus | unavailable | `:9465` not reachable from collect script |

**Do not** run full `parser discover serp` for Phase 0: 151 DuckDuckGo dorks at ~1-2 min each on `http 202`/EOF plus employer reverse = hours. Use `bash scripts/ops/buyer-discover-fast.sh` (triage only) when registry already exists in `parser_runtime`.

Re-run `--collect-only` after shard 1 session fix + 2-3 more cron cycles.

---

### Phase 1 - GEO breadth (week 1-2)

| # | Pri | Task | Deliverable | Owner |
|---|-----|------|-------------|-------|
| 1.1 | P0 | Weekly registry review ritual (30 min) | 5-10 new handles -> `chats[]`; junk -> `denylist` | ops |
| 1.2 | P0 | Promote cross-mention finds from buyer chats | New channels in registry with `query=cross_mention` triaged | ops |
| 1.3 | P1 | Add 5-10 verified regional seeds per geo gap | LATAM / IN / Africa / KZ-UZ handles in `sources.telegram.yaml` | ops + OSINT |
| 1.4 | P1 | Add CPA network channels as `supply` only | New handles in yaml + `cpa_network_intel` hints if needed | dev |
| 1.5 | P1 | Batch discover dorks (+10-15 per region, not 50 at once) | Edits to `discover.icp.json`; re-run discover; compare registry delta | dev |
| 1.6 | P2 | tg_catalog_dorks for tgstat/telemetr per region | PT, ES, EN-PH, VI, ID catalog pages | dev |
| 1.7 | P2 | Rotate `KEYWORDS_LOCALE` weekly if needed | e.g. `es,pt` then `id,vi` on alternate weeks | ops |

**Do not:** weaken `geo_heuristic` for LATAM/SEA - it only hard-stops RU/BY infra.

---

### Phase 2 - Segment depth (week 2-4, one segment per week)

| # | Pri | Task | Deliverable | Owner |
|---|-----|------|-------------|-------|
| 2.1 | P0 | Discover pack: **push / pop / native** | `discover.icp.json`: push traffic binom, popunder postback, propellerads tracker, etc. (~15 queries) | dev |
| 2.2 | P0 | Keywords + channel_search: push/pop | `keywords.json` or gray: redirect speed, manual cost sync, event overage; yaml `channel_search` terms | dev |
| 2.3 | P1 | Discover pack: **RSOC / search arbitrage** | RSOC tracking, tonic voluum, search arbitrage postback (~15 queries) | dev |
| 2.4 | P1 | Keywords: RSOC | revenue attribution, click level revenue, AFD migration pain | dev |
| 2.5 | P1 | Discover pack: **PWA / WebView** | webview postback keitaro, KClient subid (extend `pwa_dorks`) | dev |
| 2.6 | P1 | Discover pack: **Telegram Ads / Mini App** | telegram ads tracker, mini app FTD postback, start_param (~10 queries) | dev |
| 2.7 | P1 | Keywords + channel_search: PWA + TG Mini App | webview subid, start_param, FTD not tracked | dev |
| 2.8 | P2 | HR / hiring scoring boost | Vacancy with media buyer + budget + tracker stack: do not `neg-we-are-hiring`; entity OSINT path | dev |
| 2.9 | P3 | Discover pack: pay-per-call (Ringba) | Lower priority; forum dorks only | dev |
| 2.10 | P3 | Separate scoring tier solo vs team (optional) | Team size signal in score or CRM tag | dev |

After each segment pack: run discover, wait one week, check **accepted rate** vs baseline before next segment.

---

### Phase 3 - Qualification (week 3-4, parallel with Phase 2)

| # | Pri | Task | Deliverable | Owner |
|---|-----|------|-------------|-------|
| 3.1 | P0 | **Entity-first geo** in `ShouldReject` | Reject on `company_country` RU/BY; person RU only without safe harbor | dev |
| 3.2 | P0 | Safe harbor list (CY, CW, MT, AE, NL, ...) | Code + tests: RU text + CY Ltd + USDT -> accept | dev |
| 3.3 | P0 | Gemini geo prompt: language != jurisdiction | Update `geoSystemPrompt` in `internal/gemini/geo.go` | dev |
| 3.4 | P1 | CRM tag **`geo_review`** for gray leads | Not auto-reject; inbox filter for human question on legal entity | dev |
| 3.5 | P1 | Expand `supply` role auto-detect | More CPA network username/title hints in `cpa_network_intel.py` | dev |
| 3.6 | P2 | Pre-sale checklist wired in CRM notes template | Section 10 fields on accepted high-score leads | ops |

---

### Phase 4 - Sources beyond Telegram (week 4-6)

| # | Pri | Task | Deliverable | Owner |
|---|-----|------|-------------|-------|
| 4.1 | P1 | Forum discover: push/RSOC pain threads | `serp_dorks` on affiliatefix/stm for segment-specific pain | dev |
| 4.2 | P1 | GitHub: keitaro/binom postback issues | Existing github harvest; add queries if rotate script supports | dev |
| 4.3 | P2 | Reddit: r/affiliatemarketing, r/media_buying | serp_dorks already partial; expand + monitor accept rate | dev |
| 4.4 | P2 | Discord discover cron on VPS | `discord-discover.sh` + review `discovered_discord_invites.json` | ops |
| 4.5 | P2 | Jobboard discover -> employer OSINT only | djinni/affgate queries; reject candidate, capture company name | dev |
| 4.6 | P3 | tgweb domain registry for tracker fingerprint | intel path; not 24/7 unless volume justifies | ops |

---

### Phase 5 - Outbound feedback loop (ongoing, weekly)

| # | Pri | Task | Deliverable | Owner |
|---|-----|------|-------------|-------|
| 5.1 | P0 | Manual review 10 accepted leads / week | Note: segment, geo, query, USDT signal yes/no | sales + ops |
| 5.2 | P0 | Manual review 20 rejected high-score / week | False reject? -> fix qualify; true junk? -> denylist/keyword | ops |
| 5.3 | P1 | Update this doc: what worked / what failed | 2-3 bullets under "Feedback log" below | ops |
| 5.4 | P1 | Promote winning queries to `chats[]` seeds | Proven channels get guaranteed scrape slots | ops |
| 5.5 | P2 | Demote sources with 0 accept over 30 days | `enabled: false` or denylist | ops |

---

### Master order (do in sequence)

1. **0.1** - 0.4 (baseline)
2. **1.1** - 1.2 (weekly ritual + cross-mention)
3. **3.1** - 3.3 (entity-first geo - do before scaling GEO discover)
4. **1.3** - 1.5 (regional seeds + CPA supply + dork batches)
5. **2.1** - 2.2 (push/pop segment)
6. **2.3** - 2.4 (RSOC segment)
7. **2.5** - 2.7 (PWA + TG Mini App)
8. **3.4** - 3.5 (geo_review + supply expand)
9. **4.1** - 4.2 (forum + github)
10. **5.1** - 5.3 (feedback loop starts week 2, never stops)
11. **1.6** - 1.7, **2.8** - 2.10, **4.3** - 4.6, **3.6**, **5.4** - 5.5 (P2/P3 as capacity allows)

---

### Feedback log

| Date | Accepted wins | Reject fixes | Next discover batch |
|------|---------------|--------------|---------------------|
| 2026-03-12 | (baseline not run yet) | - | GEO seeds deployed in repo; VPS verify pending |
| 2026-09-12 | Triage 318 channels; shard0 scrape 30 accepted | `triage` host/volume split fixed; skip full `discover serp` | Fix `telethon.session.1` auth; use `buyer-discover-fast.sh` |
| 2026-09-12 | 3-session pool live: 4 hot scrapes/h + cold triage 03:15 | Killed stuck history-export; `cron_shared_session=false` | All pool slots same TG user; add 2nd/3rd accounts later for real multi-account |

---

### Anti-patterns (do not)

- Add 50+ dorks in one PR without measuring registry quality
- Put CPA network channels in `buyer_supergroup` (use `supply`)
- Treat Keitaro/Voluum vendor accounts as ICP buyers
- Reject Cyrillic or Russian language without RU/BY legal/presence signals
- Scale discover before entity-first geo (cuts valid offshore teams)

---

## 13. Hot + cold outreach (teams and people)

Two parallel outcomes; do not score them on one scale.

| Funnel | Signal | Contact | CRM use |
|--------|--------|---------|---------|
| **Hot** | Tracker/postback/USDT pain, displacement | Message **sender** `@username` + optional bio | `telegram_dm`, pilot tags, high `engage_priority` |
| **Cold** | Hiring, team scaling, in-house buyer stack | **Company/team** from vacancy + HR channel OSINT | Entity inbox, `research_contact` / DM to poster if human |
| **Intel** | CPA network, vendor channel | Channel admin / cross-mentions | `supply` role; not invoice buyer |

### What the repo does today

| Capability | Status | Where |
|------------|--------|-------|
| Sender `@username` + `user_id` on emitted messages | yes | `sources/telegram/scraper.py` -> `sender_user_id`, `contact` |
| Bio via GetFullUser (cached) | yes, **10/run default** | `user_enrich.py`, `TELEGRAM_USER_ENRICH_LIMIT` |
| Bio folded into score/Gemini text | yes | `processor.go` appends `user_bio:` |
| Linked channel **discussion** scrape (comments) | optional | `TELEGRAM_DISCUSSION_SCRAPE`, `discussion.py` |
| History export (backfill senders in supergroups) | yes, manual/cron | `history_export.py` |
| **Chat participant list** (all members) | **yes** (scrape) | `iter_participants` in `participants.py`; sqlite `telegram_chat_members`; export `data/runtime/telegram_chat_members.json` |
| Hiring as **team OSINT** (not hard-reject on TG) | partial | UA/RU vacancies often pass; `context_drop` drops forum job posts |
| Entity merge on telegram contacts | yes | `internal/entity/` |
| Person-level "cold outreach OK?" classify | partial | `cold_outreach_fit` heuristic on GetFullUser profile; Gemini entity path optional |
| Global search (SearchGlobal) | yes (default on) | `global_search.py`, env `TELEGRAM_GLOBAL_SEARCH=1` |
| Auto-join invite chats | yes (default on) | `join_policy.py`, `TELEGRAM_INVITE_JOIN` + `TELEGRAM_INVITE_JOIN_HOT` |
| Message forwards / views / reactions / poll | yes (NDJSON) | `message_meta.py` -> `message_meta` on emit |
| Channel meta (counts, admins, pinned) | yes (JSON export) | `channel_meta.py` |
| Full user profile (not only bio) | yes | `user_enrich.py` -> `sender_profile` + `telegram_user_profiles.json` |
| Cross-mention on scrape | yes | `scrape_crossmention.py` |

### Gap (requested)

1. **Participant harvest** - enumerate active members in buyer supergroups; store `telegram:user_id` + `@username` in entity/contacts table.
2. **Profile pass** - GetFullUser for each candidate (rate-limited); rules + optional Gemini: media buyer / team lead / AM / newbie / vendor.
3. **Dual export** - CRM views: `lead_type=hot_pain` vs `lead_type=cold_team` vs `contact_type=person`.

### Near-term ops (no new code)

- Raise `TELEGRAM_USER_ENRICH_LIMIT` (e.g. 40) on VPS P0 env.
- Enable `TELEGRAM_DISCUSSION_SCRAPE=true` on high-signal supergroups only.
- Prioritize scrape slots: `mediabuyers_lenkep`, `arbitrajvaka255`, `aff_search`, buyer supergroups (not `keitaro_tracker` / `drcash*`).
- History-export backfill on top 5 supergroups (cold session `.2`, not hot).
- CRM sort: `engage_priority` + filter tags `pilot-nurture` (team) vs displacement/pain tags.

### Build order (dev)

1. **P0** - Separate scoring tier for `job_offer` / hiring (entity = company; contact = poster username when not channel broadcast).
2. **P1** - `participant_harvest` job (Telethon) + sqlite registry `telegram_people`.
3. **P2** - Batch profile enrich + `cold_outreach_fit` enum (yes / maybe / no) on entity doc.
4. **P3** - CRM export columns: `person_username`, `profile_bio`, `outreach_fit`, `team_entity_id`.

---

*Last updated: 2026-09-13. Sync parser changes with this doc when adding regions, roles, or completing roadmap tasks.*
