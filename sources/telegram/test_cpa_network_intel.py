import unittest

from sources.telegram.channel_role import infer_channel_role
from sources.telegram.cpa_network_intel import (
    is_cpa_network_supply_promo,
    is_media_buying_team_hiring,
    supply_emit_allowed,
)
from sources.telegram.prefilter import should_emit_message


class CpaNetworkIntelTest(unittest.TestCase):
    def test_infer_supply_role(self) -> None:
        self.assertEqual(
            infer_channel_role("drcashglobal", "dr.cash global channel", ""),
            "supply",
        )
        self.assertEqual(
            infer_channel_role("affiliatepartnersafrica", "1XBET Affiliate Partners", ""),
            "supply",
        )
        self.assertEqual(
            infer_channel_role("chat_affiliates", "Affiliate Marketing Chat", ""),
            "buyer_supergroup",
        )

    def test_team_hiring_passes(self) -> None:
        text = "Hiring media buyer remote, budget $50k/month, keitaro experience"
        self.assertTrue(is_media_buying_team_hiring(text))
        self.assertTrue(supply_emit_allowed(text, channel_role="supply"))
        self.assertTrue(
            should_emit_message(text, "drcashglobal", channel_role="supply")
        )

    def test_network_promo_dropped(self) -> None:
        promo = (
            "Join our CPA network. High converting offers. "
            "Affiliate manager @am_contact. Register now."
        )
        self.assertTrue(is_cpa_network_supply_promo(promo))
        self.assertFalse(supply_emit_allowed(promo, channel_role="supply"))

    def test_buyer_pain_in_network_channel(self) -> None:
        pain = "postback failing on keitaro after network switched payout model"
        self.assertTrue(supply_emit_allowed(pain, channel_role="supply"))


if __name__ == "__main__":
    unittest.main()
