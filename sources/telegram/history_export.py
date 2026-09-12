"""Historical chat mining for outreach backlog (M5).

Applies the same pain AND-gate as realtime alerts (M3): tracker+pain or
crypto-gray, reachable @username, no channel-broadcast noise.
"""

from __future__ import annotations

import argparse
import asyncio
import csv
import json
import logging
import os
import sys
from datetime import datetime, timezone
from pathlib import Path
from typing import Any, TextIO

from .alert_dispatcher import pain_alert_tags, passes_pain_emit_gate
from .config import ChatConfig, ScraperConfig, load_config
from .connect import connect_telegram_client
from .cursor import CursorStore
from .discover import merge_chat_lists, sync_manual_curation
from .geo_heuristic import channel_geo_reject, channel_geo_texts
from .history_chunk import iter_messages_chunked
from .message_text import combined_message_text
from .pain import (
    has_commercial_pain_intent,
    has_operational_pain_signal,
    is_channel_broadcast_alert_skip,
)
from .prefilter import has_crypto_gray_icp_signal, has_tracker_pain_signal
from .realtime import realtime_chat_allowed
from .scraper import (
    build_telegram_client,
    entity_chat_type,
    fetch_channel_about,
    resolve_chat_entity,
    sender_user_id,
    sender_username,
)
from .session_lock import session_exclusive_lock

LOG = logging.getLogger("telegram.history_export")

_CSV_FIELDS = (
    "username",
    "user_id",
    "posted_at",
    "chat",
    "message_id",
    "link",
    "text",
    "pain_tag",
)


def parse_since_date(raw: str) -> datetime:
    text = (raw or "").strip()
    if not text:
        raise ValueError("empty --since date")
    dt = datetime.strptime(text, "%Y-%m-%d")
    return dt.replace(tzinfo=timezone.utc)


def message_link(chat_username: str, message_id: int) -> str:
    uname = chat_username.strip().lstrip("@").lower()
    if not uname or message_id <= 0:
        return ""
    return f"https://t.me/{uname}/{message_id}"


