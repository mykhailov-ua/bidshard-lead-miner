import tempfile
import unittest
from pathlib import Path
from types import SimpleNamespace
from unittest.mock import AsyncMock, patch

from sources.telegram.cursor import CursorStore
from sources.telegram.user_enrich import UserBioEnricher, user_enrich_limit


class UserEnrichLimitTest(unittest.TestCase):
    def test_default_limit(self) -> None:
        with patch.dict("os.environ", {}, clear=True):
            self.assertEqual(user_enrich_limit(), 40)

    def test_custom_limit(self) -> None:
        with patch.dict("os.environ", {"TELEGRAM_USER_ENRICH_LIMIT": "3"}):
            self.assertEqual(user_enrich_limit(), 3)


class UserBioCacheTest(unittest.TestCase):
    def test_roundtrip(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            store = CursorStore(Path(tmp) / "crawler.db")
            self.assertIsNone(store.get_user_bio(42))
            store.set_user_bio(42, "keitaro media buyer")
            self.assertEqual(store.get_user_bio(42), "keitaro media buyer")
            store.close()


class UserBioEnricherTest(unittest.IsolatedAsyncioTestCase):
    async def test_uses_cache_without_api_call(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            store = CursorStore(Path(tmp) / "crawler.db")
            store.set_user_profile(99, {"about": "binom affiliate", "user_id": 99})
            enricher = UserBioEnricher(store, 5)
            sender = SimpleNamespace(id=99)
            client = SimpleNamespace()
            with patch("sources.telegram.user_enrich._sender_id", return_value=99):
                bio, profile = await enricher.enrich(client, sender)
            self.assertEqual(bio, "binom affiliate")
            self.assertEqual(profile.get("about"), "binom affiliate")
            store.close()

    async def test_rate_limit_blocks_fetch(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            store = CursorStore(Path(tmp) / "crawler.db")
            enricher = UserBioEnricher(store, 0)
            sender = SimpleNamespace(id=77)
            client = SimpleNamespace()
            with patch("sources.telegram.user_enrich._sender_id", return_value=77):
                bio, profile = await enricher.enrich(client, sender)
            self.assertEqual(bio, "")
            self.assertEqual(profile, {})
            store.close()

    async def test_fetch_and_cache(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            store = CursorStore(Path(tmp) / "crawler.db")
            enricher = UserBioEnricher(store, 2)
            sender = SimpleNamespace(id=55)
            client = AsyncMock()
            with patch("sources.telegram.user_enrich._sender_id", return_value=55):
                with patch(
                    "sources.telegram.user_enrich._fetch_full_user_profile",
                    AsyncMock(
                        return_value={
                            "user_id": 55,
                            "about": "voluum postback pain",
                        }
                    ),
                ):
                    bio, profile = await enricher.enrich(client, sender)
            self.assertIn("voluum", bio)
            self.assertEqual(store.get_user_bio(55), "voluum postback pain")
            store.close()


if __name__ == "__main__":
    unittest.main()
