import unittest
from types import SimpleNamespace

from sources.telegram.message_text import (
    combined_message_text,
    entities_extra_text,
    get_media_caption,
    message_body_text,
)


class MessageTextTest(unittest.TestCase):
    def test_caption_only_photo(self) -> None:
        msg = SimpleNamespace(
            text="",
            message="",
            photo=SimpleNamespace(caption="voluum alternative postback failing"),
            entities=None,
        )
        self.assertEqual(
            message_body_text(msg), "voluum alternative postback failing"
        )

    def test_text_or_message_preference(self) -> None:
        msg = SimpleNamespace(
            text="unified text",
            message="raw message",
            entities=None,
        )
        self.assertEqual(message_body_text(msg), "unified text")

    def test_get_media_caption_from_document(self) -> None:
        msg = SimpleNamespace(
            message="",
            document=SimpleNamespace(caption="tracker migration help"),
        )
        self.assertEqual(get_media_caption(msg), "tracker migration help")

    def test_entities_text_url_appended(self) -> None:
        class MessageEntityTextUrl:
            pass

        body = "see offer"
        ent = MessageEntityTextUrl()
        ent.offset = 0
        ent.length = 9
        ent.url = "https://t.me/affiliate_latam"
        extra = entities_extra_text(SimpleNamespace(entities=[ent]), body)
        self.assertIn("https://t.me/affiliate_latam", extra)
        combined = combined_message_text(
            SimpleNamespace(
                text=body,
                message=body,
                entities=[ent],
            )
        )
        self.assertIn("affiliate_latam", combined)


if __name__ == "__main__":
    unittest.main()
