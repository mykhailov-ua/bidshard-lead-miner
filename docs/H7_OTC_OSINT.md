# H7 OTC / crypto desk OSINT (manual only)

Closed OTC USDT desks may host network owners and finance leads with tracker infra needs. Parser does **not** automate this surface.

## Policy

| Do | Do not |
|----|--------|
| Manual referral notes in CRM | Telethon scrape of closed OTC groups |
| Pain text required before outreach | Auto-CRM from desk membership alone |
| H1 geo gate on bookmaker/grey noise | SERP harvest of OTC dorks |

## Workflow

1. Run `bash scripts/ops/otc-osint-manual.sh` for checklist.
2. When a trusted intro shares tracker pain, ingest via normal Telegram cron path or manual `history-export`.
3. Tag outreach `source=otc_manual` in CRM notes.

## discover.icp.json

`otc_manual_ref` entries are documentation only; they are **not** fed to SERP harvest.
