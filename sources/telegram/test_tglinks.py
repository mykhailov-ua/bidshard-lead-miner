import unittest

from sources.telegram.tglinks import (
    extract_from_texts,
    telegram_handles,
    telegram_invite_hashes,
)


class TgLinksTest(unittest.TestCase):
    def test_handles_and_invites(self) -> None:
        text = (
            "Join https://t.me/affiliate_latam and @media_buyer_mx "
            "invite https://t.me/+AbCdEfGhIjKlMn"
        )
        handles = telegram_handles(text)
        self.assertIn("affiliate_latam", handles)
        self.assertIn("media_buyer_mx", handles)
        invites = telegram_invite_hashes(text)
        self.assertIn("AbCdEfGhIjKlMn", invites)

    def test_skips_bots(self) -> None:
        handles = telegram_handles("contact @helper_bot or t.me/RealChannel")
        self.assertEqual(handles, ["realchannel"])

    def test_extract_from_texts_dedup(self) -> None:
        handles, invites = extract_from_texts(
            ["@foo_chat https://t.me/foo_chat", "https://t.me/joinchat/OldStyleHash12"]
        )
        self.assertEqual(handles, ["foo_chat"])
        self.assertEqual(invites, ["OldStyleHash12"])


if __name__ == "__main__":
    unittest.main()
