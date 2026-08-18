#!/usr/bin/env python3
from __future__ import annotations

import argparse
import asyncio
import json
import logging
import os
import sys
import time
from typing import Any, TextIO

from .config import ChatConfig, ScraperConfig, load_config
from .cursor import CursorStore

LOG = logging.getLogger("telegram.scraper")


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Telethon MTProto scraper → NDJSON stdout")
    parser.add_argument("--config", default="config/sources.telegram.yaml")
    parser.add_argument("--once", action="store_true", help="single pass over enabled chats")
    parser.add_argument("--stdout", action="store_true", help="emit NDJSON to stdout (default)")
    parser.add_argument("--dry-run", action="store_true", help="emit fixture lines without Telethon")
    return parser.parse_args()


def emit_line(out: TextIO, chat: ChatConfig, text: str, username: str, message_id: int) -> None:
    payload = {
        "source": f"telegram:@{chat.username}",
        "text": text,
        "username": username,
        "message_id": message_id,
    }
    out.write(json.dumps(payload, ensure_ascii=False) + "\n")
    out.flush()


def dry_run(cfg: ScraperConfig, out: TextIO) -> int:
    if not cfg.chats:
        LOG.error("no enabled chats in config")
        return 1
    for i, chat in enumerate(cfg.chats):
        emit_line(
            out,
            chat,
            "voluum alternative needed. postback failing on FTD again.",
            f"media_buyer_{i}",
            1001 + i,
        )
    return 0


def sender_username(sender: Any) -> str:
    from telethon.tl.types import User

    if isinstance(sender, User) and sender.username:
        return sender.username
    return ""


async def scrape_chat(
    client: Any,
    chat: ChatConfig,
    store: CursorStore,
    cfg: ScraperConfig,
    out: TextIO,
) -> int:
    from telethon.errors import FloodWaitError

    chat_key = chat.username or chat.name
    entity = chat.chat_id or chat.username
    if not entity:
        LOG.warning("skip chat without username/chat_id name=%s", chat.name)
        return 0

    last_id = store.get_last_message_id(chat_key)
    max_seen = last_id
    emitted = 0

    for attempt in range(2):
        try:
            async for message in client.iter_messages(entity, limit=cfg.message_limit):
                if message.id <= last_id:
                    break
                if not message.message:
                    continue
                username = sender_username(await message.get_sender())
                emit_line(out, chat, message.message, username, message.id)
                emitted += 1
                if message.id > max_seen:
                    max_seen = message.id
            break
        except FloodWaitError as exc:
            LOG.warning("FloodWait chat=%s seconds=%s attempt=%s", chat_key, exc.seconds, attempt + 1)
            await asyncio.sleep(exc.seconds + 1)
            if attempt == 1:
                return emitted

    if max_seen > last_id:
        store.set_last_message_id(chat_key, max_seen)
    return emitted


async def scrape(cfg: ScraperConfig, out: TextIO) -> int:
    from telethon import TelegramClient
    from telethon.errors import SessionPasswordNeededError

    api_id = os.environ.get("TELEGRAM_API_ID")
    api_hash = os.environ.get("TELEGRAM_API_HASH")
    if not api_id or not api_hash:
        LOG.error("TELEGRAM_API_ID and TELEGRAM_API_HASH required")
        return 1

    store = CursorStore(cfg.cursor_db)
    client = TelegramClient(cfg.session, int(api_id), api_hash)
    await client.connect()
    if not await client.is_user_authorized():
        LOG.error("telethon session not authorized; run interactive login first")
        await client.disconnect()
        store.close()
        return 1

    total = 0
    try:
        for chat in cfg.chats:
            count = await scrape_chat(client, chat, store, cfg, out)
            total += count
            await asyncio.sleep(cfg.poll_delay_sec)
    finally:
        await client.disconnect()
        store.close()

    LOG.info("telegram scrape done chats=%d emitted=%d", len(cfg.chats), total)
    return 0


async def main_async(args: argparse.Namespace) -> int:
    logging.basicConfig(
        level=logging.INFO,
        format="%(asctime)s %(levelname)s %(message)s",
        stream=sys.stderr,
    )
    cfg = load_config(args.config)
    out = sys.stdout

    if args.dry_run:
        return dry_run(cfg, out)

    try:
        return await scrape(cfg, out)
    except Exception as exc:
        from telethon.errors import SessionPasswordNeededError

        if isinstance(exc, SessionPasswordNeededError):
            LOG.error("2FA required; complete login interactively")
            return 1
        raise


def main() -> None:
    args = parse_args()
    try:
        code = asyncio.run(main_async(args))
    except KeyboardInterrupt:
        code = 130
    sys.exit(code)


if __name__ == "__main__":
    main()
