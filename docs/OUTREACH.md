# BidShard outreach playbook

For BizDev following up on leads from `lead-intent-processor` / `lead-tail`.

Product: [BidShard](https://bidshard.com/) - self-hosted stack (tracker routing, antifraud, reporting) on **your VPS**. Import from Keitaro/Binom. **10-day free pilot**, USDT after.

## When to reach out

- Pain keywords: postback failing, tracker migration, keitaro/binom cost, junk traffic, 502/upstream, click_id lost.
- Technical author: long post with nginx/htop/logs (likely team lead or DevOps).
- Forum/GitHub/Telegram question (not seller broadcast).

Do **not** outreach: CPA network AM promo, account farm, course seller, SSP adops from supply crawl.

## Message frame (Telegram / email)

1. **Mirror pain** (one line from their post, no PII).
2. **Parallel pilot** - run alongside Keitaro, import JSON, compare margin.
3. **Low risk** - no credit card, 10-day pilot, data stays on their server.
4. **CTA** - message @bidshardsupportbot or site Telegram (per landing).

Example (EN):

> Saw your post about [postback delay / Keitaro bill / upstream 502]. We ship a self-hosted stack that replaces tracker + cloaker + separate IVT bill on your VPS. Teams usually run a parallel pilot against Keitaro/Binom (JSON import, no reinstall to start). Happy to send a pilot license if you share expected traffic + VPS spec - same flow as bidshard.com.

Example (RU, if post is RU - only for non-geo-blocked contexts):

> Видел пост про [postback / keitaro / 502]. Self-hosted стек на вашем VPS, импорт из Keitaro/Binom, параллельный пилот 10 дней. Если актуально - напишите трафик и VPS, скинем лицензию.

## Channels

| Channel | Use |
|---------|-----|
| Telegram DM / forum reply | Primary for gray-market buyers |
| Email from ads.txt CONTACT | Only if pain context in supply intel |
| LinkedIn | Company from job OSINT, not candidate |

## CRM handoff

- `PARSER_CRM_WEBHOOK` -> `crm-bot` on accept + after-analysis.
- `/export new 50` in CRM Telegram bot for daily review.
- `lead-tail` on laptop for live accepted leads.

## Historical pain export (M5)

Batch export (`parser telegram history-export`) writes **NDJSON or CSV** under `data/export/`.
This is a **manual outreach queue**, not CRM ingest - same M3 pain AND-gate as realtime alerts.

We keep it file-based (no `historical_prospects` Mongo collection yet) because:

- Bizdev reviews a one-off spreadsheet/NDJSON; no webhook or score pipeline needed.
- Avoids polluting `leads` with 6-12 month old messages that never hit the hot path.
- Re-run is cheap: same command, new `--since`, overwrite or version the export file.

When a row is outreach-worthy, log the DM in your sheet; do not replay export NDJSON into CRM.

VPS run (stops session holders first): `bash scripts/ops/vps-history-export.sh --since YYYY-MM-DD`.
Add `--relax` to log `tracker_only` vs `pain_only` near-miss counts for gate tuning.

## Objections

| Objection | Response |
|-----------|----------|
| Already on Keitaro | Parallel pilot; import JSON; switch when numbers match (site FAQ) |
| Need cloaker | Starter includes funnel protection; test block rates on pilot |
| Not a developer | Need someone on VPS/Docker; wizard + admin console |
| CPA network handles tracking | They are not ICP; individual buyer on own VPS is |
