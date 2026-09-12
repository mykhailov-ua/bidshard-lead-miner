import unittest

from sources.telegram.prefilter import (
    has_antifraud_pain_signal,
    has_crypto_gray_icp_signal,
    has_crypto_payout_signal,
)


class CryptoGrayPrefilterTest(unittest.TestCase):
    def test_payout_alone_not_icp(self) -> None:
        self.assertTrue(has_crypto_payout_signal("pay in usdt weekly"))
        self.assertFalse(has_crypto_gray_icp_signal("pay in usdt weekly"))

    def test_infra_and_antifraud(self) -> None:
        self.assertTrue(
            has_crypto_gray_icp_signal("hideclick bot click fraud on keitaro camp")
        )

    def test_antifraud_hint(self) -> None:
        self.assertTrue(has_antifraud_pain_signal("network froze balance after scrubbing"))


if __name__ == "__main__":
    unittest.main()
