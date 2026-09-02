#!/usr/bin/env python3
from __future__ import annotations

import argparse
import asyncio
import json
import logging
import os
import sys
from pathlib import Path
from typing import Any, TextIO

from .config import ChatConfig, ScraperConfig, load_config
from .connect import connect_telegram_client
from .cursor import CursorStore
from .channel_search import iter_search_hits
from .discussion import (
    discussion_cursor_key,
    discussion_scrape_enabled,
    linked_discussion_id,
    scrape_discussion_messages,
)
from .discover import chats_for_scrape, merge_chat_lists, run_discover
from .global_search import run_global_search
from .join_policy import resolve_invite_entity
from .domains import RegistryEntry, append_domains
from .pain import message_has_pain
from .registry_export import export_channels_json
from .session_lock import session_exclusive_lock
from .tglinks import web_domains

LOG = logging.getLogger("telegram.scraper")


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Telethon MTProto scraper -> NDJSON stdout"
    )
    parser.add_argument("--config", default="config/sources.telegram.yaml")
    parser.add_argument(
        "--once", action="store_true", help="single pass over enabled chats"
    )
    parser.add_argument(
        "--stdout", action="store_true", help="emit NDJSON to stdout (default)"
    )
    parser.add_argument(
        "--dry-run", action="store_true", help="emit fixture lines without Telethon"
    )
    parser.add_argument(
        "--login",
        action="store_true",
        help="phone/code MTProto login; saves session file from config",
    )
    parser.add_argument(
        "--login-qr",
        action="store_true",
        help="QR login via Telegram app (Settings -> Devices -> Link Desktop Device)",
    )
    parser.add_argument(
        "--discover-only",
        action="store_true",
        help="discover channels (search + SERP registry + invite links); no message scrape",
    )
    parser.add_argument(
        "--login-fresh",
        action="store_true",
        help="delete existing session before login",
    )
    parser.add_argument(
        "--export-registry",
        action="store_true",
        help="write discovered_telegram_channels.json from crawler.db and exit",
    )
    parser.add_argument(
        "--realtime",
        action="store_true",
        help="long-running NewMessage listener (requires TELEGRAM_REALTIME=1)",
    )
    return parser.parse_args()


from .geo_heuristic import channel_geo_reject
from .message_text import combined_message_text, message_body_text
from .prefilter import should_emit_message
from .telethon_retry import call_with_flood_wait, is_flood_wait


def emit_line(
    out: TextIO,
    chat: ChatConfig,
    text: str,
    username: str,
    message_id: int,
    channel_about: str = "",
    reply_to_message_id: int = 0,
    reply_context: str = "",
    chat_type: str = "",
    sender_user_id: int = 0,
) -> bool:
    # Drop spam/empty before NDJSON; Go pipeline never sees filtered messages.
    if not should_emit_message(text):
        return False
    if chat.username:
        source = f"telegram:@{chat.username}"
    elif chat.invite_hash:
        source = f"telegram:invite:{chat.invite_hash[:12]}"
    else:
        source = f"telegram:{chat.name}"
    payload = {
        "source": source,
        "text": text,
        "username": username,
        "message_id": message_id,
    }
    if sender_user_id > 0 and not username:
        payload["sender_user_id"] = sender_user_id
        payload["contact"] = f"telegram:user_id:{sender_user_id}"
    if reply_to_message_id > 0:
        payload["reply_to_message_id"] = reply_to_message_id
    if reply_context:
        payload["reply_context"] = reply_context
    if chat_type:
        payload["chat_type"] = chat_type
    if channel_about:
        payload["channel_about"] = channel_about
    out.write(json.dumps(payload, ensure_ascii=False) + "\n")
    out.flush()
    return True


def dry_run(cfg: ScraperConfig, out: TextIO) -> int:
    LOG.warning(
        "DRY-RUN: fixture NDJSON only - not real Telegram data; pipeline skips fixture:* sources"
    )
    chats = cfg.chats
    if not chats:
        chats = [
            ChatConfig(
                name="fixture_latam", username="affiliate_latam_en", geo="latam"
            ),
            ChatConfig(name="fixture_eu", username="igaming_acquisition", geo="eu"),
            ChatConfig(name="fixture_global", username="stmaffiliate", geo="global"),
        ]
    for i, chat in enumerate(chats):
        payload = {
            "source": f"fixture:telegram:@{chat.username}",
            "fixture": True,
            "text": "voluum alternative needed. postback failing on FTD again.",
            "username": f"media_buyer_{i}",
            "message_id": 1001 + i,
        }
        out.write(json.dumps(payload, ensure_ascii=False) + "\n")
        out.flush()
    return 0


