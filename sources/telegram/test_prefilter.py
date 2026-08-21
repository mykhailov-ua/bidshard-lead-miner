import unittest

from sources.telegram.prefilter import (
    channel_icp_relevant,
    is_spam_message,
    should_emit_message,
)


class PrefilterTest(unittest.TestCase):
    def test_spam_rejected(self) -> None:
        self.assertTrue(is_spam_message("join our channel for vip signals"))

    def test_pain_emitted(self) -> None:
        self.assertTrue(
            should_emit_message("voluum alternative postback failing badly")
        )

    def test_channel_icp_relevant(self) -> None:
        self.assertTrue(
            channel_icp_relevant(["Igaming affiliate media buying tracker"])
        )

    def test_channel_spam_not_relevant(self) -> None:
        self.assertFalse(
            channel_icp_relevant(["VIP signal course mentorship paid tips only"])
        )


if __name__ == "__main__":
    unittest.main()
