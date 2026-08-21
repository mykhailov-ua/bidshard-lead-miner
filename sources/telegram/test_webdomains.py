import unittest
from unittest.mock import patch

from sources.telegram.tglinks import (
    is_valid_web_host,
    resolve_redirect_url,
    web_domains,
)


class WebDomainsTest(unittest.TestCase):
    def test_extracts_site_domains(self) -> None:
        got = web_domains("https://www.topxpartners.com/contact and spinbetter.io")
        self.assertIn("topxpartners.com", got)
        self.assertIn("spinbetter.io", got)

    def test_skips_social(self) -> None:
        got = web_domains(
            "https://t.me/foo https://instagram.com/bar https://real-aff.net"
        )
        self.assertEqual(got, ["real-aff.net"])

    def test_skips_file_like_hosts(self) -> None:
        got = web_domains("https://www.13-02-2023-1.jpg https://partner.example.com")
        self.assertEqual(got, ["partner.example.com"])

    def test_resolves_shortener_to_final_host(self) -> None:
        with patch(
            "sources.telegram.tglinks.resolve_redirect_url",
            return_value="https://partner.example.com/landing",
        ):
            got = web_domains("https://bit.ly/abc123", resolve_short=True)
        self.assertEqual(got, ["partner.example.com"])

    def test_resolve_redirect_url_uses_opener(self) -> None:
        class FakeResp:
            url = "https://partner.example.com/final"

            def close(self) -> None:
                return None

        def opener(req: object) -> FakeResp:
            return FakeResp()

        final = resolve_redirect_url("https://bit.ly/x", opener=opener)
        self.assertEqual(final, "https://partner.example.com/final")

    def test_is_valid_web_host_rejects_invalid_tld(self) -> None:
        self.assertFalse(is_valid_web_host("durov.gram"))
        self.assertTrue(is_valid_web_host("bojoko.com"))

    def test_skips_google_and_gov(self) -> None:
        self.assertFalse(is_valid_web_host("news.google.com"))
        self.assertFalse(is_valid_web_host("agency.gov"))
        self.assertFalse(is_valid_web_host("www.nhs.gov.uk"))
        got = web_domains("https://maps.google.com/foo https://partner.example.com")
        self.assertEqual(got, ["partner.example.com"])

    def test_skips_crypto_noise(self) -> None:
        got = web_domains(
            "https://magiceden.io https://jup.ag https://affiliate.example.net"
        )
        self.assertEqual(got, ["affiliate.example.net"])


if __name__ == "__main__":
    unittest.main()