def sender_username(sender: Any) -> str:
    from telethon.tl.types import User

    if isinstance(sender, User) and sender.username:
        return sender.username
    return ""


def sender_user_id(sender: Any) -> int:
    from telethon.tl.types import User

    if isinstance(sender, User) and sender.id:
        return int(sender.id)
    return 0


def entity_chat_type(entity: Any) -> str:
    from telethon.tl.types import Channel, Chat

    if isinstance(entity, Channel):
        return "supergroup" if entity.megagroup else "channel"
    if isinstance(entity, Chat):
        return "group"
    return ""


async def resolve_chat_entity(client: Any, chat: ChatConfig, store: CursorStore) -> Any:
    if chat.chat_id is not None:
        return chat.chat_id
    if chat.invite_hash:
        return await resolve_invite_entity(client, chat, store)
    if chat.username:
        return chat.username
    raise ValueError("chat has no username, invite_hash, or chat_id")


async def fetch_reply_context(
    client: Any, entity: Any, parent_id: int
) -> str:
    """One-hop parent fetch for reply context; skip parent on FloodWait."""
    if parent_id <= 0:
        return ""
    try:
        result = await client.get_messages(entity, ids=parent_id)
    except Exception as exc:
        if is_flood_wait(exc):
            LOG.warning(
                "reply parent fetch flood wait parent_id=%s seconds=%s skip",
                parent_id,
                getattr(exc, "seconds", 0),
            )
            return ""
        LOG.debug(
            "reply parent fetch failed parent_id=%s error=%s", parent_id, exc
        )
        return ""
    parent = result[0] if isinstance(result, list) else result
    if parent is None:
        return ""
    return message_body_text(parent)


async def fetch_channel_about(client: Any, entity: Any) -> str:
    from telethon.tl.functions.channels import GetFullChannelRequest
    from telethon.tl.types import Channel

    if not isinstance(entity, Channel):
        return ""
    try:
        from .telethon_input import channel_input_peer

        full = await client(GetFullChannelRequest(channel_input_peer(entity)))
        about = getattr(full.full_chat, "about", None)
        if about:
            return str(about)
    except Exception as exc:
        LOG.debug("channel about fetch failed error=%s", exc)
    return ""


async def process_scrape_message(
    client: Any,
    entity: Any,
    message: Any,
    chat: ChatConfig,
    about_text: str,
    chat_kind: str,
    out: TextIO,
    store: CursorStore,
    chat_key: str,
) -> bool:
    """Emit one NDJSON row when body passes prefilter. Returns True if emitted."""
    body = combined_message_text(message)
    if not body:
        return False
    sender = await message.get_sender()
    username = sender_username(sender)
    user_id = sender_user_id(sender)
    reply_to = 0
    reply_context = ""
    if message.reply_to and getattr(message.reply_to, "reply_to_msg_id", None):
        reply_to = int(message.reply_to.reply_to_msg_id)
        if reply_to > 0:
            reply_context = await fetch_reply_context(client, entity, reply_to)
    if not emit_line(
        out,
        chat,
        body,
        username,
        message.id,
        channel_about=about_text,
        reply_to_message_id=reply_to,
        reply_context=reply_context,
        chat_type=chat_kind,
        sender_user_id=user_id,
    ):
        return False
    store.record_emit(chat_key, message_has_pain(body))
    return True


def channel_search_enabled(cfg: ScraperConfig, chat: ChatConfig) -> bool:
    if not cfg.channel_search.enabled:
        return False
    if not cfg.channel_search.channel_usernames:
        return bool(chat.username)
    uname = chat.username.lower().lstrip("@")
    return uname in cfg.channel_search.channel_usernames


def channel_search_daily_limit() -> int:
    raw = os.environ.get("TELEGRAM_CHANNEL_SEARCH_LIMIT", "3").strip()
    try:
        return max(0, int(raw))
    except ValueError:
        return 3


