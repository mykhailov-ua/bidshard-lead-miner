#!/usr/bin/env python3
"""Build staggered VPS crontab lines for telegram pain scrape.

Single MTProto account: alternate shards on :07 and :37 UTC (30 min apart, 2/hour).
Multiple hot accounts: interleave sessions (:07/:37, :22/:52, ...) so no sync burst.
"""

from __future__ import annotations

import json
import sys
from typing import Any

# Default shared session until each pool slot has its own parser telegram login.
SHARED_HOT_SESSION = "data/runtime/telethon.session"

# (minute, shard) for one-account mode: 2 runs/hour, shards alternate.
SINGLE_ACCOUNT_SLOTS = ((7, 0), (37, 1))

# Per-session base minute when each hot slot has its own authorized session.
MULTI_ACCOUNT_BASE_MINUTES = (7, 22, 37, 52)


def _hot_sessions(pool: dict[str, Any]) -> list[dict[str, Any]]:
    out: list[dict[str, Any]] = []
    for row in pool.get("sessions", []):
        if row.get("role") != "hot":
            continue
        if row.get("shard") is None:
            continue
        out.append(row)
    return sorted(out, key=lambda r: int(r.get("shard", 0)))


def _session_path(row: dict[str, Any], shared: bool) -> str:
    if shared:
        return SHARED_HOT_SESSION
    custom = str(row.get("cron_session", "")).strip()
    if custom:
        return custom
    return str(row.get("runtime", SHARED_HOT_SESSION)).strip()


def _cron_minutes(row: dict[str, Any], slot_index: int, shared: bool) -> tuple[int, ...]:
    raw = row.get("cron_minutes")
    if isinstance(raw, list) and raw:
        return tuple(int(m) % 60 for m in raw)
    if shared:
        return tuple(m for m, _ in SINGLE_ACCOUNT_SLOTS)
    base = MULTI_ACCOUNT_BASE_MINUTES[slot_index % len(MULTI_ACCOUNT_BASE_MINUTES)]
    return (base, (base + 30) % 60)


def build_cron_lines(pool_path: str, remote_dir: str) -> list[str]:
    with open(pool_path, encoding="utf-8") as fh:
        pool = json.load(fh)

    shard_count = int(pool.get("shard_count") or 1)
    hot = _hot_sessions(pool)
    shared = bool(pool.get("cron_shared_session", True))

    lines = [
        "# lead-intent-processor telegram cron (H9, staggered, no realtime)",
        f"TELEGRAM_SHARD_COUNT={shard_count}",
        "TELEGRAM_LEASE_ENABLED=1",
        "TELEGRAM_SESSION_ROLE=hot",
        f"# mode={'shared_session' if shared else 'per_session'} hot_slots={len(hot)}",
    ]

    if shared and len(hot) > 1:
        for minute, shard in SINGLE_ACCOUNT_SLOTS:
            lines.append(
                f"{minute} * * * * cd {remote_dir} && "
                f"TELEGRAM_SHARD={shard} TELEGRAM_SHARD_COUNT={shard_count} "
                f"TELEGRAM_SESSION={SHARED_HOT_SESSION} TELEGRAM_WORKER_ID=vps-hot-{shard} "
                f"TELEGRAM_SESSION_ROLE=hot TELEGRAM_LEASE_ENABLED=1 "
                f"bash scripts/ops/telegram-pain-cron.sh >> var/telegram-pain-cron.log 2>&1"
            )
    else:
        for idx, row in enumerate(hot):
            shard = int(row.get("shard", idx))
            session = _session_path(row, shared=False)
            worker = f"vps-hot-{row.get('id', shard)}"
            for minute in _cron_minutes(row, idx, shared=False):
                lines.append(
                    f"{minute} * * * * cd {remote_dir} && "
                    f"TELEGRAM_SHARD={shard} TELEGRAM_SHARD_COUNT={shard_count} "
                    f"TELEGRAM_SESSION={session} TELEGRAM_WORKER_ID={worker} "
                    f"TELEGRAM_SESSION_ROLE=hot TELEGRAM_LEASE_ENABLED=1 "
                    f"bash scripts/ops/telegram-pain-cron.sh >> var/telegram-pain-cron.log 2>&1"
                )

    for row in pool.get("sessions", []):
        if row.get("role") != "cold":
            continue
        runtime = row["runtime"]
        discover = str(
            row.get("discover_script", "scripts/ops/buyer-discover-fast.sh")
        ).strip()
        log_name = discover.rsplit("/", 1)[-1].replace(".sh", ".log")
        lines.append(
            f"15 3 * * * cd {remote_dir} && "
            f"TELEGRAM_SESSION={runtime} TELEGRAM_SESSION_ROLE=cold "
            f"bash {discover} >> var/{log_name} 2>&1"
        )

    return lines


def main() -> None:
    pool_path, remote_dir = sys.argv[1], sys.argv[2]
    print("\n".join(build_cron_lines(pool_path, remote_dir)))


if __name__ == "__main__":
    main()
