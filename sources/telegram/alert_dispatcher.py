"""Pain-trigger alerts to a private bizdev Telegram channel (Bot API)."""

from __future__ import annotations

import asyncio
import html
import json
import logging
import os
import time
import urllib.error
import urllib.parse
import urllib.request

from .bot_api import TelegramBotAPIError, log_bot_api_error, parse_bot_api_response
from datetime import datetime, timezone
from typing import Any

from .h1_geo_block import h1_should_drop
from .pain import is_channel_broadcast_alert_skip, message_has_tracker_pain
from .pain_taxonomy import classify_bidshard_pain, format_pain_bucket_label, format_tier_label
from .prefilter import ANTIFRAUD_PAIN_HINTS, CRYPTO_PAYOUT_HINTS, PAIN_HINTS, TRACKER_PAIN_HINTS, has_crypto_gray_icp_signal

LOG = logging.getLogger("telegram.alert")


def alert_configured() -> bool:
    return bool(_bot_token() and _alert_channel())


def _bot_token() -> str:
    return os.environ.get("TELEGRAM_ALERT_BOT_TOKEN", "").strip()


def _alert_channel() -> str:
    return os.environ.get("TELEGRAM_ALERT_CHANNEL", "").strip()


def _highlight_pain_keyword(text: str) -> tuple[str, str]:
    """Return (highlighted_html, matched_keyword)."""
    lower = text.lower()
    for hint in (
        TRACKER_PAIN_HINTS
        + tuple(PAIN_HINTS)
        + tuple(CRYPTO_PAYOUT_HINTS)
        + tuple(ANTIFRAUD_PAIN_HINTS)
    ):
        idx = lower.find(hint)
        if idx < 0:
            continue
        end = idx + len(hint)
        before = html.escape(text[:idx])
        match = html.escape(text[idx:end])
        after = html.escape(text[end:])
        return f"{before}<b>{match}</b>{after}", hint
    escaped = html.escape(text)
    return escaped, ""


def _author_line(username: str, user_id: int) -> str:
    if username:
        handle = username.lstrip("@")
        return f'<a href="https://t.me/{html.escape(handle)}">@{html.escape(handle)}</a>'
    if user_id > 0:
        return f"user_id:{user_id}"
    return "unknown"


def _message_link(chat_username: str, message_id: int) -> str | None:
    uname = chat_username.strip().lstrip("@").lower()
    if not uname or message_id <= 0:
        return None
    return f"https://t.me/{uname}/{message_id}"


def format_pain_alert_card(
    *,
    username: str,
    user_id: int,
    text: str,
    posted_at: datetime | None,
    chat_username: str,
    message_id: int,
    source_label: str,
) -> str:
    quote, keyword = _highlight_pain_keyword(text.strip())
    if len(quote) > 900:
        quote = quote[:900] + "..."
    when = ""
    if posted_at is not None:
        if posted_at.tzinfo is None:
            posted_at = posted_at.replace(tzinfo=timezone.utc)
        when = posted_at.astimezone(timezone.utc).strftime("%Y-%m-%d %H:%M UTC")
    else:
        when = "unknown"

    pain = classify_bidshard_pain(text)
    header = "<b>Pain alert</b>"
    if pain is not None:
        header = (
            f"<b>[HOT] POTENTIAL CLIENT [{html.escape(format_tier_label(pain.tier_hint))}]</b>"
        )

    lines = [
        header,
        f"Contact: {_author_line(username, user_id)}",
        f"Chat: @{html.escape(chat_username.strip().lstrip('@'))}",
    ]
    if pain is not None:
        lines.append(f"Pain: {html.escape(format_pain_bucket_label(pain.pain_bucket))}")
        lines.append(f"BidShard angle: {html.escape(pain.pitch_line)}")
    lines.append(f"Source: {html.escape(source_label)}")
    link = _message_link(chat_username, message_id)
    if link:
        lines.append(f'<a href="{html.escape(link)}">Open message</a>')
    lines.append(f"Posted: {when}")
    if keyword:
        lines.append(f"Keyword: <code>{html.escape(keyword)}</code>")
    lines.append("")
    lines.append(quote)
    return "\n".join(lines)


def _send_bot_message_sync(chat_id: str, text: str, token: str) -> None:
    url = f"https://api.telegram.org/bot{token}/sendMessage"
    payload = {
        "chat_id": chat_id,
        "text": text,
        "parse_mode": "HTML",
        "disable_web_page_preview": True,
    }
    data = json.dumps(payload).encode("utf-8")
    last_exc: BaseException | None = None
    for attempt in range(4):
        req = urllib.request.Request(
            url,
            data=data,
            method="POST",
            headers={"Content-Type": "application/json"},
        )
        try:
            with urllib.request.urlopen(req, timeout=15) as resp:
                raw = resp.read()
                status = int(getattr(resp, "status", 200) or 200)
        except urllib.error.HTTPError as exc:
            raw = exc.read() if exc.fp else b""
            try:
                parse_bot_api_response("sendMessage", exc.code, raw)
            except TelegramBotAPIError as api_exc:
                last_exc = api_exc
                if api_exc.error_code == 429 and attempt < 3:
                    time.sleep(1.0 + attempt)
                    continue
                raise
        else:
            parse_bot_api_response("sendMessage", status, raw)
            return
    if last_exc is not None:
        raise last_exc


async def dispatch_pain_alert(
    *,
    username: str,
    user_id: int,
    text: str,
    posted_at: datetime | None,
    chat_username: str,
    message_id: int,
    source_label: str,
) -> None:
    if not alert_configured():
        return
    token = _bot_token()
    channel = _alert_channel()
    card = format_pain_alert_card(
        username=username,
        user_id=user_id,
        text=text,
        posted_at=posted_at,
        chat_username=chat_username,
        message_id=message_id,
        source_label=source_label,
    )
    try:
        await asyncio.to_thread(_send_bot_message_sync, channel, card, token)
        LOG.info(
            "pain alert sent channel=%s author=%s msg_id=%d tag=%s",
            channel,
            username or user_id,
            message_id,
            pain_alert_tags(text),
        )
    except (urllib.error.URLError, TelegramBotAPIError, TimeoutError) as exc:
        log_bot_api_error(
            "pain alert send failed",
            exc,
            channel=channel,
            message_id=message_id,
        )


def passes_pain_emit_gate(
    text: str,
    username: str = "",
    *,
    chat_type: str = "",
    reply_to_message_id: int = 0,
    channel_about: str = "",
) -> bool:
    """M3 pain AND-gate without env or Bot API side effects."""
    if h1_should_drop(text, username, channel_about=channel_about)[0]:
        return False
    if not message_has_tracker_pain(text):
        return False
    if not (username or "").strip().lstrip("@"):
        return False
    if is_channel_broadcast_alert_skip(chat_type, reply_to_message_id, text):
        return False
    return True


def should_alert_on_emit(
    text: str,
    username: str = "",
    *,
    chat_type: str = "",
    reply_to_message_id: int = 0,
) -> bool:
    if os.environ.get("TELEGRAM_ALERT_ENABLED", "0").strip().lower() not in (
        "1",
        "true",
        "yes",
    ):
        return False
    return passes_pain_emit_gate(
        text,
        username,
        chat_type=chat_type,
        reply_to_message_id=reply_to_message_id,
        channel_about="",
    )


def pain_alert_tags(text: str) -> str:
    """Log/ops tag for alert cohort (M11 crypto-gray)."""
    if has_crypto_gray_icp_signal(text):
        return "crypto_gray"
    return "tracker_pain"
