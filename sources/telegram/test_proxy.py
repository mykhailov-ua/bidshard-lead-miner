import unittest

try:
    import socks
except ImportError:
    socks = None  # type: ignore[assignment,misc]

from sources.telegram.proxy import parse_telegram_proxy


class TelegramProxyTest(unittest.TestCase):
    @unittest.skipIf(socks is None, "PySocks not installed")
    def test_parse_socks5(self) -> None:
        proxy = parse_telegram_proxy("socks5://user:pass@proxy.example:1080")
        if proxy is None:
            self.fail("expected proxy tuple")
        self.assertEqual(proxy[1], "proxy.example")
        self.assertEqual(proxy[2], 1080)
        self.assertEqual(proxy[4], "user")
        self.assertEqual(proxy[5], "pass")

    def test_empty_returns_none(self) -> None:
        self.assertIsNone(parse_telegram_proxy(""))

    def test_bad_scheme_raises(self) -> None:
        with self.assertRaises(ValueError):
            parse_telegram_proxy("ftp://proxy.example:21")


if __name__ == "__main__":
    unittest.main()
