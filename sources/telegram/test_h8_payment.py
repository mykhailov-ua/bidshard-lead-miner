import unittest

from sources.telegram.h8_payment import reject_crypto_payout_only, reject_h8_payment_vertical
from sources.telegram.prefilter import should_emit_message


class H8PaymentTest(unittest.TestCase):
    def test_fop_drop(self) -> None:
        drop, reason = reject_h8_payment_vertical("оплата на ФОП без трекера")
        self.assertTrue(drop)
        self.assertIn("fop", reason)

    def test_ua_usdt_pass(self) -> None:
        text = "Ищем media buyer Киев, USDT TRC20, voluum postback fail"
        self.assertFalse(reject_h8_payment_vertical(text)[0])

    def test_crypto_only_drop(self) -> None:
        self.assertTrue(reject_crypto_payout_only("join vip usdt pump channel")[0])

    def test_prefilter_crypto_only(self) -> None:
        self.assertFalse(should_emit_message("join vip usdt pump channel", "buyer_x"))


if __name__ == "__main__":
    unittest.main()
