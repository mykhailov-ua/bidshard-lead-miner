import io
import struct
import tempfile
import unittest
from pathlib import Path

try:
    import msgpack
except ImportError:
    msgpack = None

from sources.telegram.ipc import MsgpackStreamSink, NDJSONSink, emit_payload


class NDJSONSinkTest(unittest.TestCase):
    def test_emit_roundtrip(self) -> None:
        buf = io.StringIO()
        sink = NDJSONSink(buf)
        sink.emit({"source": "telegram:@x", "text": "pain", "message_id": 1})
        self.assertIn("telegram:@x", buf.getvalue())


@unittest.skipIf(msgpack is None, "msgpack not installed")
class MsgpackStreamSinkTest(unittest.TestCase):
    def test_framed_payload(self) -> None:
        buf = io.BytesIO()
        wrapper = io.TextIOWrapper(buf, encoding="utf-8", write_through=True)
        sink = MsgpackStreamSink(wrapper)
        sink.emit(
            {
                "source": "telegram:@aff",
                "text": "voluum",
                "message_id": 9,
                "username": "buyer",
            }
        )
        raw = buf.getvalue()
        size = struct.unpack(">I", raw[:4])[0]
        self.assertEqual(size, len(raw) - 4)
        row = msgpack.unpackb(raw[4:], raw=False)
        self.assertEqual(row["message_id"], 9)


class EmitPayloadCompatTest(unittest.TestCase):
    def test_textio_fallback(self) -> None:
        buf = io.StringIO()
        emit_payload(buf, {"source": "telegram:@x", "text": "t", "message_id": 2})
        self.assertIn("message_id", buf.getvalue())


if __name__ == "__main__":
    unittest.main()