async def scrape_discussion_for_channel(
    client: Any,
    chat: ChatConfig,
    full_entity: Any,
    store: CursorStore,
    cfg: ScraperConfig,
    out: TextIO,
    about_text: str,
    chat_key: str,
) -> int:
    if not discussion_scrape_enabled():
        return 0
    linked_id = await linked_discussion_id(client, full_entity)
    if not linked_id:
        return 0
    disc_key = discussion_cursor_key(chat_key)
    try:
        disc_entity = await client.get_entity(linked_id)
    except Exception as exc:
        LOG.warning(
            "discussion entity resolve failed chat=%s linked_id=%s error=%s skip",
            chat_key,
            linked_id,
            exc,
        )
        return 0

    last_id = store.get_last_message_id(disc_key)
    max_seen = last_id
    emitted = 0

    async def on_discussion_message(message: Any) -> bool:
        nonlocal emitted, max_seen
        if await process_scrape_message(
            client,
            disc_entity,
            message,
            chat,
            about_text,
            "discussion",
            out,
            store,
            chat_key,
        ):
            emitted += 1
            max_seen = max(max_seen, message.id)
            return True
        return False

    emitted = await scrape_discussion_messages(
        client,
        disc_entity,
        cursor_key=disc_key,
        last_id=last_id,
        message_limit=cfg.message_limit,
        on_message=on_discussion_message,
    )
    if max_seen > last_id:
        store.set_last_message_id(disc_key, max_seen)
    if emitted:
        LOG.info(
            "discussion scrape chat=%s cursor=%s emitted=%d",
            chat_key,
            disc_key,
            emitted,
        )
    return emitted


async def scrape_chat(
    client: Any,
    chat: ChatConfig,
    store: CursorStore,
    cfg: ScraperConfig,
    out: TextIO,
) -> int:
    chat_key = chat.channel_key()
    try:
        entity = await resolve_chat_entity(client, chat, store)
    except Exception as exc:
        LOG.warning("skip chat=%s: %s", chat_key, exc)
        return 0

    about_text = await fetch_channel_about(client, entity)
    if channel_geo_reject([about_text, chat.name]):
        LOG.info("skip chat geo heuristic chat=%s", chat_key)
        return 0

    full_entity = entity
    try:
        full_entity = await client.get_entity(entity)
    except Exception:
        pass
    chat_kind = entity_chat_type(full_entity)

    last_id = store.get_last_message_id(chat_key)
    max_seen = last_id
    emitted = 0
    texts_for_domains: list[str] = []
    seen_message_ids: set[int] = set()

    async def emit_message(message: Any, *, require_new: bool) -> bool:
        nonlocal emitted, max_seen
        if message.id in seen_message_ids:
            return False
        seen_message_ids.add(message.id)
        body = combined_message_text(message)
        if not body:
            return False
        texts_for_domains.append(body)
        if require_new and message.id <= last_id:
            return False
        if await process_scrape_message(
            client, entity, message, chat, about_text, chat_kind, out, store, chat_key
        ):
            emitted += 1
            if message.id > last_id:
                max_seen = max(max_seen, message.id)
            return True
        return False

    if channel_search_enabled(cfg, chat) and store.can_channel_search(
        chat_key, channel_search_daily_limit()
    ):
        searches, search_emitted = await iter_search_hits(
            client,
            entity,
            cfg.channel_search.terms,
            cfg.channel_search.messages_per_term,
            channel_search_daily_limit(),
            lambda message: emit_message(message, require_new=False),
        )
        store.record_channel_search(chat_key, searches)
        emitted += search_emitted

    async def iter_chat_messages() -> None:
        async for message in client.iter_messages(entity, limit=cfg.message_limit):
            if message.id <= last_id:
                break
            await emit_message(message, require_new=True)

    try:
        await call_with_flood_wait(
            f"scrape:{chat_key}",
            iter_chat_messages,
            attempts=2,
        )
    except Exception as exc:
        if is_flood_wait(exc):
            LOG.warning(
                "FloodWait chat=%s seconds=%s abort chat",
                chat_key,
                getattr(exc, "seconds", 0),
            )
            return emitted
        LOG.warning(
            "skip chat=%s entity=%s: %s (check username/chat_id in config/sources.telegram.yaml)",
            chat_key,
            entity,
            exc,
        )
        return emitted

    if max_seen > last_id:
        store.set_last_message_id(chat_key, max_seen)
    store.mark_scraped(chat_key)

    if cfg.discover.domains_path and texts_for_domains:
        # Register web domains from scrape batch; tgweb Go crawler reads the same JSON file.
        channel_name = (chat.username or chat.name or "").strip().lstrip("@").lower()
        entries: list[RegistryEntry] = []
        seen_domains: set[str] = set()
        for domain in web_domains("\n".join(texts_for_domains)):
            if domain in seen_domains:
                continue
            seen_domains.add(domain)
            entries.append(
                {
                    "domain": domain,
                    "channel": channel_name,
                    "source": "scrape",
                    # Scrape batch cannot split about vs post body; treat as message mention.
                    "kind": "mentioned_in_message",
                }
            )
        if entries:
            added = append_domains(cfg.discover.domains_path, entries)
            if added:
                LOG.info(
                    "telegram scrape domains registered channel=%s added=%d",
                    channel_name,
                    added,
                )

    emitted += await scrape_discussion_for_channel(
        client, chat, full_entity, store, cfg, out, about_text, chat_key
    )

    return emitted


