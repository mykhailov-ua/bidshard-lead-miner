"""Telethon -> Go IPC: NDJSON stdout or length-prefixed MessagePack over Unix socket."""

from __future__ import annotations

import json
import os
import socket
import struct
import time
from typing import Any, Protocol, TextIO

try:
    import msgpack
except ImportError:  # pragma: no cover - optional until venv install
    msgpack = None  # type: ignore[assignment]


class MessageSink(Protocol):
    def emit(self, payload: dict[str, Any]) -> None: ...


class NDJSONSink:
    def __init__(self, out: TextIO) -> None:
        self._out = out

    def emit(self, payload: dict[str, Any]) -> None:
        self._out.write(json.dumps(payload, ensure_ascii=False) + "\n")
        self._out.flush()


class MsgpackStreamSink:
    """Length-prefixed MessagePack frames (pipe or any binary stream)."""

    def __init__(self, out: TextIO) -> None:
        self._out = out

    def emit(self, payload: dict[str, Any]) -> None:
        if msgpack is None:
            raise RuntimeError("msgpack package required for TELETHON_IPC_FORMAT=msgpack")
        data = msgpack.packb(payload, use_bin_type=True)
        self._out.buffer.write(struct.pack(">I", len(data)))
        self._out.buffer.write(data)
        self._out.buffer.flush()


class MsgpackUDSSink:
    """Unix domain socket client; Go parser listens and accepts one connection."""

    def __init__(self, path: str, retries: int = 100, delay_sec: float = 0.1) -> None:
        if msgpack is None:
            raise RuntimeError("msgpack package required for TELETHON_IPC_SOCKET")
        self._path = path
        self._sock = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
        last_err: Exception | None = None
        for _ in range(max(1, retries)):
            try:
                self._sock.connect(path)
                return
            except OSError as exc:
                last_err = exc
                time.sleep(delay_sec)
        raise ConnectionError(f"telethon ipc connect failed: {self._path}: {last_err}")

    def emit(self, payload: dict[str, Any]) -> None:
        data = msgpack.packb(payload, use_bin_type=True)
        frame = struct.pack(">I", len(data)) + data
        self._sock.sendall(frame)

    def close(self) -> None:
        try:
            self._sock.close()
        except OSError:
            pass


def ipc_format() -> str:
    return os.environ.get("TELETHON_IPC_FORMAT", "ndjson").strip().lower()


def ipc_socket_path() -> str:
    return os.environ.get("TELETHON_IPC_SOCKET", "").strip()


def open_sink(default_out: TextIO) -> MessageSink:
    path = ipc_socket_path()
    fmt = ipc_format()
    if path:
        return MsgpackUDSSink(path)
    if fmt in ("msgpack", "messagepack"):
        return MsgpackStreamSink(default_out)
    return NDJSONSink(default_out)


def emit_payload(sink: MessageSink | TextIO, payload: dict[str, Any]) -> None:
    if hasattr(sink, "emit"):
        sink.emit(payload)  # type: ignore[union-attr]
        return
    sink.write(json.dumps(payload, ensure_ascii=False) + "\n")  # type: ignore[union-attr]
    sink.flush()  # type: ignore[union-attr]
