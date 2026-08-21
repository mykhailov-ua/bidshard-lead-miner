import json
import tempfile
import unittest
from pathlib import Path

from sources.telegram.domains import append_domains


class DomainsMetadataTest(unittest.TestCase):
    def test_append_domains_dict_with_kind(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "domains.json"
            added = append_domains(
                path,
                [
                    {
                        "domain": "affnet.com",
                        "channel": "wooden_blog",
                        "source": "cross_mention",
                        "kind": "mentioned_in_about",
                        "discovered_via": "seed_chat",
                    }
                ],
            )
            self.assertEqual(added, 1)
            data = json.loads(path.read_text(encoding="utf-8"))
            row = data["domains"][0]
            self.assertEqual(row["domain"], "affnet.com")
            self.assertEqual(row["kind"], "mentioned_in_about")
            self.assertEqual(row["discovered_via"], "seed_chat")
            self.assertIn("at", row)

    def test_append_domains_tuple_still_works(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "domains.json"
            added = append_domains(path, [("bojoko.com", "seed", "scrape")])
            self.assertEqual(added, 1)
            data = json.loads(path.read_text(encoding="utf-8"))
            self.assertEqual(data["domains"][0]["domain"], "bojoko.com")

    def test_upgrade_kind_on_existing_domain(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "domains.json"
            append_domains(
                path,
                [
                    {
                        "domain": "affnet.com",
                        "channel": "seed",
                        "source": "scrape",
                        "kind": "mentioned_in_message",
                    }
                ],
            )
            added = append_domains(
                path,
                [
                    {
                        "domain": "affnet.com",
                        "channel": "seed",
                        "source": "cross_mention",
                        "kind": "mentioned_in_about",
                        "discovered_via": "wooden_blog",
                    }
                ],
            )
            self.assertEqual(added, 0)
            data = json.loads(path.read_text(encoding="utf-8"))
            self.assertEqual(len(data["domains"]), 1)
            self.assertEqual(data["domains"][0]["kind"], "mentioned_in_about")
            self.assertEqual(data["domains"][0]["discovered_via"], "wooden_blog")

    def test_legacy_forwarded_from_alias(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "domains.json"
            added = append_domains(
                path,
                [
                    {
                        "domain": "affnet.com",
                        "channel": "seed",
                        "source": "cross_mention",
                        "forwarded_from": "legacy_seed",
                    }
                ],
            )
            self.assertEqual(added, 1)
            data = json.loads(path.read_text(encoding="utf-8"))
            self.assertEqual(data["domains"][0]["discovered_via"], "legacy_seed")


if __name__ == "__main__":
    unittest.main()
