import unittest

from sources.headless.geo_from_proxy import (
    country_code_from_proxy_username,
    infer_locale_timezone,
)


class GeoFromProxyTest(unittest.TestCase):
    def test_country_pl_token(self) -> None:
        user = "customer-session-abc-country-pl"
        self.assertEqual(country_code_from_proxy_username(user), "pl")

    def test_infer_poland(self) -> None:
        url = "http://user-country-de:pass@gw.example.com:823"
        got = infer_locale_timezone(url)
        self.assertIsNotNone(got)
        self.assertEqual(got[0], "de-DE")
        self.assertEqual(got[1], "Europe/Berlin")

    def test_unknown_country_returns_none(self) -> None:
        self.assertIsNone(infer_locale_timezone("http://nohint:pass@1.2.3.4:8080"))


if __name__ == "__main__":
    unittest.main()
