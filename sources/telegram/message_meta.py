"""Extract forwards, views, reactions, and poll structure from Telethon messages."""

from __future__ import annotations

from typing import Any


def message_meta_fields(message: Any) -> dict[str, Any]:
    out: dict[str, Any] = {}
    views = getattr(message, "views", None)
    if views is not None:
        try:
            out["views"] = int(views)
        except (TypeError, ValueError):
            pass

    forwards = getattr(message, "forwards", None)
    if forwards is not None:
        try:
            out["forwards"] = int(forwards)
        except (TypeError, ValueError):
            pass

    fwd = getattr(message, "fwd_from", None)
    if fwd is not None:
        fwd_row = _forward_meta(fwd)
        if fwd_row:
            out["forward"] = fwd_row

    reactions = getattr(message, "reactions", None)
    if reactions is not None:
        rx = _reactions_meta(reactions)
        if rx:
            out["reactions"] = rx

    poll = _poll_meta(message)
    if poll:
        out["poll"] = poll

    return out


def _forward_meta(fwd: Any) -> dict[str, Any]:
    row: dict[str, Any] = {}
    if getattr(fwd, "date", None) is not None:
        row["date"] = fwd.date.isoformat()
    from_name = getattr(fwd, "from_name", None)
    if from_name:
        row["from_name"] = str(from_name)
    channel_post = getattr(fwd, "channel_post", None)
    if channel_post is not None:
        row["channel_post"] = int(channel_post)
    saved_peer = getattr(fwd, "saved_from_peer", None)
    if saved_peer is not None:
        cid = getattr(saved_peer, "channel_id", None)
        if cid is not None:
            row["from_channel_id"] = int(cid)
    saved_from = getattr(fwd, "saved_from_name", None)
    if saved_from:
        row["saved_from_name"] = str(saved_from)
    return row


def _reactions_meta(reactions: Any) -> dict[str, Any]:
    results = getattr(reactions, "results", None) or []
    items: list[dict[str, Any]] = []
    total = 0
    for r in results:
        count = int(getattr(r, "count", 0) or 0)
        total += count
        emoji = getattr(r, "reaction", None)
        label = ""
        if emoji is not None:
            emoticon = getattr(emoji, "emoticon", None)
            if emoticon:
                label = str(emoticon)
        items.append({"emoji": label, "count": count})
    if not items:
        return {}
    return {"total": total, "items": items}


def _poll_meta(message: Any) -> dict[str, Any] | None:
    media = getattr(message, "media", None)
    if media is None:
        return None
    poll = getattr(media, "poll", None)
    if poll is None:
        return None
    question = getattr(poll, "question", None)
    q_text = ""
    if question is not None:
        q_text = str(getattr(question, "text", "") or "")
    answers: list[str] = []
    for ans in getattr(poll, "answers", None) or []:
        text = getattr(ans, "text", None)
        if text is not None:
            answers.append(str(getattr(text, "text", "") or ""))
    if not q_text and not answers:
        return None
    return {
        "question": q_text,
        "answers": answers,
        "closed": bool(getattr(poll, "closed", False)),
        "public_voters": bool(getattr(poll, "public_voters", False)),
    }
