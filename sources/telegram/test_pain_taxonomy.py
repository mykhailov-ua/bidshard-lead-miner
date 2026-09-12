import unittest

from sources.telegram.pain_taxonomy import classify_bidshard_pain


class PainTaxonomyTest(unittest.TestCase):
    def test_cloak_stack(self) -> None:
        pain = classify_bidshard_pain("Adspect too expensive for FB campaigns")
        self.assertIsNotNone(pain)
        self.assertEqual(pain.pain_bucket, "cloak_stack_cost")
        self.assertEqual(pain.tier_hint, "pro")

    def test_shave_discrepancy(self) -> None:
        pain = classify_bidshard_pain("PP shaves leads, how to prove discrepancy")
        self.assertIsNotNone(pain)
        self.assertEqual(pain.pain_bucket, "shave_discrepancy")

    def test_infra_scale(self) -> None:
        pain = classify_bidshard_pain("keitaro on 200k clicks/day hangs server, admin 2 min load")
        self.assertIsNotNone(pain)
        self.assertEqual(pain.pain_bucket, "infra_scale")
        self.assertEqual(pain.tier_hint, "network")

    def test_noise(self) -> None:
        self.assertIsNone(classify_bidshard_pain("good morning team"))


if __name__ == "__main__":
    unittest.main()
