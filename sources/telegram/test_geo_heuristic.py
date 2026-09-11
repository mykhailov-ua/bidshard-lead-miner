import os
import unittest

from sources.telegram.geo_heuristic import (
    ACCEPT_UA,
    NEUTRAL_CIS,
    REJECT_RU,
    channel_geo_reject,
    channel_geo_texts,
    evaluate_channel_geo,
    geo_heuristic_enabled,
    has_ua_affinity,
    is_blocked_web_tld,
    is_ru_infrastructure_hard_stop,
)


class GeoHeuristicTest(unittest.TestCase):
    def test_blocked_tld(self) -> None:
        self.assertTrue(is_blocked_web_tld("tracker.example.ru"))
        self.assertFalse(is_blocked_web_tld("partner.example.com"))

    def test_rejects_moscow_about(self) -> None:
        self.assertEqual(
            evaluate_channel_geo(["Affiliate channel", "Office in Moscow, Russia"]),
            REJECT_RU,
        )

    def test_allows_english_about(self) -> None:
        self.assertEqual(
            evaluate_channel_geo(["LATAM affiliate", "voluum alternative discussion"]),
            NEUTRAL_CIS,
        )

    def test_rejects_blocked_tld_in_about(self) -> None:
        about = (
            "Popular media about traffic arbitrage. Affiliate Marketing, CPA, SEO, SMM. "
            "http://cpalenta.ru"
        )
        self.assertTrue(channel_geo_reject(["cpa_lenta", about]))

    def test_allows_russian_title_without_ru_infra(self) -> None:
        from types import SimpleNamespace

        title = "CPALENTA | Арбитраж трафика"
        entity = SimpleNamespace(title=title)
        self.assertEqual(
            evaluate_channel_geo(channel_geo_texts("cpa_lenta", "", entity)),
            NEUTRAL_CIS,
        )

    def test_accepts_ua_team_russian_text(self) -> None:
        text = "Ищем media buyer в Киев, оплата USDT TRC20, voluum postback не сходится"
        self.assertEqual(evaluate_channel_geo([text]), ACCEPT_UA)
        self.assertTrue(has_ua_affinity(text))
        self.assertFalse(channel_geo_reject([text]))

    def test_neutral_cis_russian_arbitrage(self) -> None:
        text = "арбитраж трафика на gambling, keitaro alternative"
        self.assertEqual(evaluate_channel_geo([text]), NEUTRAL_CIS)
        self.assertFalse(channel_geo_reject([text]))

    def test_message_hard_stop_sber(self) -> None:
        self.assertTrue(
            is_ru_infrastructure_hard_stop("Оплата через Сбер, пишите в личку")
        )

    def test_disabled_via_env(self) -> None:
        os.environ["TELEGRAM_GEO_HEURISTIC"] = "false"
        self.assertFalse(geo_heuristic_enabled())
        self.assertFalse(channel_geo_reject(["Moscow office"]))
        self.assertFalse(is_ru_infrastructure_hard_stop("оплата через сбер"))
        os.environ.pop("TELEGRAM_GEO_HEURISTIC", None)


if __name__ == "__main__":
    unittest.main()
