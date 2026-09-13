import unittest

from sources.telegram.pain import (
    is_channel_broadcast_alert_skip,
    message_has_pain,
    message_has_tracker_pain,
)
from sources.telegram.prefilter import has_crypto_gray_icp_signal


class PainTest(unittest.TestCase):
    def test_tracker_and_operational_pain(self) -> None:
        self.assertTrue(message_has_tracker_pain("voluum postback failing"))
        self.assertTrue(message_has_pain("keitaro nginx timeout after upgrade"))
        self.assertFalse(message_has_tracker_pain("keitaro tutorial step by step"))
        self.assertFalse(message_has_tracker_pain("good morning team"))

    def test_crypto_gray_icp_path(self) -> None:
        text = "binom shaving on weekly usdt payouts from network"
        self.assertTrue(has_crypto_gray_icp_signal(text))
        self.assertTrue(message_has_tracker_pain(text))

    def test_h11_m3_gaps(self) -> None:
        self.assertTrue(message_has_tracker_pain("PP shaves leads, how to prove"))
        self.assertTrue(message_has_tracker_pain("Adspect too expensive for FB"))
        self.assertTrue(
            message_has_tracker_pain("keitaro on 200k clicks/day hangs server, admin 2 min load")
        )
        self.assertTrue(message_has_tracker_pain("в трекере 100, в партнерке 75 депозитов"))

    def test_cis_profanity_requires_tracker_context(self) -> None:
        self.assertTrue(message_has_tracker_pain("блять постбек опять не доходит"))
        self.assertTrue(message_has_tracker_pain("keitaro заебал уже третий раз"))
        self.assertTrue(message_has_tracker_pain("йобаний voluum знову впав"))
        self.assertFalse(message_has_tracker_pain("блять как дела ребят"))

    def test_channel_broadcast_skip(self) -> None:
        self.assertTrue(
            is_channel_broadcast_alert_skip("channel", 0, "daily casino tips follow us")
        )
        self.assertFalse(
            is_channel_broadcast_alert_skip(
                "channel", 0, "voluum postback failing after nginx timeout"
            )
        )
        self.assertFalse(
            is_channel_broadcast_alert_skip("supergroup", 0, "random promo")
        )
        self.assertFalse(
            is_channel_broadcast_alert_skip("channel", 42, "random promo")
        )


if __name__ == "__main__":
    unittest.main()
