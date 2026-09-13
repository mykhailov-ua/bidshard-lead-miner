"""Telegram Bot API helpers: parse errors and structured logging (no raw dumps)."""

from __future__ import annotations

import json
import logging
from typing import Any

LOG = logging.getLogger("telegram.bot_api")


class TelegramBotAPIError(Exception):
    def __init__(
        self,
        *,
        method: str,
        error_code: int = 0,
        description: str = "",
        http_status: int = 0,
    ) -> None:
        self.method = method
        self.error_code = int(error_code or 0)
        self.description = (description or "").strip()
        self.http_status = int(http_status or 0)
        super().__init__(self.format_short())

    def hint(self) -> str:
        code = self.error_code
        desc = self.description.lower()
        if code == 401:
            return "invalid bot token (TELEGRAM_ALERT_BOT_TOKEN / CRM_TELEGRAM_BOT_TOKEN)"
        if code == 403:
            return "bot blocked, kicked, or not in chat"
        if code == 400 and "chat not found" in desc:
            return "wrong TELEGRAM_ALERT_CHANNEL or bot not added to group"
        if code == 400 and "parse" in desc:
            return "HTML parse error in alert card"
        if code == 409:
            return "another getUpdates poll is running for this bot"
        if code == 429:
            return "rate limited; retry later"
        if self.http_status in (502, 503):
            return "telegram api temporarily unavailable"
        return ""

    def format_short(self) -> str:
        parts = [self.method or "bot_api"]
        if self.error_code:
            parts.append(f"code={self.error_code}")
        if self.http_status:
            parts.append(f"http={self.http_status}")
        if self.description:
            parts.append(self.description)
        hint = self.hint()
        if hint:
            parts.append(f"({hint})")
        return "telegram " + " ".join(parts)

    def log_fields(self) -> dict[str, Any]:
        return {
            "telegram_method": self.method,
            "telegram_error_code": self.error_code,
            "telegram_description": _truncate(self.description, 500),
            "http_status": self.http_status or None,
            "telegram_hint": self.hint() or None,
        }


def parse_bot_api_response(
    method: str,
    http_status: int,
    raw: bytes,
) -> dict[str, Any] | None:
    """Return result dict when ok=true; raise TelegramBotAPIError when ok=false."""
    if not raw:
        raise TelegramBotAPIError(
            method=method,
            http_status=http_status,
            description="empty response body",
        )
    try:
        body = json.loads(raw.decode("utf-8"))
    except (UnicodeDecodeError, json.JSONDecodeError) as exc:
        raise TelegramBotAPIError(
            method=method,
            http_status=http_status,
            description=f"response not json: {_truncate(raw.decode('utf-8', errors='replace'), 200)}",
        ) from exc
    if not isinstance(body, dict):
        raise TelegramBotAPIError(
            method=method,
            http_status=http_status,
            description="response is not an object",
        )
    if body.get("ok"):
        result = body.get("result")
        if isinstance(result, dict):
            return result
        return {}
    raise TelegramBotAPIError(
        method=method,
        error_code=int(body.get("error_code") or 0),
        description=str(body.get("description") or "unknown error"),
        http_status=http_status,
    )


def log_bot_api_error(msg: str, exc: BaseException, **extra: Any) -> None:
    tail = " ".join(f"{k}={v}" for k, v in extra.items() if v is not None)
    if isinstance(exc, TelegramBotAPIError):
        LOG.warning(
            "%s method=%s error_code=%s http_status=%s hint=%s description=%s %s",
            msg,
            exc.method,
            exc.error_code,
            exc.http_status or "",
            exc.hint(),
            _truncate(exc.description, 200),
            tail,
        )
        return
    LOG.warning("%s error=%s %s", msg, _truncate(str(exc), 500), tail)


def _truncate(s: str, max_len: int) -> str:
    s = (s or "").strip()
    if len(s) <= max_len:
        return s
    return s[:max_len] + "..."
