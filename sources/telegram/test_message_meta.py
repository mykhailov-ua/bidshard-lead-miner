import unittest
from types import SimpleNamespace

from sources.telegram.message_meta import message_meta_fields


class MessageMetaTest(unittest.TestCase):
    def test_views_and_forward(self) -> None:
        msg = SimpleNamespace(
            views=120,
            forwards=3,
            fwd_from=SimpleNamespace(from_name="Ops", channel_post=9),
            reactions=None,
            media=None,
        )
        meta = message_meta_fields(msg)
        self.assertEqual(meta["views"], 120)
        self.assertEqual(meta["forward"]["from_name"], "Ops")
