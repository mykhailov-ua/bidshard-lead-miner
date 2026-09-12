#!/usr/bin/env bash
# H7 OTC / crypto desk OSINT - manual checklist only. Do not automate scrape.
#
# Usage: bash scripts/ops/otc-osint-manual.sh

set -euo pipefail

cat <<'EOF'
H7 OTC desk OSINT (manual only)

Policy:
- No parser scrape / Telethon join for closed OTC desks.
- Use for outreach queue notes after H1-H5 cron path is stable.

Checklist per desk referral:
1. Confirm USDT settlement + media buying team (not pure exchanger spam).
2. Capture tracker pain quote (postback, keitaro, redirect) from trusted intro thread.
3. Tag lead source otc_manual in CRM; do not auto-ingest without pain text.
4. Skip RU/CIS grey bookmaker-only desks (H1 conflict).

Reference dorks (manual search, not automated):
- private OTC USDT media buying
- affiliate network fin dir telegram

See docs/H7_OTC_OSINT.md
EOF
