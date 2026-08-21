import json
import tempfile
import unittest
from pathlib import Path

from sources.telegram.domains import prune_domains


class DomainsPruneTest(unittest.TestCase):
    def test_prune_invalid_hosts(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            path = Path(tmp) / "domains.json"
            path.write_text(
                json.dumps(
                    {
                        "domains": [
                            {"domain": "bojoko.com", "channel": "x"},
                            {"domain": "13-02-2023-1.jpg", "channel": "x"},
                            {"domain": "google.co.uk", "channel": "x"},
                        ]
                    }
                ),
                encoding="utf-8",
            )
            kept, removed = prune_domains(path)
            self.assertEqual(kept, 1)
            self.assertEqual(removed, 2)
            data = json.loads(path.read_text(encoding="utf-8"))
            self.assertEqual([e["domain"] for e in data["domains"]], ["bojoko.com"])


if __name__ == "__main__":
    unittest.main()
