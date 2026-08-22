from __future__ import annotations

import json
import logging
from datetime import UTC, datetime
from pathlib import Path
from typing import Any

from .tglinks import is_valid_web_host

LOG = logging.getLogger("telegram.domains")

DEFAULT_DOMAINS_PATH = "data/runtime/discovered_telegram_domains.json"

# Shared with Go tgweb crawler (internal/sources/tgweb/registry.go).
RegistryEntry = tuple[str, str, str] | dict[str, Any]

_KIND_RANK = {
    "mentioned_in_message": 1,
    "mentioned_in_about": 2,
}


def _kind_rank(kind: str) -> int:
    return _KIND_RANK.get(kind, 0)


def _normalize_entry(entry: RegistryEntry) -> dict[str, Any] | None:
    if isinstance(entry, dict):
        domain = str(entry.get("domain", "")).lower().strip()
        channel = str(entry.get("channel", "")).strip().lstrip("@").lower()
        source = str(entry.get("source", "")).strip() or "discover"
        kind = str(entry.get("kind", "")).strip()
        discovered_via = (
            str(entry.get("discovered_via", entry.get("forwarded_from", "")))
            .strip()
            .lstrip("@")
            .lower()
        )
        row: dict[str, Any] = {
            "domain": domain,
            "channel": channel,
            "source": source,
        }
        if kind:
            row["kind"] = kind
        if discovered_via:
            row["discovered_via"] = discovered_via
        return row

    if not isinstance(entry, tuple) or len(entry) < 3:
        return None
    domain, channel, source = entry[0], entry[1], entry[2]
    return {
        "domain": str(domain).lower().strip(),
        "channel": str(channel).strip().lstrip("@").lower(),
        "source": str(source).strip() or "discover",
    }


def _upgrade_existing_row(existing: dict[str, Any], row: dict[str, Any]) -> bool:
    """Upgrade provenance when a stronger signal arrives for an existing domain."""
    changed = False
    if _kind_rank(str(row.get("kind", ""))) > _kind_rank(str(existing.get("kind", ""))):
        existing["kind"] = row["kind"]
        changed = True
    for field in ("channel", "source", "discovered_via"):
        new_val = str(row.get(field, "")).strip()
        if new_val and not str(existing.get(field, "")).strip():
            existing[field] = new_val
            changed = True
    return changed


def append_domains(
    path: str | Path,
    entries: list[RegistryEntry],
) -> int:
    """Append or upgrade domains. Tuple rows omit kind; dict rows may set kind/discovered_via."""
    p = Path(path or DEFAULT_DOMAINS_PATH)
    data = _read_file(p)
    by_domain: dict[str, dict[str, Any]] = {}
    for entry in data.get("domains", []):
        domain = str(entry.get("domain", "")).lower().strip()
        if domain:
            by_domain[domain] = entry

    added = 0
    upgraded = 0
    now = datetime.now(UTC).isoformat()
    for entry in entries:
        row = _normalize_entry(entry)
        if row is None:
            continue
        domain = row["domain"]
        if not domain or not is_valid_web_host(domain):
            continue
        if domain in by_domain:
            if _upgrade_existing_row(by_domain[domain], row):
                upgraded += 1
            continue
        row["at"] = now
        data.setdefault("domains", []).append(row)
        by_domain[domain] = row
        added += 1

    if added > 0 or upgraded > 0:
        _write_file(p, data)
        LOG.info(
            "telegram domains registry updated path=%s added=%d upgraded=%d total=%d",
            p,
            added,
            upgraded,
            len(data["domains"]),
        )
    return added


def prune_domains(path: str | Path) -> tuple[int, int]:
    """Drop invalid hosts from registry. Returns (kept, removed)."""
    p = Path(path or DEFAULT_DOMAINS_PATH)
    data = _read_file(p)
    kept: list[dict[str, Any]] = []
    removed = 0
    for entry in data.get("domains", []):
        domain = str(entry.get("domain", "")).lower().strip()
        if not domain or not is_valid_web_host(domain):
            removed += 1
            continue
        entry["domain"] = domain
        kept.append(entry)
    if removed > 0:
        data["domains"] = kept
        _write_file(p, data)
        LOG.info(
            "telegram domains pruned path=%s kept=%d removed=%d", p, len(kept), removed
        )
    return len(kept), removed


def _read_file(path: Path) -> dict[str, Any]:
    if not path.exists():
        return {"domains": []}
    try:
        return json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as exc:
        LOG.warning("domains file unreadable path=%s error=%s", path, exc)
        return {"domains": []}


def _write_file(path: Path, data: dict[str, Any]) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(
        json.dumps(data, indent=2, ensure_ascii=False) + "\n", encoding="utf-8"
    )
