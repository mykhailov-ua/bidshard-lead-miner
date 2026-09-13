import unittest

from sources.telegram.outreach_fit import cold_outreach_fit


class OutreachFitTest(unittest.TestCase):
    def test_buyer_bio_yes(self) -> None:
        fit = cold_outreach_fit(
            {"username": "buyer1", "premium": True},
            "Hiring media buyer for FB campaigns",
        )
        self.assertEqual(fit, "yes")

    def test_bot_no(self) -> None:
        self.assertEqual(cold_outreach_fit({"is_bot": True}, "hello"), "no")
