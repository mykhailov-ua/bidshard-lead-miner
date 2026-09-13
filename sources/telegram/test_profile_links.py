import tempfile
import unittest
from pathlib import Path

from sources.telegram.cursor import CursorStore
from sources.telegram.profile_links import enqueue_profile_links, links_from_profile


class ProfileLinksTest(unittest.TestCase):
    def test_links_from_about_and_personal_channel(self) -> None:
        profile = {
            "about": "Team chat https://t.me/voluum_pain join @mediabuy_arb_team",
            "linked_chat_username": "my_personal_channel",
        }
        links = links_from_profile(profile)
        kinds = {(k, v) for k, v, _ in links}
        self.assertIn(("username", "voluum_pain"), kinds)
        self.assertIn(("username", "mediabuy_arb_team"), kinds)
        self.assertIn(("username", "my_personal_channel"), kinds)

    def test_enqueue_respects_fit_and_dedupes(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            store = CursorStore(Path(tmp) / "crawler.db")
            try:
                profile = {
                    "user_id": 42,
                    "username": "buyer1",
                    "about": "https://t.me/new_arb_chat",
                    "cold_outreach_fit": "yes",
                }
                n = enqueue_profile_links(
                    store, profile, enqueued_so_far=0, limit=10
                )
                self.assertEqual(n, 1)
                n2 = enqueue_profile_links(
                    store, profile, enqueued_so_far=1, limit=10
                )
                self.assertEqual(n2, 0)
                pending = store.list_pending_profile_links(10)
                self.assertEqual(len(pending), 1)
                self.assertEqual(pending[0]["link_value"], "new_arb_chat")
            finally:
                store.close()

    def test_enqueue_skips_no_fit(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            store = CursorStore(Path(tmp) / "crawler.db")
            try:
                profile = {
                    "user_id": 7,
                    "username": "botlike",
                    "about": "https://t.me/some_chat",
                    "cold_outreach_fit": "no",
                }
                n = enqueue_profile_links(
                    store, profile, enqueued_so_far=0, limit=10
                )
                self.assertEqual(n, 0)
            finally:
                store.close()


if __name__ == "__main__":
    unittest.main()
