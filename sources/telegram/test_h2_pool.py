import unittest

from sources.telegram.h2_pool import reject_h2_cis_pool


class H2PoolTest(unittest.TestCase):
    def test_blocks_known_cis_handles(self) -> None:
        for user in ("maximaffiliate", "zuevaff", "zuevchatcpa"):
            reject, reason = reject_h2_cis_pool(user, [])
            self.assertTrue(reject, msg=user)
            self.assertEqual(reason, "h2_cis_arbitrage")

    def test_passes_ww_chat(self) -> None:
        reject, _ = reject_h2_cis_pool("buyermedia", ["media buying LATAM"])
        self.assertFalse(reject)

    def test_cpa_rip_ru_setup(self) -> None:
        reject, reason = reject_h2_cis_pool(
            "cpa_chat", ["cpa.rip funnels for Russia +7 support"]
        )
        self.assertTrue(reject)
        self.assertEqual(reason, "h2_cpa_rip_ru")

    def test_cpa_rip_with_voluum_passes(self) -> None:
        reject, _ = reject_h2_cis_pool(
            "aff_chat", ["cpa.rip thread but voluum postback pain"]
        )
        self.assertFalse(reject)


if __name__ == "__main__":
    unittest.main()
