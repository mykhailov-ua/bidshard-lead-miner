# Parser — лидогенерация для BidShard (серый / чёрный рынок)

Парсер ищет **горячих покупателей** self-hosted трекера [BidShard](../bidshard) в сегментах арбитража, iGaming, CPA, nutra/sweeps, pop/push.  
Результаты сохраняются в **MongoDB** (dedup по контактам) и опционально экспортируются в **JSON** (NDJSON).

> **Scope:** серый/чёрный рынок, **весь мир кроме России и Беларуси**. White AdTech (enterprise SSP/DSP, GAM, brand safety) — вне зоны.  
> **LinkedIn — вне зоны полностью** (99% white enterprise или не по теме; в `bidshard-leads` есть legacy-парсинг профилей через Reddit — **не развивать, не использовать**).

---

## Гео: критично

**Целевой рынок:** LATAM, EU, UK, US, CA, AU, SEA, Africa, Middle East, Украина и др. — везде, где есть affiliate/iGaming, **кроме РФ и РБ**.

| Правило | Реализация |
|---------|------------|
| Исключить домены | `.ru`, `.рф`, `.by`, `.бел` в email и site |
| Исключить телефоны | `+7` (RU/KZ — осторожно), `+375` (BY) |
| Исключить источники | VK, RU-only Telegram-чаты, форумы вроде CPAmafia (RU-аудитория) |
| Язык | EN приоритет; ES/PT/PL/DE/FR — ок. Длинный кириллический контекст без EN-сигналов → **reject** или score −20 |
| Timezone / bio | `Europe/Moscow`, `Minsk`, `Russia`, `Belarus` в профиле → reject |
| WHOIS / IP | Страна регистрации домена RU/BY → не питчить |

Фильтр ставится **до** scoring: `geo.Filter(raw) → reject | pass`. Лучше потерять пограничный лид, чем тратить outreach на нецелевое гео.

---

## Что такое BidShard и зачем им это

BidShard — self-hosted трекер, инжест событий, RTB. Для серого рынка критичны:

| Фича | Зачем |
|------|-------|
| `GET /click` — 302 за &lt;80 ms | Меньше слива; gclid/ttclid/sub-ID на ленд |
| Atomic budget / zero afterburn | Нет перерасхода после паузы |
| Safe page (cloak companion) | IVT → white page; чистый трафик → money page |
| S2S postbacks + idempotency | FTD/CPA не теряются |
| Self-hosted, flat fee | SaaS не видит углы; нет счёта за event |
| Bot / fraud filter | Меньше ботового слива |

Конкуренты для миграции: **Keitaro, Voluum, Binom, RedTrack, BeMob, PeerClick, ThriveTracker, ClickFlare**.

В web BidShard уже заложены **36 affiliate postback presets** (MaxBounty, ClickDealer, Mobidea, CrakRevenue, Everflow, …) — это язык боли покупателя, не источник лидов, но полезен для matching в scoring (`bidshard/web/src/models/affiliate_postback_presets.ts`).

---

## Целевая аудитория (ЦА)

GTM-воронка: **пилот 10 дней, $0** (`sku: pilot`) → платный тариф USDT/мес, без setup fee. Парсер и питчи ведут на **конкретный тариф + измеримую боль**, не на абстрактный «трекер».

Источник тарифов: `deploy/vendor/SALES_KIT.md`, `deploy/vendor/sku.yaml`, license UI в web (`Settings → License`).

### Карта тарифов

| SKU | USDT/мес | + VPS типично | **All-in** | RPS | Фокус продаж |
|-----|---------:|--------------:|-----------:|----:|--------------|
| **pilot** | $0 / 10 дн | $40–60 | ~$50 | 50k | Квалифицированный лид; RTB **включён** (в отличие от Starter) |
| **starter** | $129 | $40–60 | **$169–189** | 10k | **Основной таргет** |
| **pro** | $329 | $40–80 | **$369–409** | 25k | Команды с OpenRTB / programmatic |
| **scale** | $649 | $80–120 | **$729–769** | 75k | После первых Starter/Pro — upsell |
| **network** | $1,199+ | multi-host | $1,300+ | 150k | Inbound / реферал |
| **enterprise** | $2,500+ | custom | — | custom | Inbound, custom SLA |

**Ценовой якорь:** Starter **дороже** связки Keitaro+VPS ($75–155) — фильтр price shoppers. Продаём **болью**: счёт Voluum, afterburn, потеря FTD, приватность self-hosted, safe page.

Конкурентные якоря (SALES_KIT): Keitaro €40–70, Binom $149, RedTrack $89–199, Voluum $199–999+, in-house OpenRTB $800–1,500.

---

### ICP #1 — Starter ($169–189 all-in) — **80% усилий парсера**

Кто платит $129+ когда Keitaro стоит €40: не новичок на первом ленде, а команда с **доказанным spend** и конкретной причиной сменить трекер.

| Поле | Профиль |
|------|---------|
| **Роли** | Media buyer lead, affiliate team lead, solo buyer с $15k+/мес spend, ops/devops кто админит Keitaro/Binom |
| **Вертикали** | Nutra, sweeps, dating, crypto, push/pop, FB/TikTok gray, мелкий iGaming web |
| **Spend на трафик** | **$15k–150k/мес** (ниже — не потянут overpay vs Keitaro; выше без техлида — риск упереться в 10k RPS) |
| **Текущий стек** | Voluum, RedTrack, ClickFlare, BeMob, **Keitaro на 2+ VPS**, Binom — ищут замену, не первый трекер |
| **Триггеры (парсить)** | `voluum alternative`, `tracker too expensive`, `per-event pricing`, `postback failing`, `missing ftd`, `budget overburn`, `afterburn`, `self-hosted tracker`, `safe page`, `cloak` |
| **Фичи из web, которые продаём** | `GET /click` + macros, S2S postback (36 network presets), Cost Sync, Margin Guard, Meta CAPI (Starter = Meta only), safe page companion |
| **Не обещать** | OpenRTB live (заблокирован на Starter), ClickHouse-аналитику на старте (ingest-only stack ≈ 6–8 GB RAM) |
| **Пилот 10 дн** | Redirect latency, postback idempotency, budget stop — на реальном объёме |
| **Конверсия pilot→Starter** | День 7–8: USDT invoice $129; экономия vs счёт Voluum/RedTrack |

