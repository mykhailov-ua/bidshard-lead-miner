import unittest

from sources.telegram.h1_geo_block import (
    REJECT_CIS_GREY_MARKET,
    REJECT_GEO_BLOCK_RU_BY,
    h1_should_drop,
)


class H1GeoBlockTest(unittest.TestCase):
    def test_rejects_1win_promo(self) -> None:
        drop, reason = h1_should_drop("Join our 1win funnel", "buyer_en")
        self.assertTrue(drop)
        self.assertTrue(reason.startswith(REJECT_CIS_GREY_MARKET))

    def test_passes_ua_voluum_pain(self) -> None:
        text = "Ищем media buyer в Киев, USDT TRC20, voluum postback не сходится"
        drop, reason = h1_should_drop(text, "team_lead")
        self.assertFalse(drop, msg=reason)

    def test_bookmaker_ww_voice_exempt(self) -> None:
        text = "1win postbacks not matching voluum logs, USDT payout"
        drop, _ = h1_should_drop(text, "buyer")
        self.assertFalse(drop)

    def test_rejects_ru_handle(self) -> None:
        drop, reason = h1_should_drop("voluum alternative", "affiliate_team.ru")
        self.assertTrue(drop)
        self.assertTrue(reason.startswith(REJECT_GEO_BLOCK_RU_BY))

    def test_rejects_sber(self) -> None:
        drop, _ = h1_should_drop("Оплата через Сбер, пишите в личку", "")
        self.assertTrue(drop)

    def test_cpa_rip_without_ru_passes(self) -> None:
        drop, _ = h1_should_drop("cpa.rip thread about voluum pricing", "")
        self.assertFalse(drop)


if __name__ == "__main__":
    unittest.main()
