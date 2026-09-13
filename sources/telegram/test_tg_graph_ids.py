import unittest

from sources.telegram.tg_graph_ids import (
    telegram_chat_ref,
    telegram_chat_ref_from_username,
    telegram_member_edge_key,
    telegram_person_key,
)


class TgGraphIDsTest(unittest.TestCase):
    def test_person_and_chat_refs(self) -> None:
        self.assertEqual(telegram_person_key(42), "tg_user:42")
        self.assertEqual(telegram_chat_ref("u:aff"), "tg_chat:u:aff")
        self.assertEqual(telegram_chat_ref_from_username("@Aff"), "tg_chat:u:aff")

    def test_member_edge(self) -> None:
        edge = telegram_member_edge_key("tg_chat:u:cpa", "tg_user:7")
        self.assertEqual(edge, "tg_member:tg_chat:u:cpa|tg_user:7")


if __name__ == "__main__":
    unittest.main()
