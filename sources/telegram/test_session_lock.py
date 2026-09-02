import os
import unittest
from pathlib import Path
from unittest import mock

from sources.telegram.session_lock import parent_holds_session_lock, session_exclusive_lock


class SessionLockTest(unittest.TestCase):
    def test_parent_holds_skips_flock(self) -> None:
        with mock.patch.dict(os.environ, {"TELETHON_PARENT_HOLDS_LOCK": "1"}):
            self.assertTrue(parent_holds_session_lock())
            with mock.patch("fcntl.flock") as flock:
                with session_exclusive_lock(Path("/tmp/telethon.session")):
                    pass
                flock.assert_not_called()

    def test_direct_cli_acquires_flock(self) -> None:
        with mock.patch.dict(os.environ, {}, clear=True):
            self.assertFalse(parent_holds_session_lock())


if __name__ == "__main__":
    unittest.main()
