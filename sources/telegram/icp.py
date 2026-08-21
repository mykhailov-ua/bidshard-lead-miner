from __future__ import annotations

import json
from pathlib import Path

DEFAULT_ICP_PATH = Path("config/discover.icp.json")


def load_icp_queries(path: str | Path | None = None) -> tuple[list[str], list[str]]:
    p = Path(path or DEFAULT_ICP_PATH)
    if not p.exists():
        return [], []
    data = json.loads(p.read_text(encoding="utf-8"))
    telegram = [
        str(q).strip() for q in data.get("telegram_search", []) if str(q).strip()
    ]
    serp = [str(q).strip() for q in data.get("serp_dorks", []) if str(q).strip()]
    return telegram, serp