**Сигнал «горячий Starter» в тексте лида:** жалоба на **счёт SaaS** + упоминание конкурента + живой контакт + spend-язык («$Xk/mo on traffic», «our buyers», «we run 30 campaigns»).

**Отсечь:** «какой трекер выбрать новичку», «keitaro vs binom для старта», spend &lt;$5k/мес, нет VPS/не готовы в `install.sh`.

---

### ICP #2 — Pro ($369–409 all-in) — **15% усилий, выше чек**

Кому нужен **OpenRTB `/openrtb/bid`** (live на Pro; на Starter заблокирован). В web: RTB Integration page, campaign tracking section с OpenRTB endpoint.

| Поле | Профиль |
|------|---------|
| **Роли** | CTO / platform engineer affiliate network, head of programmatic, ad network founder |
| **Кто** | Малая push/native сеть, CPA network с in-house bidder, iGaming affiliate platform, команда которая **уже** платит за ingress |
| **Spend / infra** | $50k+/мес media или **сравнивают с $800–1,500/mo** за свой OpenRTB ingress |
| **RPS** | 10–25k sustained (влезает в Pro; пилот даёт 50k RPS — хорош для smoke test) |
| **Триггеры** | `openrtb`, `rtb exchange`, `bid request`, `dsp bidder`, `prebid server`, «building our own bidder», postback + **programmatic** в одном треде |
| **Пилот** | Shadow→live RTB, validate-bid smoke из web onboarding; **не** «потестить трекер для FB» |
| **Конверсия** | Pilot показал RTB + tracking в одном стеке → Pro $329 vs отдельный ingress |

**Сигнал «горячий Pro»:** dev language + OpenRTB + существующий tracker pain; GitHub issues, STM technical threads.

**Не питчить Pro:** обычный FB media buyer без programmatic; ожидают Pro-фичи по цене Starter.

---

### ICP #3 — Scale ($729+) — inbound / upsell

3 hosts, 75k RPS, `ivt_ml` / fraud-scorer. Покупатель сравнивает с **инфра-бюджетом**, не с Keitaro.

| Поле | Профиль |
|------|---------|
| **Кто** | iGaming operator acquisition, крупная CPA сеть, multi-geo arbitrage shop 5+ баеров |
| **Spend** | $200k+/мес, уже **3+ сервера** под трекинг |
| **Триггеры** | `high qps`, `sharding`, `fraud detection`, `ivt`, multi-region |
| **Вход** | Upsell с Pro, реферал, тёплый inbound с форумов |

---

### Anti-ЦА — не тратить пилот и outreach

| Сегмент | Почему |
|---------|--------|
| **Keitaro entry buyer** (€40, первый трекер, &lt;$10k spend) | Starter на $40+ дороже без боли; сожрут support, не купят |
| **Price-only сравнение** | «cheapest tracker», «free alternative» |
| **Network / Enterprise cold** | Длинный цикл; приоритет — Starter/Pro |
| **Нет live traffic на пилот** | 10 дней — нужен реальный proof |
| **Нет VPS / не поставят за 48h** | hard bind fingerprint; без install нет proof |
| **White AdTech** | GAM, Prebid publisher, brand DSP |
| **LinkedIn enterprise** | закрыт |
| **РФ / РБ** | гео-фильтр |
| HR, курсы, «how to start», job seekers | negative keywords |

---

### Квалификация лида на пилот (чеклист перед outreach)

Пилот — **10 дней на их VPS**: install assist ≤2h, scope — latency, postbacks, budget stop на live traffic. Конверсия на день 7–8.

| # | Вопрос / сигнал | Pass | Fail |
|---|-----------------|------|------|
| 1 | Есть **живой трафик** на этой неделе? | да | «планируем через месяц» |
| 2 | Могут поднять VPS (Hetzner/OVH/DO) за 1–2 дня? | да | «нужен managed only» без бюджета |
| 3 | Названа **измеримая боль**? | postback/afterburn/bill/latency | «просто посмотреть» |
| 4 | Текущий трекер = **Voluum / RedTrack / Keitaro scale / Binom**? | да | только бесплатные / Excel |
| 5 | Spend **≥$15k/mo** или счёт SaaS **≥$150/mo**? | да | ниже порога |
| 6 | Гео **не РФ/РБ**? | да | reject |
| 7 | Готовы к **USDT** после пилота? | да / «card ok later» | «только invoice NET-60 enterprise» |
| 8 | Один decision maker в Telegram/email? | да | committee без контакта |

В CRM: тег `pilot-qualified` → питч с CTA «10-day pilot, install assist»; иначе nurture.

---

### Матрица: боль → тариф → proof в пилоте

| Боль в лиде | Тариф | Proof за 10 дн | Аргумент в питче |
|-------------|-------|----------------|------------------|
| Voluum / RedTrack **счёт растёт с объёмом** | Starter | Cost Sync + flat $129 | Таблица TCO из SALES_KIT |
| **Afterburn** / overspend после паузы | Starter | Atomic budget на их кампании | Redis CAS, мгновенный stop |
| **Postback / FTD** теряются | Starter | S2S + idempotency | 36 network presets из web |
| **Cloak / safe page** | Starter | Safe page redirect на IVT | Campaign safe page UI |
| **SaaS видит углы** | Starter | Self-hosted на их VPS | Data ownership |
| Нужен **OpenRTB**, не только redirect | Pro | Pilot: RTB shadow→live | $329 vs $800+ ingress |
| **Fraud / IVT** на объёме | Scale | После Pro | `ivt_ml` на Scale SKU |

---

### Приоритет парсера по ARPU

Ориентир SALES_KIT: **3 × $129 MRR floor** → **8 × ~$200 base**.

