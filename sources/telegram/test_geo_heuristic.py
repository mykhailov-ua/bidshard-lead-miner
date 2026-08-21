import os
import unittest

from sources.telegram.geo_heuristic import (
    channel_geo_reject,
    geo_heuristic_enabled,
    is_blocked_web_tld,
)


class GeoHeuristicTest(unittest.TestCase):
    def test_blocked_tld(self) -> None:
        self.assertTrue(is_blocked_web_tld("tracker.example.ru"))
        self.assertFalse(is_blocked_web_tld("partner.example.com"))

    def test_rejects_moscow_about(self) -> None:
        self.assertTrue(channel_geo_reject(["Affiliate channel", "Office in Moscow, Russia"]))

    def test_allows_english_about(self) -> None:
        self.assertFalse(
            channel_geo_reject(["LATAM affiliate", "voluum alternative discussion"])
        )

    def test_disabled_via_env(self) -> None:
        os.environ["TELEGRAM_GEO_HEURISTIC"] = "false"
        self.assertFalse(geo_heuristic_enabled())
        self.assertFalse(channel_geo_reject(["Moscow office"]))
        os.environ.pop("TELEGRAM_GEO_HEURISTIC", None)


if __name__ == "__main__":
    unittest.main()
