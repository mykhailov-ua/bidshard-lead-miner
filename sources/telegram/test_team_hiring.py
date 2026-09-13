import unittest

from sources.telegram.team_hiring import extract_company_hint, is_telegram_cold_team_post


class TeamHiringTest(unittest.TestCase):
    def test_cold_team_hiring_detected(self) -> None:
        text = "Hiring: media buyer for FB/Google. DM @owner"
        self.assertTrue(is_telegram_cold_team_post(text))

    def test_company_hint_from_hiring_line(self) -> None:
        text = "Acme Media is hiring a senior buyer for EU geo"
        self.assertEqual(extract_company_hint(text), "Acme Media")


if __name__ == "__main__":
    unittest.main()
