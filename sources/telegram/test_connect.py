import unittest
from unittest import mock

from sources.telegram.connect import connect_telegram_client


class ConnectTelegramClientTest(unittest.TestCase):
    def test_retries_on_session_locked(self) -> None:
        client = mock.AsyncMock()
        import sqlite3

        client.connect = mock.AsyncMock(
            side_effect=[
                sqlite3.OperationalError("database is locked"),
                None,
            ]
        )

        async def run() -> None:
            await connect_telegram_client(client, retries=3, base_delay_sec=0)

        import asyncio

        asyncio.run(run())
        self.assertEqual(client.connect.await_count, 2)


if __name__ == "__main__":
    unittest.main()
