import os
import unittest
from pathlib import Path
from unittest import mock

from sources.headless.proxy_list import (
    profile_dir_name,
    proxy_settings_at,
    proxy_urls_from_env,
    resolve_proxy_index,
)
from sources.headless.storage import storage_state_path


class ProxyListTest(unittest.TestCase):
    def test_proxy_index_from_env(self) -> None:
        with mock.patch.dict(os.environ, {"PARSER_HEADLESS_PROXY_INDEX": "3"}):
            self.assertEqual(resolve_proxy_index(), 3)

    def test_proxy_settings_picks_index(self) -> None:
        env = {
            "PARSER_PROXY_LIST": "http://a:1,http://user:pass@b:2",
            "PARSER_HEADLESS_PROXY_INDEX": "1",
        }
        with mock.patch.dict(os.environ, env, clear=False):
            settings = proxy_settings_at(resolve_proxy_index())
            self.assertIsNotNone(settings)
            self.assertEqual(settings["server"], "http://b:2")
            self.assertEqual(settings["username"], "user")

    def test_storage_path_per_proxy(self) -> None:
        with mock.patch.dict(
            os.environ,
            {"PARSER_HEADLESS_PROFILE_ROOT": "/tmp/hb_profiles"},
        ):
            p = storage_state_path(2)
            self.assertEqual(p, Path("/tmp/hb_profiles/proxy_2/storage_state.json"))

    def test_profile_dir_direct(self) -> None:
        self.assertEqual(profile_dir_name(-1), "direct")


if __name__ == "__main__":
    unittest.main()