def build_history_export_row(
    *,
    username: str,
    user_id: int,
    text: str,
    posted_at: datetime | None,
    chat: ChatConfig,
    message_id: int,
) -> dict[str, Any]:
    when = ""
    if posted_at is not None:
        if posted_at.tzinfo is None:
            posted_at = posted_at.replace(tzinfo=timezone.utc)
        when = posted_at.astimezone(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
    chat_label = (chat.username or chat.name or "").strip()
    return {
        "username": username.lstrip("@"),
        "user_id": user_id,
        "posted_at": when,
        "chat": chat_label,
        "message_id": message_id,
        "link": message_link(chat.username, message_id),
        "text": text.strip(),
        "pain_tag": pain_alert_tags(text),
    }


def passes_history_export_filter(
    text: str,
    username: str,
    *,
    chat_type: str = "",
    reply_to_message_id: int = 0,
) -> bool:
    """M3 AND-gate without requiring TELEGRAM_ALERT_ENABLED."""
    return passes_pain_emit_gate(
        text,
        username,
        chat_type=chat_type,
        reply_to_message_id=reply_to_message_id,
    )


def classify_pain_near_miss(
    text: str,
    username: str,
    *,
    chat_type: str = "",
    reply_to_message_id: int = 0,
) -> str | None:
    """Bizdev tuning: one-leg misses that fail passes_pain_emit_gate."""
    if passes_pain_emit_gate(
        text,
        username,
        chat_type=chat_type,
        reply_to_message_id=reply_to_message_id,
    ):
        return None
    if not (username or "").strip().lstrip("@"):
        return None
    if is_channel_broadcast_alert_skip(chat_type, reply_to_message_id, text):
        return None
    if has_crypto_gray_icp_signal(text):
        return None
    has_tracker = has_tracker_pain_signal(text)
    has_pain = has_operational_pain_signal(text) or has_commercial_pain_intent(text)
    if has_tracker and not has_pain:
        return "tracker_only"
    if has_pain and not has_tracker:
        return "pain_only"
    return None


def _open_export_writer(
    out_path: str, fmt: str
) -> tuple[TextIO, Any | None, csv.DictWriter | None]:
    path = Path(out_path)
    path.parent.mkdir(parents=True, exist_ok=True)
    handle = path.open("w", encoding="utf-8", newline="")
    if fmt == "csv":
        writer = csv.DictWriter(handle, fieldnames=_CSV_FIELDS)
        writer.writeheader()
        return handle, None, writer
    return handle, handle, None


def _write_row(
    handle: TextIO,
    ndjson_out: TextIO | None,
    csv_writer: csv.DictWriter | None,
    row: dict[str, Any],
) -> None:
    if csv_writer is not None:
        csv_writer.writerow(row)
        return
    assert ndjson_out is not None
    ndjson_out.write(json.dumps(row, ensure_ascii=False) + "\n")


async def export_chat_history(
    client: Any,
    entity: Any,
    chat: ChatConfig,
    about: str,
    chat_kind: str,
    *,
    since: datetime,
    out_handle: TextIO,
    ndjson_out: TextIO | None,
    csv_writer: csv.DictWriter | None,
    relax: bool = False,
    near_miss: dict[str, int] | None = None,
) -> int:
    emitted = 0
    scanned = 0
    chat_key = chat.channel_key()
    try:
        async for message in iter_messages_chunked(
            client,
            entity,
            total_limit=0,
            stop_before_date=since,
        ):
            scanned += 1
            body = combined_message_text(message)
            if not body:
                continue
            sender = await message.get_sender()
            username = sender_username(sender)
            user_id = sender_user_id(sender)
            reply_to = 0
            if message.reply_to and getattr(message.reply_to, "reply_to_msg_id", None):
                reply_to = int(message.reply_to.reply_to_msg_id)
            if relax and near_miss is not None:
                miss = classify_pain_near_miss(
                    body,
                    username,
                    chat_type=chat_kind,
                    reply_to_message_id=reply_to,
                )
                if miss:
                    near_miss[miss] = near_miss.get(miss, 0) + 1
            if not passes_history_export_filter(
                body,
                username,
                chat_type=chat_kind,
                reply_to_message_id=reply_to,
            ):
                continue
            row = build_history_export_row(
                username=username,
                user_id=user_id,
                text=body,
                posted_at=getattr(message, "date", None),
                chat=chat,
                message_id=int(message.id),
            )
            _write_row(out_handle, ndjson_out, csv_writer, row)
            emitted += 1
    except Exception as exc:
        LOG.warning("history export error chat=%s: %s", chat_key, exc)
    LOG.info(
        "history export done chat=%s scanned=%d emitted=%d since=%s",
        chat_key,
        scanned,
        emitted,
        since.date().isoformat(),
    )
    return emitted


async def resolve_export_targets(
    client: Any, chats: list[ChatConfig], store: CursorStore
) -> list[tuple[Any, ChatConfig, str, str]]:
    out: list[tuple[Any, ChatConfig, str, str]] = []
    for chat in chats:
        if not chat.enabled:
            continue
        if not realtime_chat_allowed(chat):
            LOG.debug(
                "history export skip role chat=%s role=%s",
                chat.name,
                chat.role,
            )
            continue
        chat_key = chat.channel_key()
        try:
            entity = await resolve_chat_entity(client, chat, store)
            full = await client.get_entity(entity)
            about = await fetch_channel_about(client, full)
            if channel_geo_reject(channel_geo_texts(chat.name, about, full)):
                LOG.info("history export skip geo chat=%s", chat_key)
                continue
            kind = entity_chat_type(full)
            out.append((full, chat, about, kind))
        except Exception as exc:
            LOG.warning("history export skip chat=%s: %s", chat_key, exc)
    return out


def _log_history_export_complete(
    *,
    total: int,
    out_path: str,
    exit_code: int,
    relax: bool,
    near_miss: dict[str, int],
) -> None:
    if relax:
        LOG.info(
            "history export near_miss tracker_only=%d pain_only=%d",
            near_miss.get("tracker_only", 0),
            near_miss.get("pain_only", 0),
        )
    LOG.info(
        "history export complete rows=%d path=%s exit=%d",
        total,
        out_path,
        exit_code,
    )


async def run_history_export(
    cfg: ScraperConfig,
    *,
    since: datetime,
    out_path: str,
    fmt: str = "ndjson",
    role_filter: str = "buyer_supergroup",
    relax: bool = False,
) -> int:
    total = 0
    exit_code = 1
    near_miss: dict[str, int] = {"tracker_only": 0, "pain_only": 0}
    api_id = os.environ.get("TELEGRAM_API_ID")
    api_hash = os.environ.get("TELEGRAM_API_HASH")
    if not api_id or not api_hash:
        LOG.error("TELEGRAM_API_ID and TELEGRAM_API_HASH required")
        _log_history_export_complete(
            total=total,
            out_path=out_path,
            exit_code=exit_code,
            relax=relax,
            near_miss=near_miss,
        )
        return exit_code

    os.environ["TELEGRAM_REALTIME_ROLE_FILTER"] = role_filter.strip()

    session_path = Path(cfg.session)
    with session_exclusive_lock(session_path):
        client = build_telegram_client(cfg, api_id, api_hash)
        store: CursorStore | None = None
        handle: TextIO | None = None
        try:
            await connect_telegram_client(client)
            if not await client.is_user_authorized():
                LOG.error("telethon session not authorized; run: parser telegram login")
                return exit_code

            store = CursorStore(cfg.cursor_db)
            sync_manual_curation(cfg.chats, store)
            chats = merge_chat_lists(cfg.chats, store.list_enabled_chats())
            targets = await resolve_export_targets(client, chats, store)
            enabled = sum(1 for c in chats if c.enabled)
            LOG.info(
                "history export chats enabled=%d targets=%d role_filter=%s",
                enabled,
                len(targets),
                role_filter.strip(),
            )
            if not targets:
                LOG.error(
                    "no resolvable chats for history export role_filter=%s",
                    role_filter.strip(),
                )
                return exit_code

            handle, ndjson_out, csv_writer = _open_export_writer(out_path, fmt)
            LOG.info(
                "history export starting chats=%d since=%s out=%s format=%s relax=%s",
                len(targets),
                since.date().isoformat(),
                out_path,
                fmt,
                relax,
            )
            for entity, chat, about, kind in targets:
                total += await export_chat_history(
                    client,
                    entity,
                    chat,
                    about,
                    kind,
                    since=since,
                    out_handle=handle,
                    ndjson_out=ndjson_out,
                    csv_writer=csv_writer,
                    relax=relax,
                    near_miss=near_miss,
                )
            exit_code = 0
            return exit_code
        finally:
            if handle is not None:
                handle.close()
            if store is not None:
                store.close()
            await client.disconnect()
            _log_history_export_complete(
                total=total,
                out_path=out_path,
                exit_code=exit_code,
                relax=relax,
                near_miss=near_miss,
            )


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Export historical Telegram pain messages for outreach (M5)"
    )
    parser.add_argument("--config", default="config/sources.telegram.yaml")
    parser.add_argument(
        "--since",
        required=True,
        help="ISO date YYYY-MM-DD; stop when older messages are reached",
    )
    parser.add_argument(
        "--out",
        default="data/export/tg_history_pain.ndjson",
        help="output file path (NDJSON or CSV)",
    )
    parser.add_argument(
        "--format",
        choices=("ndjson", "csv"),
        default="ndjson",
        help="output format",
    )
    parser.add_argument(
        "--role-filter",
        default=os.environ.get("TELEGRAM_REALTIME_ROLE_FILTER", "buyer_supergroup"),
        help="chat role filter (same as realtime listener)",
    )
    parser.add_argument(
        "--relax",
        action="store_true",
        help="log near-miss counts (tracker_only vs pain_only) for bizdev tuning",
    )
    return parser.parse_args()


async def main_async(args: argparse.Namespace) -> int:
    logging.basicConfig(
        level=logging.INFO,
        format="%(asctime)s %(levelname)s %(message)s",
        stream=sys.stderr,
    )
    cfg = load_config(args.config)
    since = parse_since_date(args.since)
    return await run_history_export(
        cfg,
        since=since,
        out_path=args.out,
        fmt=args.format,
        role_filter=args.role_filter,
        relax=bool(args.relax),
    )


def main() -> None:
    args = parse_args()
    try:
        code = asyncio.run(main_async(args))
    except KeyboardInterrupt:
        code = 130
    except ValueError as exc:
        logging.error("%s", exc)
        code = 1
    sys.exit(code)


if __name__ == "__main__":
    main()