def env_truthy(name: str) -> bool:
    return os.environ.get(name, "").strip().lower() in ("1", "true", "yes")


def login_pending_path(session_path: Path) -> Path:
    return session_path.with_suffix(session_path.suffix + ".login.json")


def wipe_session_files(session_path: Path) -> None:
    for path in (
        session_path,
        Path(str(session_path) + ".session"),
        login_pending_path(session_path),
    ):
        if path.exists():
            path.unlink()


def describe_sent_code(sent: Any) -> str:
    code_type = getattr(sent, "type", None)
    name = type(code_type).__name__ if code_type is not None else "unknown"
    if name == "SentCodeTypeApp":
        return (
            "code delivery: Telegram app only (open Telegram -> chat "
            "'Telegram' / service notifications; SMS is not sent to third-party apps)"
        )
    if name == "SentCodeTypeSms":
        return "code delivery: SMS"
    if name == "SentCodeTypeCall":
        return "code delivery: voice call"
    return f"code delivery: {name}"


def build_telegram_client(cfg: ScraperConfig, api_id: str, api_hash: str) -> Any:
    from sources.telegram.proxy import telegram_proxy_from_env
    from telethon import TelegramClient

    proxy = telegram_proxy_from_env()
    if proxy is not None:
        client = TelegramClient(cfg.session, int(api_id), api_hash, proxy=proxy)
    else:
        client = TelegramClient(cfg.session, int(api_id), api_hash)
    session = client.session
    if env_truthy("TELEGRAM_USE_TEST_DC") and session is not None:
        session.set_dc(2, "149.154.167.40", 443)
    return client


def print_qr_login_help(url: str, session_path: Path) -> None:
    """Show scannable QR + steps (tg:// link alone is not scannable without a QR image)."""
    png_session = session_path.with_suffix(".login-qr.png")
    png_export = Path("data/export/telegram-login-qr.png")
    banner = (
        "\n"
        "=== Telegram QR login ===\n"
        "1. On phone: Telegram -> Settings -> Devices -> Link Desktop Device\n"
        "2. Scan the QR below with THAT screen (terminal ASCII or PNG file)\n"
        "   PNG on host (after run): data/export/telegram-login-qr.png\n"
        "3. Or paste the tg:// link into Saved Messages on phone and tap it\n"
        "4. Keep this terminal open until login completes (~3 min)\n"
        "=========================\n"
    )
    print(banner, flush=True)

    try:
        import qrcode

        qr = qrcode.QRCode(border=1)
        qr.add_data(url)
        qr.make(fit=True)
        qr.print_ascii(invert=True)
        print(flush=True)
        try:
            img = qrcode.make(url)
            for png_path in (png_session, png_export):
                try:
                    png_path.parent.mkdir(parents=True, exist_ok=True)
                    with png_path.open("wb") as fh:
                        img.save(fh)
                    LOG.info("QR PNG saved: %s", png_path)
                except OSError as exc:
                    LOG.warning("could not save QR PNG %s: %s", png_path, exc)
        except ImportError:
            LOG.warning(
                "Pillow missing; scan ASCII QR above or use Saved Messages link"
            )
    except ImportError:
        LOG.warning(
            "install qrcode package for terminal QR; use Saved Messages link fallback"
        )

    print(url, flush=True)
    print(flush=True)


