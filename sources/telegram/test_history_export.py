import unittest
from datetime import datetime, timezone

from sources.telegram.alert_dispatcher import passes_pain_emit_gate
from sources.telegram.config import ChatConfig
from sources.telegram.history_export import (
    build_history_export_row,
    classify_pain_near_miss,
    parse_outreach_fit_filter,
    parse_since_date,
    passes_history_export_filter,
    passes_outreach_fit_gate,
)


class HistoryExportFilterTest(unittest.TestCase):
    def test_tracker_without_pain_rejected(self) -> None:
        self.assertFalse(
            passes_history_export_filter("we use keitaro daily", "buyer1")
        )

    def test_pain_without_username_rejected(self) -> None:
        self.assertFalse(
            passes_history_export_filter(
                "keitaro postback failing again", ""
            )
        )

    def test_tracker_and_pain_accepted(self) -> None:
        self.assertTrue(
            passes_history_export_filter(
                "keitaro postback failing again", "buyer1"
            )
        )

    def test_export_filter_matches_alert_gate(self) -> None:
        text = "binom migration from voluum, postback timeout"
        username = "media_buyer"
        self.assertEqual(
            passes_history_export_filter(text, username),
            passes_pain_emit_gate(text, username),
        )


class HistoryExportRowTest(unittest.TestCase):
    def test_build_row(self) -> None:
        chat = ChatConfig(name="team", username="affchat")
        posted = datetime(2025, 6, 1, 12, 30, tzinfo=timezone.utc)
        row = build_history_export_row(
            username="@buyer",
            user_id=42,
            text="keitaro postback failing",
            posted_at=posted,
            chat=chat,
            message_id=99,
        )
        self.assertEqual(row["username"], "buyer")
        self.assertEqual(row["user_id"], 42)
        self.assertEqual(row["posted_at"], "2025-06-01T12:30:00Z")
        self.assertEqual(row["chat"], "affchat")
        self.assertEqual(row["message_id"], 99)
        self.assertEqual(row["link"], "https://t.me/affchat/99")


class ParseSinceTest(unittest.TestCase):
    def test_parse_since_date(self) -> None:
        dt = parse_since_date("2025-03-01")
        self.assertEqual(dt, datetime(2025, 3, 1, tzinfo=timezone.utc))


class OutreachFitFilterTest(unittest.TestCase):
    def test_parse_outreach_fit(self) -> None:
        self.assertEqual(parse_outreach_fit_filter("yes,maybe"), {"yes", "maybe"})

    def test_gate_requires_fit_when_filtered(self) -> None:
        allowed = {"yes", "maybe"}
        self.assertTrue(passes_outreach_fit_gate("yes", allowed))
        self.assertFalse(passes_outreach_fit_gate("", allowed))
        self.assertFalse(passes_outreach_fit_gate("no", allowed))


class NearMissTest(unittest.TestCase):
    def test_tracker_only_near_miss(self) -> None:
        self.assertEqual(
            classify_pain_near_miss("we use keitaro daily", "buyer1"),
            "tracker_only",
        )

    def test_pain_only_near_miss(self) -> None:
        self.assertEqual(
            classify_pain_near_miss("nginx timeout on upstream 502", "buyer1"),
            "pain_only",
        )

    def test_full_gate_not_near_miss(self) -> None:
        self.assertIsNone(
            classify_pain_near_miss("keitaro postback failing again", "buyer1")
        )

    def test_crypto_gray_not_near_miss(self) -> None:
        self.assertIsNone(
            classify_pain_near_miss(
                "binom shaving on weekly usdt payouts", "buyer1"
            )
        )


if __name__ == "__main__":
    unittest.main()
