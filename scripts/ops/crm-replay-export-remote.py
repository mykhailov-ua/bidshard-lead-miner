#!/usr/bin/env python3
"""Replay JSONL export leads to local crm-bot webhook (run on VPS)."""
import json
import os
import sys
import time
import urllib.error
import urllib.request
from pathlib import Path

ROOT = Path(__file__).resolve().parents[2]
ENV_PATH = ROOT / ".env"
EXPORT = Path(os.environ.get("EXPORT_PATH", "data/export/leads.jsonl"))
MIN_SCORE = int(os.environ.get("CRM_REPLAY_MIN_SCORE", "50"))
URL = os.environ.get("PARSER_CRM_WEBHOOK_URL", "http://127.0.0.1:8080/v1/leads")


def load_secret() -> str:
    if not ENV_PATH.is_file():
        return ""
    text = ENV_PATH.read_text(encoding="utf-8")
    for key in ("PARSER_CRM_WEBHOOK_SECRET", "CRM_WEBHOOK_SECRET"):
        for line in text.splitlines():
            if line.startswith(f"{key}="):
                return line.split("=", 1)[1].strip()
    return ""


def main() -> int:
    if not EXPORT.is_file():
        print(f"crm-replay-export: missing {EXPORT}", file=sys.stderr)
        return 1
    secret = load_secret()
    sent = skipped = 0
    with EXPORT.open(encoding="utf-8") as f:
        for line in f:
            line = line.strip()
            if not line:
                continue
            doc = json.loads(line)
            score = int(doc.get("score") or 0)
            if score < MIN_SCORE:
                skipped += 1
                continue
            req = urllib.request.Request(
                URL,
                data=line.encode("utf-8"),
                headers={"Content-Type": "application/json"},
                method="POST",
            )
            if secret:
                req.add_header("Authorization", f"Bearer {secret}")
            try:
                with urllib.request.urlopen(req, timeout=15) as resp:
                    if 200 <= resp.status < 300:
                        sent += 1
                    else:
                        print(
                            f"crm-replay-export: POST failed score={score} http={resp.status}",
                            file=sys.stderr,
                        )
            except urllib.error.HTTPError as e:
                print(
                    f"crm-replay-export: POST failed score={score} http={e.code}",
                    file=sys.stderr,
                )
            except Exception as e:
                print(f"crm-replay-export: POST failed score={score} error={e}", file=sys.stderr)
            time.sleep(0.3)
    print(f"crm-replay-export: sent={sent} skipped={skipped} min_score={MIN_SCORE}")
    return 0 if sent > 0 or skipped > 0 else 1


if __name__ == "__main__":
    raise SystemExit(main())
