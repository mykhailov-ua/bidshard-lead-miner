import unittest
from datetime import datetime, timezone

from sources.telegram.alert_dispatcher import (
    format_pain_alert_card,
    should_alert_on_emit,
)


class AlertDispatcherTest(unittest.TestCase):
    def test_format_card_highlights_keyword(self) -> None:
        card = format_pain_alert_card(
            username="buyer_ops",
            user_id=0,
            text="Keitaro postback timeout after nginx upstream timed out",
            posted_at=datetime(2026, 3, 11, 12, 30, tzinfo=timezone.utc),
            chat_username="aff_chat",
            message_id=42,
            source_label="aff_chat",
        )
        self.assertIn("buyer_ops", card)
        self.assertIn("<b>Keitaro</b>", card)
        self.assertIn("2026-03-11 12:30 UTC", card)
        self.assertIn("t.me/aff_chat/42", card)

    def test_should_alert_requires_env(self) -> None:
        import os
        from unittest.mock import patch

        with patch.dict(os.environ, {}, clear=True):
            self.assertFalse(should_alert_on_emit("postback failing", "buyer_ops"))
        with patch.dict(
            os.environ,
            {"TELEGRAM_ALERT_ENABLED": "1"},
            clear=True,
        ):
            self.assertTrue(
                should_alert_on_emit(
                    "voluum postback failing after nginx timeout",
                    "buyer_ops",
                )
            )
            self.assertFalse(should_alert_on_emit("keitaro", "buyer_ops"))
            self.assertFalse(should_alert_on_emit("hello world", "buyer_ops"))
            self.assertFalse(
                should_alert_on_emit(
                    "voluum postback failing",
                    "",
                )
            )
            self.assertFalse(
                should_alert_on_emit(
                    "daily casino tips",
                    "buyer_ops",
                    chat_type="channel",
                    reply_to_message_id=0,
                )
            )
            self.assertFalse(
                should_alert_on_emit(
                    "melbet landing pack for sale",
                    "buyer_ops",
                )
            )
            self.assertTrue(
                should_alert_on_emit(
                    "voluum postback failing after nginx timeout",
                    "buyer_ops",
                )
            )
            self.assertTrue(
                should_alert_on_emit(
                    "PP shaves leads, how to prove",
                    "buyer_ops",
                )
            )
            self.assertTrue(
                should_alert_on_emit(
                    "Adspect too expensive for FB",
                    "traffic_lead",
                )
            )


if __name__ == "__main__":
    unittest.main()
