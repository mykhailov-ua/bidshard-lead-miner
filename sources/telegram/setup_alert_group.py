"""Create (or reuse) a private supergroup for pain alerts and invite the alert bot."""

from __future__ import annotations

import asyncio
import logging
import os
import sys
from pathlib import Path

from telethon import TelegramClient
from telethon.tl.functions.channels import CreateChannelRequest, InviteToChannelRequest
from telethon.tl.functions.messages import GetDialogsRequest
from telethon.tl.types import InputPeerEmpty

LOG = logging.getLogger("telegram.setup_alert_group")

GROUP_TITLE = os.environ.get("TELEGRAM_ALERT_GROUP_TITLE", "BidShard Pain Alerts")
BOT_USERNAME = os.environ.get("TELEGRAM_ALERT_BOT_USERNAME", "LeadProceesorBot").lstrip("@")


def bot_api_chat_id(channel_id: int) -> str:
    # Bot API supergroup id: -100 + channel_id (Telethon entity.id).
    return f"-100{channel_id}"


async def find_existing_group(client: TelegramClient, title: str):
    dialogs = await client(
        GetDialogsRequest(
            offset_date=None,
            offset_id=0,
            offset_peer=InputPeerEmpty(),
            limit=200,
            hash=0,
        )
    )
    for chat in dialogs.chats:
        if getattr(chat, "title", None) == title and getattr(chat, "megagroup", False):
            return chat
    return None


async def main() -> int:
    api_id = os.environ.get("TELEGRAM_API_ID", "").strip()
    api_hash = os.environ.get("TELEGRAM_API_HASH", "").strip()
    if not api_id or not api_hash:
        LOG.error("TELEGRAM_API_ID and TELEGRAM_API_HASH required")
        return 1

    session = os.environ.get("TELEGRAM_SESSION", "data/runtime/telethon.session")
    Path(session).parent.mkdir(parents=True, exist_ok=True)

    client = TelegramClient(session, int(api_id), api_hash)
    await client.connect()
    if not await client.is_user_authorized():
        LOG.error("telethon session not authorized; run: parser telegram login --qr")
        return 1

    group = await find_existing_group(client, GROUP_TITLE)
    if group is None:
        LOG.info("creating supergroup title=%s", GROUP_TITLE)
        result = await client(
            CreateChannelRequest(
                title=GROUP_TITLE,
                about="Pain alerts from lead-intent-processor",
                megagroup=True,
            )
        )
        group = result.chats[0]
    else:
        LOG.info("reusing existing supergroup title=%s id=%s", GROUP_TITLE, group.id)

    bot = await client.get_entity(BOT_USERNAME)
    try:
        await client(InviteToChannelRequest(group, [bot]))
        LOG.info("invited bot @%s", BOT_USERNAME)
    except Exception as exc:
        # Already member or privacy blocks invite; bot can join via admin link.
        LOG.warning("invite bot skipped: %s", exc)

    chat_id = bot_api_chat_id(group.id)
    print(f"TELEGRAM_ALERT_CHANNEL={chat_id}")
    print(f"TELEGRAM_ALERT_GROUP_TITLE={GROUP_TITLE}")
    await client.disconnect()
    return 0


if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO, format="%(levelname)s %(message)s")
    raise SystemExit(asyncio.run(main()))
