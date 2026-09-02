from __future__ import annotations

from typing import Any


def get_media_caption(message: Any) -> str:
    """Caption on photo/document/video when message.message is empty."""
    for attr in ("photo", "document", "video", "audio", "voice", "poll"):
        media = getattr(message, attr, None)
        if media is None:
            continue
        cap = getattr(media, "caption", None)
        if cap:
            return str(cap)
    return ""


def message_body_text(message: Any) -> str:
    """Unified post body: text message, media caption, or Telethon .text."""
    text = getattr(message, "text", None) or getattr(message, "message", None) or ""
    if not text:
        text = get_media_caption(message)
    return str(text).strip()


def entities_extra_text(message: Any, body: str) -> str:
    """Expand URL/mention entities for tglinks when not already in body."""
    entities = getattr(message, "entities", None)
    if not entities or not body:
        return ""
    body_lower = body.lower()
    parts: list[str] = []
    for ent in entities:
        cls = type(ent).__name__
        offset = int(getattr(ent, "offset", 0))
        length = int(getattr(ent, "length", 0))
        if cls == "MessageEntityTextUrl":
            url = str(getattr(ent, "url", "") or "").strip()
            if url and url.lower() not in body_lower:
                parts.append(url)
        elif cls == "MessageEntityUrl":
            snippet = body[offset : offset + length].strip()
            if snippet and snippet.lower() not in body_lower:
                parts.append(snippet)
        elif cls == "MessageEntityMention":
            snippet = body[offset : offset + length].strip()
            if snippet and snippet.lower() not in body_lower:
                parts.append(snippet)
    return " ".join(parts)


def combined_message_text(message: Any) -> str:
    """Body plus entity URL/mention expansions for scrape and NDJSON."""
    body = message_body_text(message)
    extra = entities_extra_text(message, body)
    if extra:
        if body:
            return f"{body}\n{extra}"
        return extra
    return body
