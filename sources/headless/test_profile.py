import os
import unittest
from unittest import mock

from sources.headless.profile import (
    DEFAULT_USER_AGENT,
    chromium_launch_kwargs,
    load_profile,
)


class ProfileTest(unittest.TestCase):
    def test_default_matches_http_client_ua(self) -> None:
        with mock.patch.dict(os.environ, {}, clear=False):
            os.environ.pop("PARSER_HEADLESS_USER_AGENT", None)
            p = load_profile()
            self.assertEqual(p.user_agent, DEFAULT_USER_AGENT)
            self.assertIn("Sec-Ch-Ua", p.document_headers)
            self.assertEqual(p.viewport["width"], 1920)

    def test_launch_disables_automation_controlled(self) -> None:
        p = load_profile()
        kw = chromium_launch_kwargs(p)
        self.assertIn("--disable-blink-features=AutomationControlled", kw["args"])
        self.assertIn("--enable-automation", kw["ignore_default_args"])

    def test_headed_skips_headless_new_arg(self) -> None:
        with mock.patch.dict(os.environ, {"PARSER_HEADLESS_HEADED": "true"}):
            p = load_profile()
            kw = chromium_launch_kwargs(p)
            self.assertFalse(kw["headless"])
            self.assertNotIn("--headless=new", kw["args"])

    def test_locale_from_proxy_when_unset(self) -> None:
        env = {
            "PARSER_PROXY_LIST": "http://user-country-pl:pass@gw.example:823",
            "PARSER_HEADLESS_PROXY_INDEX": "0",
        }
        with mock.patch.dict(os.environ, env, clear=False):
            os.environ.pop("PARSER_HEADLESS_LOCALE", None)
            os.environ.pop("PARSER_HEADLESS_TIMEZONE", None)
            p = load_profile()
            self.assertEqual(p.locale, "pl-PL")
            self.assertEqual(p.timezone_id, "Europe/Warsaw")


if __name__ == "__main__":
    unittest.main()