| Приоритет | Сегмент | Ожид. тариф | Parser weight |
|-----------|---------|-------------|---------------|
| **P0** | Voluum / RedTrack / ClickFlare refugees | Starter | +25 intent keywords |
| **P0** | Keitaro/**Binom power users** (не entry) | Starter | competitor + pain stack |
| **P1** | iGaming web, FTD pain, EU/LATAM | Starter → Pro | vertical + ftd keywords |
| **P1** | Small network / OpenRTB curious | Pro (pilot first) | tech + openrtb |
| **P2** | Solo nutra/sweeps &lt;$15k spend | Starter (pilot-qualified) | ниже score threshold |
| **P3** | Scale / Network / Enterprise inbound | Scale+ | inbound only |

Персоны для шаблонов питча: `media-buyer` (Starter), `cto` (Pro), `affiliate` (оба), `ad-ops` — `bidshard-leads/internal/bot/persona.go`.

---

## Полный каталог способов поиска клиентов

Ниже — **все** каналы: что уже крутится в `bidshard-leads`, что делает этот репо, и что можно добавить. LinkedIn везде вычёркнут.

### A. Социальные сигналы и форумы (intent = жалоба на трекер)

| # | Канал | Статус | Как искать | Гео |
|---|-------|--------|------------|-----|
| 1 | **Telegram** (чаты/каналы) | **P0, parser** | MTProto + Telethon, см. ниже | EN/LATAM/EU чаты, не RU |
| 2 | **STM Forum** | P0 | HTML/RSS, разделы Tracking, iGaming, Facebook | Global EN |
| 3 | **AffiliateFix** | P0 | Subforums: tracking, iGaming, nutra | Global |
| 4 | **BlackHatWorld** | P0 | Tracking, CPA, FB/IG ads, cloaking | Global EN |
| 5 | **Reddit** | частично в leads | r/affiliatemarketing, r/media_buying, r/gambling, r/poker, r/juststart — PullPush API | Global, geo по контакту |
| 6 | **Discord** | P1 | Серверы STM/Affiliate/iGaming; bot или user token | Global |
| 7 | **Warrior Forum / CPALead** | P1 | Nutra/sweeps/CPA треды | US/EU |
| 8 | **Facebook Groups** | P2 | «Affiliate marketing», «Media buying» (осторожно с ToS) | US/LATAM |

**Уже в bidshard-leads (не дублировать без расширения):** Reddit (27 сабов), GitHub users/issues, HN Algolia, Lobsters.  
**Legacy, отключить:** `parseLinkedIn()` — LinkedIn URL из Reddit-постов.

### B. Техническая разведка по доменам (proactive)

| # | Метод | Что даёт | Как |
|---|-------|----------|-----|
| 9 | **ads.txt crawl** | Publisher/ad network с programmatic стеком; DIRECT-строки с `contact@` в sellers.json | `GET https://{domain}/ads.txt` — см. раздел ниже |
| 10 | **sellers.json crawl** | `contact_email`, seller domains, имена сетей | `GET https://{domain}/sellers.json` |
| 11 | **CNAME / subdomain fingerprint** | `track.domain.com` → `*.voluum.com`, `*.keitaro.io` | DNS lookup + crt.sh |
| 12 | **Certificate Transparency** | Новые поддомены `track.`, `click.`, `go.` | crt.sh, Facebook CT |
| 13 | **Landing page tech detect** | Пиксели Voluum/Binom/Keitaro в HTML/JS | HTTP GET + regex; для Next.js — hydration |
| 14 | **Postback URL patterns** | На лендах и в view-source: `postback?`, `clickid=`, сети из presets BidShard | curl + parse |
| 15 | **app-ads.txt** | Mobile affiliate / iGaming apps | `GET /app-ads.txt` |
| 16 | **WHOIS / RDAP** | Registrant email (фильтр RU/BY) | whois API |

Seed-листы доменов: топ casino affiliates (non-RU), push networks, nutra landers из ad spy (Anstrex, AdPlexity — ручной экспорт), списки из STM «follow along».

В BidShard **ads.txt/sellers.json — фича для покупателя** (Integrations → Supply), не встроенный lead crawler. Парсер использует **чужие** supply-файлы как источник контактов и сигнала «у них есть ad stack».

### C. GitHub / dev-следы

| # | Метод | Запросы |
|---|-------|---------|
| 17 | User search | `keitaro`, `voluum`, `binom`, `igaming tracking` in:bio |
| 18 | Issues / discussions | «self-hosted tracker», «postback latency», «migrate from voluum» |
| 19 | Gists / docker-compose | `keitaro`, `binom`, exposed `.env` с contact |

Уже в `bidshard-leads/internal/parser/parser.go` — расширять EN-запросы, не RU.

### D. Отзывы и сравнения (pain harvesting)

| # | Источник | Сигнал |
|---|----------|--------|
| 20 | **Trustpilot / G2 / Capterra** | 1–2★ отзывы на Voluum, Keitaro, RedTrack |
| 21 | **STM/AffiliateFix «X alternative»** | Треды «voluum alternative 2026» |
| 22 | **Product Hunt / IndieHackers** | Реже, но self-hosted / cost threads |

### E. Outreach-исполнение (не discovery, но воронка)

Уже в bidshard-leads:

| Канал | Файлы |
|-------|-------|
| Email SMTP / Gmail OAuth | `internal/pitch/smtp.go`, `gmail.go` |
| Telegram CRM (лента, питчи, snooze) | `internal/bot/*` |
| Calendly в шаблонах | `CALENDLY_URL`, `{{calendly}}` |
| LLM summary / draft pitch | `internal/llm/` |
| Дайджест 09:00, SLA pitch/reply | `internal/bot/scheduler.go` |
| Pilot JWT **10 дней** | `bidshard/deploy/vendor/SALES_KIT.md`, `sku.yaml` (`valid_days: 14` в yaml — выставлять `--days 10` при issue) |

**Не использовать:** LinkedIn copy-paste (`pitch_flow.go` linkedin channel) — канал закрыт для outreach.

### F. Что в web BidShard **не** ищет клиентов

Эти страницы — **onboarding уже купивших**, не prospecting:

- Campaign Integration kit (`integration_kit.ts`) — click URL, S2S postback
- RTB integration page
- Integrations → Supply (`integrations_supply_page.tsx`) — **свой** sellers.json/ads.txt
- Cost Sync, Margin Guard, billing, license

Их анализируем ради **языка продукта** (какие сети и боли заложены), не как crawler.

---

## Telegram — MTProto (Telethon)

Главный канал для gray affiliate **вне РФ/РБ**. Bot API **не подходит** для чтения чужих чатов — нужен **user session** через MTProto.

### Стек

- **Python 3.11+**, [Telethon](https://docs.telethon.dev/) (или Pyrogram — но в README фиксируем Telethon)
- `api_id` / `api_hash` с [my.telegram.org](https://my.telegram.org)
- Session file: `data/telethon.session` (в `.gitignore`)

### Поток

1. `TelegramClient(session, api_id, api_hash)` — один раз интерактивный login (2FA).
2. Список чатов из `config/sources.telegram.yaml` — username или `-100…` id, тег `geo: latam|eu|global`, **exclude** чаты с тегом `ru`.
3. `client.iter_messages(chat, offset_date=since, limit=500)` — только новые с последнего `last_message_id` (хранить в SQLite/Redis).
4. На каждое сообщение: текст + `message.sender` → username, bio если доступен.
5. Extract: `@username`, `t.me/username`, email regex.
6. Geo pre-filter на sender: `lang`, bio keywords, phone из `User` (если виден).
7. Scoring → sink.

### Rate limits

- `asyncio.sleep(1–3)` между `GetHistory`
- FloodWait → exponential backoff, логировать chat_id
- Не больше ~20 чатов за один проход; round-robin

### Целевые чаты (примеры типов, не хардкод)

- EN: affiliate marketing, media buying, iGaming acquisition, nutra sweeps
- ES/PT: tráfico, afiliados, cassino
- EU: arbitrage, CPA, push traffic
- **Не мониторить:** чаты с названиями/описанием «арбитраж РФ», «СНГ», «беларусь»

### Контакт в лиде

Приоритет: `@telegram` username → потом email. Outreach: личка из **отдельного** sales-аккаунта, не из session парсера.

---

## ads.txt и sellers.json — разведка

### Зачем

У affiliate/iGaming/push-сайтов часто есть `ads.txt` — список авторизованных рекламных систем. У сетей с programmatic — `sellers.json` с **`contact_email`**. Это прямой B2B-контакт decision maker'а, не случайный коммент на Reddit.

### Алгоритм

1. **Seed domains** — списки non-RU casino affiliates, push feeds, nutra landers, ad networks (ручной CSV + CT logs).
2. `GET https://{domain}/ads.txt` — plain text, без JS.
3. Парсить строки IAB: `domain, publisher_id, DIRECT|RESELLER, [cert_id]`.
4. Интересны:
   - `DIRECT` на домен похожий на трекер конкурента
   - домены из списка programmatic SSP (сигнал «свой ad stack»)
   - редко — email в комментарии `# contact: ...`
5. `GET https://{domain}/sellers.json` — JSON, поле `contact_email` (BidShard сам так экспортирует — `service_campaign.go` `supplySettingContact`).
6. Обогатить: WHOIS domain → email (geo filter).
7. Scoring: +competitor domain в ads.txt, +`igaming`, +`programmatic`.

### Связь с BidShard product

Админка Supply (`/integrations/supply`) генерирует **исходящие** файлы для **наших** покупателей. Парсер делает **обратное** — читает **чужие** файлы для prospecting. Формат тот же IAB Tech Lab.

### Ограничения

- Многие gray landers **не** публикуют ads.txt — это не единственный источник.
- `contact_email` часто generic (`ads@`, `support@`) — валидировать и скорить ниже, чем личный email из форума.

---

## Скрапинг сайтов и гидратация Next.js

Ленды, pre-landers, affiliate tools часто на **Next.js** (App Router или Pages). Обычный `curl` отдаёт пустой shell — контент появляется после hydration.

### Уровень 1 — без браузера (предпочитать)

**Pages Router — `__NEXT_DATA__`:**

```html
<script id="__NEXT_DATA__" type="application/json">{...}</script>
```

Извлечь JSON → `props.pageProps` — там текст, meta, иногда emails.

**App Router — RSC payload:**

- В HTML искать `self.__next_f.push(...)` — flight data с чанками текста.
- Или `/_next/data/{buildId}/{locale}/{path}.json` (Pages) — buildId из `/_next/static/{buildId}/`.

**Практика в коде:**

```python
# pseudocode
html = httpx.get(url, follow_redirects=True).text
m = re.search(r'<script id="__NEXT_DATA__"[^>]*>(.+?)</script>', html)
if m:
    data = json.loads(m.group(1))
    text = json.dumps(data)  # дальше keyword scan
```

### Уровень 2 — статика в исходнике

Даже на Next часто в первом HTML есть:

- `<meta name="description">`, `<title>`
- Tracking scripts: `voluum`, `keitaro`, `binom`, `redtrack` в `<script src=...>`
- Postback URLs в inline config

Искать regex по сырому HTML **до** hydration.

### Уровень 3 — headless (дорого, fallback)

Playwright / Chromium с `wait_until=networkidle` или `domcontentloaded` + 2–3 s wait — только если L1–L2 пустые.

- Rotate UA, respect `robots.txt` pragmatically
- Кешировать по URL + etag
- **Не** массово скрапить FB/Google — бан

### Связь с BidShard safe page

В трекере BidShard есть client-side **hydrator** для safe page (`safe_page_hydrator.js` / `safe_page_panel.ts`) — он собирает behavioral events для anti-fraud. Это **не** парсинг лидов, но тот же класс задач: «контент/события после load». Для лид-парсера переиспользовать только идею: **сначала embedded JSON, потом JS runtime**.

---

## Сигналы для скоринга

Каталог: [`bidshard-leads/data/keywords.json`](../bidshard-leads/data/keywords.json).

**High intent (20–25):** `voluum alternative`, `keitaro alternative`, `self-hosted tracker`, `tracker too expensive`, `per-event pricing`.

**Pain (15–25):** `budget overburn`, `missing ftd`, `postback failing`, `tracker is down`, `ftd not credited`.

**Vertical (8–12):** `igaming`, `sportsbook`, `nutra`, `sweepstakes`, `cpa network`, `pop traffic`, `push traffic`, `arbitrage`.

**Cloak (10–15):** `safe page`, `cloak`, `white page`, `account ban`.

**Competitor (12):** `voluum`, `keitaro`, `binom`, `redtrack`, `bemob`, `peerclick`.

**Geo negative (hard reject, не score):** `.ru`/`.by` email, `+375`, bio `Russia`/`Belarus`/`Moscow`/`Minsk`.

**LinkedIn negative:** любой контакт только linkedin.com → **reject** (канал закрыт).

Пороги: High ≥ 35, Medium ≥ 15.

---

## Модель данных

Лид в Mongo / JSON export:

```go
type Lead struct {
    HashID   string    // SHA256 нормализованных контактов
    TS       time.Time
    Source   string    // "telegram:stm_en", "ads_txt:example.com"
    Title    string
    Contacts []string  // email, telegram:@user
    Matched  []string
    Score    int
    Priority string    // High | Medium | Low
    Snippet  string    // ≤500 символов контекста
}
```

`hash_id` — dedup по контактам (email case-insensitive, telegram `@user`), не по source/title.

---

## Пайплайн парсера

Цепочка данных: **Source adapters** → **Geo filter** → **Extractor** → **Normalizer** → **Scorer** → **Sink**.

| Слой | Задача |
|------|--------|
| Source adapter | Auth, pagination, rate limit, `last_seen` cursor |
| Geo filter | RF/RB reject до scoring |
| Extractor | email, telegram; Next.js `__NEXT_DATA__` |
| Normalizer | `RawLead`, truncate 500, dedup |
| Scorer | keywords.json + `keywords-gray.json` overlay |
| Sink | Mongo / NDJSON / HTTP |

Rate limit: 500 ms–2 s между HTTP; Telethon — FloodWait backoff.

---

## Структура runtime: горутины, контексты, логи, вывод

Ориентир — `bidshard-leads/cmd/leads/main.go` (scheduler + channel + worker pool), но с явным lifecycle и отменой, чтобы при SIGTERM ничего не висело.

### Дерево пакетов

```
cmd/parser/main.go          # signal ctx, slog init, Run(ctx)
internal/
  app/runner.go             # оркестрация: старт/стоп всех goroutine
  app/shutdown.go           # ordered drain: cancel → close ch → wait
  config/config.go
  model/
  geo/
  sources/
    registry.go             # список Source по имени
    coordinator.go          # fan-out одного scan round
    reddit/, supply/, …
  pipeline/
    worker.go               # N workers, семафор на Mongo
    metrics.go              # счётчики раунда (atomic)
  sink/
    mongo.go | ndjson.go | multi.go
  output/
    reporter.go             # форматированный итог раунда в stdout
  log/log.go                # slog setup: json | text
```

### Горутины и каналы

Один процесс, **5 ролей** goroutine:

| # | Goroutine | Вход | Выход | Завершение |
|---|-----------|------|-------|------------|
| G0 | **main** | SIGINT/SIGTERM | `cancel()` root ctx | `app.Run` return |
| G1 | **scheduler** | ticker `PollInterval` | `scanCh` (буфер 1) | `ctx.Done()` |
| G2 | **coordinator** | `scanCh` | пишет в `taskCh` | `ctx.Done()`; отменяет round-ctx |
| G3 | **pipeline workers** × `WorkerCount` | `taskCh` | `sink` | `taskCh` closed + ctx |
| G4 | **reporter** (опционально) | `statsCh` / ticker | stdout table | `ctx.Done()` |
| G5 | **telethon-ingest** (опционально) | stdin/NDJSON file | `taskCh` | EOF или ctx |

Каналы (все **buffered**, размер из config):

```go
taskCh  := make(chan pipeline.Task, cfg.TaskBuffer)   // default 128
scanCh  := make(chan struct{}, 1)                     // coalesce: не копить тики
statsCh := make(chan pipeline.RoundStats, 8)          // для reporter
```

`scanCh` буфер 1 — если предыдущий раунд ещё идёт, лишний тик отбрасывается (не стакаем scan'ы).

### Контексты: иерархия и отмена

```go
// main.go
ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
defer stop()

if err := app.Run(ctx, cfg); err != nil && !errors.Is(err, context.Canceled) {
    slog.Error("run failed", "error", err)
    os.Exit(1)
}
```

Внутри `app.Run`:

```go
func Run(ctx context.Context, cfg config.Config) error {
    ctx, cancel := context.WithCancel(ctx)
    defer cancel()

    var wg sync.WaitGroup

    // ... start G1–G5 с wg.Add / defer wg.Done

    <-ctx.Done()
    shutdown.Drain(ctx, cancel, &wg, taskCh, cfg.ShutdownTimeout)
    return ctx.Err()
}
```

**Три уровня context:**

| Context | Parent | TTL | Назначение |
|---------|--------|-----|------------|
| `root` | `signal.NotifyContext` | до SIGTERM | жизнь процесса |
| `round` | `root` | `ScanTimeout` (напр. 5m) | один проход всех sources; новый round **отменяет** предыдущий |
| `req` | `round` | `HTTPTimeout` (30s) | один HTTP/DNS/WHOIS вызов |

Coordinator при новом scan:

```go
func (c *Coordinator) RunRound(parent context.Context, taskCh chan<- pipeline.Task) {
    c.roundMu.Lock()
    if c.roundCancel != nil {
        c.roundCancel() // предыдущий round не дожидаемся вечно
    }
    roundCtx, cancel := context.WithTimeout(parent, c.cfg.ScanTimeout)
    c.roundCancel = cancel
    c.roundMu.Unlock()
    defer cancel()

    g, gctx := errgroup.WithContext(roundCtx)
    g.SetLimit(c.cfg.SourceConcurrency) // семафор на параллельные sources

    for _, src := range c.sources {
        src := src
        g.Go(func() error {
            return src.Collect(gctx, func(item model.RawItem) error {
                return c.emit(gctx, taskCh, item)
            })
        })
    }
    _ = g.Wait() // ошибки source логируем внутри, не роняем процесс
}
```

**Правила передачи context (обязательные):**

- Первый аргумент всех `Collect`, `Process`, `Fetch`, `Upsert` — `context.Context`.
- HTTP: `http.NewRequestWithContext(ctx, …)`; `client.Do(req)` + проверка `ctx.Err()` после.
- Mongo: `collection.UpdateOne(ctx, …)` с `context.WithTimeout` на write.
- Циклы: `select { case <-ctx.Done(): return ctx.Err(); case task := <-ch: … }`.
- `time.Sleep` в source **не использовать** — `select` + `time.After` + `ctx.Done()`.
- Telethon sidecar: Go читает NDJSON из pipe; при `ctx.Done()` закрыть `stdin` sidecar'у (SIGTERM процессу).

### emit: не блокировать source на полном taskCh

```go
func (c *Coordinator) emit(ctx context.Context, taskCh chan<- pipeline.Task, item model.RawItem) error {
    select {
    case <-ctx.Done():
        return ctx.Err()
    case taskCh <- pipeline.Task{Source: item.Source, Raw: item.Raw}:
        return nil
    default:
        slog.Warn("task channel full, dropping raw item",
            "source", item.Source,
            "contact", item.MaskedContact(),
        )
        c.metrics.Dropped.Add(1)
        return nil
    }
}
```

### Pipeline worker

По образцу `bidshard-leads/internal/pipeline/worker.go`:

```go
func (p *Pool) worker(ctx context.Context, id int, tasks <-chan pipeline.Task) {
    for {
        select {
        case <-ctx.Done():
            return
        case task, ok := <-tasks:
            if !ok {
                return
            }
            p.process(ctx, id, task)
        }
    }
}
```

`process` — geo → validate → score → sink; на reject — `slog.Debug`, не Error.

Семафор `writeSlots` на Mongo (как `DB_WRITE_SLOTS` в leads) — `semaphore.Acquire(ctx, 1)` перед upsert.

### Shutdown: порядок без зависших goroutine

```go
// shutdown.Drain
1. cancel(root)                    // все select на ctx выходят
2. close(taskCh)                   // workers дочитывают буфер и exit
3. wg.Wait() с timeout:
     waitCh := make(chan struct{})
     go func() { wg.Wait(); close(waitCh) }()
     select {
     case <-waitCh:
     case <-time.After(cfg.ShutdownTimeout): // default 30s
         slog.Warn("shutdown timeout, some goroutines may still exit")
     }
4. sink.Close(ctx)                 // flush NDJSON / mongo disconnect
```

Main **не** делает `close(taskCh)` до `cancel` — иначе race с coordinator.

### Логирование (`log/slog`)

```go
// internal/log/log.go
func NewHandler(format string, level slog.Level) slog.Handler {
    opts := &slog.HandlerOptions{Level: level, AddSource: false}
    switch format {
    case "text":
        return slog.NewTextHandler(os.Stderr, opts) // dev: человекочитаемо
    default:
        return slog.NewJSONHandler(os.Stderr, opts)  // prod / docker
    }
}
```

**Уровни и поля:**

| Level | Когда | Пример полей |
|-------|-------|--------------|
| `Debug` | reject geo/validation, dedup skip | `reason`, `source`, `worker` |
| `Info` | round complete, source done, lead upserted | `round_id`, `duration_ms`, `found`, `accepted`, `priority` |
| `Warn` | rate limit, taskCh full, source partial fail | `source`, `status`, `retry_after` |
| `Error` | mongo down, config load fail | `error`, `op` |

Каждый scan round — `round_id := uuid` в context value или closure; все логи раунда с одним id.

```go
slog.Info("scan round finished",
    "round_id", roundID,
    "duration_ms", elapsed.Milliseconds(),
    "sources_ok", stats.SourcesOK,
    "sources_fail", stats.SourcesFail,
    "raw", stats.RawTotal,
    "accepted", stats.Accepted,
    "rejected_geo", stats.RejectedGeo,
    "high", stats.High,
    "medium", stats.Medium,
)
```

Не логировать полный email/telegram в Info — только `hash` или `j***@domain.com` (как `bidshard-leads/internal/llm/mask.go`).

### Форматированный вывод (stdout)

Разделить **логи** (stderr, structured) и **результат** (stdout, для оператора/cron).

Флаг `-output`:

| Режим | Куда | Формат |
|-------|------|--------|
| `ndjson` (default cron) | stdout | одна строка JSON на лид |
| `table` (default TTY) | stdout | итог раунда + top leads |
| `quiet` | — | только slog |

**Итог раунда (`output/reporter.go`)** — после каждого round в TTY:

```
scan round a3f8c2  duration=42.3s  sources=5/6 ok
  raw=127  accepted=18  rejected_geo=41  dedup=68
  priority: high=3  medium=9  low=6

  HIGH  score=62  voluum alternative · postback failing
        @media_buyer_mx  telegram:affiliate_latam
        «Keitaro postback died again on FTD…»

  HIGH  score=58  tracker too expensive · self-hosted
        ops@igaming-team.com  ads_txt:casino-aff.example
```

NDJSON line на лид (для pipe в Mongo loader / jq):

```json
{"ts":"2026-08-16T00:30:00Z","round_id":"a3f8c2","priority":"High","score":62,"source":"telegram:stm_en","contacts":["telegram:@buyer"],"matched":["voluum alternative","postback failing"]}
```

Reporter goroutine (G4): читает `statsCh` + ring buffer последних N high leads; печатает table **в stdout**, не через slog.

### Конфиг runtime (env)

```env
PARSER_POLL_SEC=120
PARSER_WORKERS=4
PARSER_TASK_BUFFER=128
PARSER_SOURCE_CONCURRENCY=3      # параллельных source в одном round
PARSER_SCAN_TIMEOUT=5m
PARSER_HTTP_TIMEOUT=30s
PARSER_SHUTDOWN_TIMEOUT=30s
PARSER_LOG_FORMAT=json           # json | text
PARSER_LOG_LEVEL=info            # debug | info | warn
PARSER_OUTPUT=table              # table | ndjson | quiet
PARSER_WRITE_SLOTS=8             # mongo semaphore
```

### Telethon sidecar и Go

Python пишет NDJSON в stdout → Go `G5`:

```go
scanner := bufio.NewScanner(os.Stdin)
for scanner.Scan() {
    select {
    case <-ctx.Done():
        return
    default:
    }
    var item model.RawItem
    if json.Unmarshal(scanner.Bytes(), &item) != nil { continue }
    _ = coordinator.Emit(ctx, taskCh, item)
}
```

При shutdown: `cmd.Cancel` убивает Python process group, Go выходит по `ctx.Done()`.

---

## Паттерны BidShard → parser (быстро, высокий ROI)

Парсер — не RTB ingress (сотни тысяч RPS), но те же **операционные** приёмы из BidShard и `bidshard-leads` дают максимум эффекта за минимум кода. Ниже — что переносить, откуда взять, что **не** тащить.

### Уже есть в bidshard-leads — взять как есть

| Паттерн | Где | Что даёт |
|---------|-----|----------|
| Worker pool + `taskCh` | `internal/pipeline/worker.go` | Параллельная обработка без блокировки sources |
| `semaphore.Weighted` на Mongo | `DB_WRITE_SLOTS`, `store.writeSlots` | Не убиваем Mongo при burst crawl |
| `Exists(hash_id)` до write | `pipeline/process` | Дешёвый dedup до scoring/upsert |
| `signal.NotifyContext` | `cmd/leads/main.go` | Корректный SIGTERM |
| Non-blocking notify | `select default` на `NotifyChan` | High-лид не стопорит pipeline |
| `ProcessDirectBatch` | `worker.go` | Один source отдал слайс — batch без лишних chan hops |

**ROI:** copy-paste pipeline + storage interface; не изобретать заново.

### P0 — сделать в фазе 1 (часы, большой эффект)

#### 1. Cheap-before-expensive (как reject hostile wire early в `PARSER_SECURITY.md`)

Порядок в `process()` — каждый шаг отсекает % без I/O:

```
geo.Filter (CPU) → keyword pre-scan (CPU) → extract contacts → score
  → in-memory dedup → Mongo Exists → MX validate (DNS, только если score ≥ medium)
```

Сейчас в leads MX идёт в `ValidateLead` на всё подряд — в parser **MX только для кандидатов на upsert**.

**Отсылка:** BidShard режет malformed body до heap alloc; мы режем DNS до Mongo.

#### 2. Shared `http.Transport` per host pool

Из `postback/postback_sender_worker.go`:

```go
transport := &http.Transport{
    MaxIdleConns:        100,
    MaxIdleConnsPerHost: 20,
    IdleConnTimeout:     90 * time.Second,
}
client := &http.Client{Transport: transport, Timeout: cfg.HTTPTimeout}
```

Один `*http.Client` на процесс (или на source adapter), не `&http.Client{}` на каждый запрос.

**ROI:** ads.txt crawl на 5k доменов — минус тысячи TCP handshake.  
**Файл-образец:** `bidshard/internal/postback/postback_sender_worker.go:58–66`.

#### 3. Circuit breaker на внешний source

Упрощённый порт `ingestion/circuit_breaker.go` — **один breaker на source name** (`reddit`, `github`, `supply`):

| Состояние | Поведение |
|-----------|-----------|
| Closed | Нормальные запросы |
| Open (после 5× 429/5xx подряд) | Skip source до `openTimeout` (60s) |
| Half-open | Один probe-запрос |

```go
if !breakers.For("reddit").Allow() {
    slog.Warn("source circuit open", "source", "reddit", "retry_after", br.WaitDuration())
    return nil
}
```

**ROI:** Reddit/PullPush при 429 не долбим 27 сабов × N раундов → не prolong ban.  
**Файл:** `bidshard/internal/ingestion/circuit_breaker.go`.

#### 4. In-memory dedup ring (local skip перед Mongo)

Аналог **local quanta full-skip** (`ad_local_quota_full_skip_total` — пропуск Redis RTT на hot path):

```go
// internal/dedup/ring.go — LRU или sync.Map на 50k hash_id, TTL 24h
type SeenCache struct { /* ... */ }
func (c *SeenCache) Seen(hashID string) bool
```

В round: `Seen` → skip. После успешного upsert → `Mark`. Mongo `Exists` только если cache miss.

**ROI:** crawl 500 raw / round, 400 repeat → 400 Mongo round-trips → 0.  
**Не нужен:** Redis для этого — процесс локальный, персистентность в Mongo.

#### 5. Per-host rate limiter (`golang.org/x/time/rate`)

Как `PostbackWorker.limiters` map[string]*rate.Limiter:

```go
// supply crawl: max 2 req/s на один registrable domain
lim := limiters.For(host) // 2/s, burst 4
if err := lim.Wait(ctx); err != nil { return err }
```

Отдельные лимиты: `api.github.com` 1/s, `api.pullpush.io` 0.5/s, generic landing 5/s.

**ROI:** не получаем 403 на весь IP; предсказуемая длительность round.

#### 6. Scan coalescing + round cancel

Уже в runtime: `scanCh` buffer 1 + `roundCancel()` при новом тике.

**Отсылка:** micro-batch pause при stream lag (`ad_micro_batch_paused`) — не копим работу, если отстаём.

#### 7. Respect `Retry-After` на 429

Как tracker `429` + `Retry-After: 60` в `handler.go`. Парсить header → `breaker.OpenFor(d)` или `lim.SetLimit(0)` на duration.

---

### P1 — фаза 2–3 (день, заметный выффект)

#### 8. BulkWrite batch sink

Micro-batch из processor (`ad_micro_batch_processed_total`): копить accepted leads в буфер, flush каждые **50 шт** или **2s**:

```go
type MongoSink struct {
    buf   []model.Lead
    mu    sync.Mutex
    flush func(ctx context.Context, batch []model.Lead) error
}
```

`BulkWrite` с `UpdateOne` + `upsert:true` на `hash_id` — один round-trip на 50 лидов.

**ROI:** 18 accepted / round → 1 Mongo call вместо 18.  
**Осторожно:** при shutdown — `flush()` в `Drain` до `close(taskCh)` drain.

#### 9. Exponential backoff на source error

Как `settings_fraud_boost.go` subscribe loop: 1s → 2s → 4s … cap 30s между retry **внутри одного round**, с отменой по `roundCtx`.

#### 10. Atomic round metrics

Как `internal/metrics/collectors.go` — без Prometheus на старте достаточно:

```go
type RoundStats struct {
    RawTotal, Accepted, RejectedGeo, DedupMem, DedupDB atomic.Int64
    High, Medium, Low atomic.Int64
    SourceFail atomic.Int64 // labeled в slog, не в struct
}
```

Reporter читает snapshot в конце round — zero lock на hot path.

#### 11. `errgroup.SetLimit` на source fan-out

Уже в coordinator; лимит **3** по умолчанию = не открывать 6 HTTP sources × 100 goroutines одновременно.

---

### P2 — не тащить (низкий ROI для parser)

| Паттерн BidShard | Почему skip |
|------------------|-------------|
| gnet / epoll engine | HTTP client crawler, не custom protocol server |
| Redis sharding / triple fallback | Нет hot-path budget; Mongo достаточно |
| Local quanta + async Redis streams | Overkill; in-memory dedup ring хватит |
| `sync.Pool` buffer pools | Имеет смысл при 0 alloc/op; у нас bottleneck — network + Mongo |
| ClickHouse spool WAL | Нет columnar analytics в parser |
| eBPF / XDP edge | Не наш слой |
| OpenRTB body parser | Не парсим bid requests |

---

### Сводка: что в какой фазе

| Фаза | Паттерны BidShard |
|------|-------------------|
| **1** | cheap-before-expensive, shared Transport, circuit breaker, seen cache, per-host rate limit, coalescing, Retry-After |
| **2** | Telethon → NDJSON pipe (уже); limiter на sidecar emit rate |
| **3** | BulkWrite sink, backoff, ads.txt с Transport + limiter + breaker |
| **4+** | Playwright pool (max 2 browser) — единственный «дорогой» воркер с отдельным sem |

### Целевые SLO parser (реалистичные, не tracker SLA)

| Метрика | Target | Как мерить |
|---------|--------|------------|
| Full scan round | &lt; 5 min @ 6 sources | `round duration_ms` в slog |
| Mongo writes / accepted lead | ≤ 2 round-trips | cache + bulk |
| Source 429 storm | 0 запросов в open circuit | breaker state в Warn log |
| Shutdown | &lt; 30s, 0 hung goroutine | `shutdown timeout` test |
| Memory | &lt; 256 MB steady | seen cache cap 50k entries |

Tracker BidShard: p99 &lt; 80 ms — **не цель** для batch crawler. Цель — **предсказуемый round** и **не баниться** у внешних API.

---

## План реализации

### Фаза 1 — каркас + geo + runtime + P0 patterns

- `internal/app/runner.go`, `shutdown.go` — goroutine lifecycle, context tree
- `internal/log/log.go` — slog json/text
- `internal/output/reporter.go` — table + ndjson
- `internal/geo/filter.go` — RF/RB rules
- `internal/httpclient/client.go` — shared Transport (BidShard postback worker)
- `internal/breaker/breaker.go` — порт `circuit_breaker.go`
- `internal/dedup/seen.go` — in-memory ring перед Mongo
- `internal/limit/host.go` — `rate.Limiter` per host
- `cmd/parser/main.go`, sink NDJSON + Mongo

### Фаза 2 — Telegram (Telethon)

- `sources/telegram/` — Python sidecar или отдельный `cmd/telethon-scraper`
- gRPC/NDJSON pipe в Go pipeline
- `config/sources.telegram.yaml`

### Фаза 3 — ads.txt / sellers.json

- `sources/supply/` — seed CSV, concurrent HTTP через shared client
- Per-host rate limit + circuit breaker
- `sink/mongo_bulk.go` — BulkWrite batch (P1)

### Фаза 4 — форумы + Next.js landers

- STM, BHW, AffiliateFix HTML adapters
- `sources/lander/` — `__NEXT_DATA__` + optional Playwright

### Фаза 5 — storage

- Mongo upsert + dedup по `hash_id` (контакты)
- JSON export (`-output=ndjson`, `PARSER_EXPORT_JSON`)
- `scripts/ops/backup-mongo.sh` — бэкапы БД

---

## Связь с репозиториями

| Репо | Роль |
|------|------|
| [bidshard](../bidshard) | Продукт; Supply UI — свой ads.txt, не crawler |
| [bidshard-leads](../bidshard-leads) | Отдельный CRM/outreach (не sink парсера) |
| **parser** | Gray sources, geo filter, Telethon, ads.txt, Mongo + JSON export |

---

## Быстрый старт

```bash
cp .env.example .env
# TELEGRAM_API_ID= TELEGRAM_API_HASH=
# MONGO_URI=mongodb://localhost:27017

