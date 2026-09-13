import unittest

from sources.telegram.bot_api import TelegramBotAPIError, parse_bot_api_response


class BotAPITest(unittest.TestCase):
    def test_parse_ok(self) -> None:
        out = parse_bot_api_response(
            "sendMessage",
            200,
            b'{"ok":true,"result":{"message_id":1}}',
        )
        self.assertEqual(out.get("message_id"), 1)

    def test_parse_not_ok(self) -> None:
        with self.assertRaises(TelegramBotAPIError) as ctx:
            parse_bot_api_response(
                "sendMessage",
                403,
                b'{"ok":false,"error_code":403,"description":"Forbidden"}',
            )
        err = ctx.exception
        self.assertEqual(err.error_code, 403)
        self.assertIn("blocked", err.hint().lower())


if __name__ == "__main__":
    unittest.main()