async def login_qr(cfg: ScraperConfig) -> int:
    from telethon.errors import SessionPasswordNeededError

    api_id = os.environ.get("TELEGRAM_API_ID")
    api_hash = os.environ.get("TELEGRAM_API_HASH")
    if not api_id or not api_hash:
        LOG.error("TELEGRAM_API_ID and TELEGRAM_API_HASH required")
        return 1

    password = os.environ.get("TELEGRAM_PASSWORD", "").strip()
    session_path = Path(cfg.session)
    session_path.parent.mkdir(parents=True, exist_ok=True)
    if env_truthy("TELEGRAM_LOGIN_FRESH"):
        wipe_session_files(session_path)

    LOG.info("session file: %s", session_path)
    with session_exclusive_lock(session_path):
        client = build_telegram_client(cfg, api_id, api_hash)
        await connect_telegram_client(client)
        if await client.is_user_authorized():
            me = await client.get_me()
            LOG.info("already authorized user=%s", me.username or me.id)
            await client.disconnect()
            return 0

        qr = await client.qr_login()
        timeout = float(os.environ.get("TELEGRAM_QR_TIMEOUT_SEC", "180"))
        print_qr_login_help(qr.url, session_path)

        try:
            await qr.wait(timeout=timeout)
        except SessionPasswordNeededError:
            if not password:
                LOG.error("2FA enabled; set TELEGRAM_PASSWORD")
                await client.disconnect()
                return 1
            await client.sign_in(password=password)
        except TimeoutError:
            LOG.error(
                "QR login timed out after %ss; retry parser telegram login --qr", timeout
            )
            await client.disconnect()
            return 1

        me = await client.get_me()
        LOG.info("login ok user=%s session=%s", me.username or me.id, session_path)
        login_pending_path(session_path).unlink(missing_ok=True)
        await client.disconnect()
    return 0


async def login(cfg: ScraperConfig) -> int:
    from telethon.errors import SessionPasswordNeededError

    api_id = os.environ.get("TELEGRAM_API_ID")
    api_hash = os.environ.get("TELEGRAM_API_HASH")
    if not api_id or not api_hash:
        LOG.error("TELEGRAM_API_ID and TELEGRAM_API_HASH required")
        return 1

    phone = os.environ.get("TELEGRAM_PHONE", "").strip()
    code = os.environ.get("TELEGRAM_CODE", "").strip()
    password = os.environ.get("TELEGRAM_PASSWORD", "").strip()

    session_path = Path(cfg.session)
    session_path.parent.mkdir(parents=True, exist_ok=True)
    if env_truthy("TELEGRAM_LOGIN_FRESH"):
        wipe_session_files(session_path)

    LOG.info("session file: %s", session_path)
    with session_exclusive_lock(session_path):
        client = build_telegram_client(cfg, api_id, api_hash)
        await connect_telegram_client(client)
        if await client.is_user_authorized():
            me = await client.get_me()
            LOG.info("already authorized user=%s", me.username or me.id)
            await client.disconnect()
            return 0

        if not phone:
            LOG.error("TELEGRAM_PHONE required for login (e.g. +380631517317)")
            await client.disconnect()
            return 1

        pending_path = login_pending_path(session_path)

        if not code:
            sent = await client.send_code_request(phone)
            pending = {"phone": phone, "phone_code_hash": sent.phone_code_hash}
            pending_path.write_text(json.dumps(pending), encoding="utf-8")
            LOG.info(describe_sent_code(sent))
            LOG.error(
                "step 1 done for %s; when code appears, retry with TELEGRAM_CODE=xxxxx "
                "(or use: parser telegram login --qr)",
                phone,
            )
            await client.disconnect()
            return 2

        if not pending_path.exists():
            LOG.error(
                "no pending login for this session; run step 1 without TELEGRAM_CODE "
                "(or parser telegram login --qr)"
            )
            await client.disconnect()
            return 1

        pending = json.loads(pending_path.read_text(encoding="utf-8"))
        pending_phone = pending.get("phone", "")
        phone_hash = pending.get("phone_code_hash", "")
        if pending_phone != phone:
            LOG.error(
                "TELEGRAM_PHONE mismatch (pending %s); use --login-fresh or same phone",
                pending_phone,
            )
            await client.disconnect()
            return 1

        try:
            await client.sign_in(phone, code, phone_code_hash=phone_hash)
        except SessionPasswordNeededError:
            if not password:
                LOG.error("2FA enabled; set TELEGRAM_PASSWORD")
                await client.disconnect()
                return 1
            await client.sign_in(password=password)
        except Exception as exc:
            LOG.error("login failed: %s (use fresh code; try login --qr)", exc)
            await client.disconnect()
            return 1

        me = await client.get_me()
        label = me.username or str(me.id)
        LOG.info("login ok user=%s session=%s", label, session_path)
        pending_path.unlink(missing_ok=True)
        await client.disconnect()
    return 0