make build
go run ./cmd/parser -scan-once                    # stub sources
go run ./cmd/parser -source=supply -scan-once     # ads.txt crawl
go run ./cmd/parser -output=ndjson -scan-once     # JSON в stdout
```

### Docker

```bash
cp .env.example .env
make docker-build
make docker-up          # daemon (poll loop)
make docker-run-once    # one-shot
```

Compose: `network_mode: host`, volumes `./data` (sessions, export) и `./config` (telegram yaml).

### Источники (`PARSER_SOURCE` / `-source`)

| Значение | Описание |
|----------|----------|
| `stub` | Фикстуры для smoke |
| `supply` | ads.txt / sellers.json по seed CSV |
| `forum` | HTML-форумы (STM, AffiliateFix) |
| `lander` | Next.js `__NEXT_DATA__` (+ `-lander-headless` для Playwright) |
| ingest | `-ingest-stdin` или `-telegram-sidecar` |

### Telethon

```bash
pip install -r requirements.txt
go run ./cmd/parser -telegram-sidecar -telegram-dry-run -scan-once   # без MTProto session
go run ./cmd/parser -telegram-sidecar -scan-once                     # реальный scrape
```

Session: `data/telethon.session`, cursors: `data/crawler.db`.

Интерактивный login (один раз):

```bash
python3 - <<'PY'
import os
from telethon import TelegramClient
api_id = int(os.environ["TELEGRAM_API_ID"])
api_hash = os.environ["TELEGRAM_API_HASH"]
client = TelegramClient("data/telethon.session", api_id, api_hash)
client.start()
print("session ok")
PY
```

### Хранение и dedup

- **MongoDB:** `MONGO_DB=parser`, коллекция `PARSER_MONGO_COLLECTION=leads`
- **Dedup:** `hash_id` = SHA256 нормализованных контактов (`email`, `telegram:@user`) — не по source/title
- **JSON file:** `PARSER_EXPORT_JSON=data/export/leads.jsonl` (append-only)
- **Seen cache:** in-memory перед Mongo `Exists`

### Бэкап MongoDB

```bash
make backup
# → backups/mongo-YYYYMMDD-HHMMSS/dump.gz

# cron (пример): 0 3 * * * cd /path/parser && make backup
```

Переменные: `MONGO_URI`, `MONGO_DB`, `BACKUP_ROOT`, `KEEP_DAYS` (default 14).  
Если `mongodump` не на хосте: `MONGO_CONTAINER=<имя mongo container>`.

Восстановление:

```bash
make restore DUMP=backups/mongo-20260816-030000/dump.gz
```

### Тесты

```bash
make test
make test-telegram
bash scripts/ci/check_parser_slop.sh
MONGO_URI=mongodb://localhost:27017 go test ./internal/sink/... -tags=integration
```

---

## Пример лида

**Источник:** STM thread «Keitaro postback delay on FTD»  
**Контекст:** «Switching from Voluum, $800/mo at our volume, need self-hosted»  
**Контакты:** buyer@igaming-team.com (WHOIS: CY)  
**Geo:** pass (not RU/BY)  
**Score:** 62 → High  
**Персона:** media-buyer  
**Outreach:** email (не LinkedIn)