def chats_for_scrape_due(manual: list[ChatConfig], store: CursorStore) -> list[ChatConfig]:
    return merge_chat_lists(manual, store.list_due_chats())


async def export_registry(cfg: ScraperConfig) -> int:
    store = CursorStore(cfg.cursor_db)
    try:
        path = cfg.discover.serp_channels_path
        if not path:
            LOG.error("discover.serp_channels_path not configured")
            return 1
        export_channels_json(store, path)
        return 0
    finally:
        store.close()


async def discover_only(cfg: ScraperConfig) -> int:
    api_id = os.environ.get("TELEGRAM_API_ID")
    api_hash = os.environ.get("TELEGRAM_API_HASH")
    if not api_id or not api_hash:
        LOG.error("TELEGRAM_API_ID and TELEGRAM_API_HASH required")
        return 1

    session_path = Path(cfg.session)
    with session_exclusive_lock(session_path):
        store = CursorStore(cfg.cursor_db)
        client = build_telegram_client(cfg, api_id, api_hash)
        await connect_telegram_client(client)
        if not await client.is_user_authorized():
            LOG.error("telethon session not authorized; run: parser telegram login")
            await client.disconnect()
            store.close()
            return 1

        try:
            total = await run_discover(client, cfg.chats, cfg.discover, store)
            if cfg.discover.serp_channels_path:
                export_channels_json(store, cfg.discover.serp_channels_path)
        finally:
            await client.disconnect()
            store.close()

    if total == 0:
        LOG.warning("telegram discover found no channels")
    return 0


async def scrape(cfg: ScraperConfig, out: TextIO) -> int:
    api_id = os.environ.get("TELEGRAM_API_ID")
    api_hash = os.environ.get("TELEGRAM_API_HASH")
    if not api_id or not api_hash:
        LOG.error("TELEGRAM_API_ID and TELEGRAM_API_HASH required")
        return 1

    session_path = Path(cfg.session)
    with session_exclusive_lock(session_path):
        store = CursorStore(cfg.cursor_db)
        client = build_telegram_client(cfg, api_id, api_hash)
        await connect_telegram_client(client)
        if not await client.is_user_authorized():
            LOG.error("telethon session not authorized; run: parser telegram login")
            await client.disconnect()
            store.close()
            return 1

        chats = chats_for_scrape_due(cfg.chats, store)
        if not chats:
            LOG.error(
                "no telegram channels due for scrape; wait for discover or check scrape_interval_days"
            )
            await client.disconnect()
            store.close()
            return 1

        total = 0
        try:
            total += await run_global_search(client, cfg, store, out)
            for chat in chats:
                total += await scrape_chat(client, chat, store, cfg, out)
                await asyncio.sleep(cfg.poll_delay_sec)
            top = store.top_channels_by_pain(10)
            if top:
                LOG.info("telegram channel stats top_pain=%s", json.dumps(top[:5]))
        finally:
            await client.disconnect()
            store.close()

        LOG.info("telegram scrape done chats=%d emitted=%d", len(chats), total)
    return 0


async def main_async(args: argparse.Namespace) -> int:
    logging.basicConfig(
        level=logging.INFO,
        format="%(asctime)s %(levelname)s %(message)s",
        stream=sys.stderr,
    )
    cfg = load_config(args.config)

    if args.login_fresh:
        os.environ["TELEGRAM_LOGIN_FRESH"] = "1"

    if args.login_qr:
        return await login_qr(cfg)

    if args.login:
        return await login(cfg)

    out = sys.stdout

    if args.dry_run:
        return dry_run(cfg, out)

    if args.export_registry:
        return await export_registry(cfg)

    if args.discover_only:
        return await discover_only(cfg)

    if args.realtime:
        from .realtime import realtime_listen

        os.environ["TELEGRAM_REALTIME"] = "1"
        return await realtime_listen(cfg, out)

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
