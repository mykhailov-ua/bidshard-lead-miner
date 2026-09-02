import unittest

from sources.telegram.pain import message_has_pain


class PainTest(unittest.TestCase):
    def test_message_has_pain(self) -> None:
        self.assertTrue(message_has_pain("voluum postback failing"))
        self.assertFalse(message_has_pain("good morning team"))


if __name__ == "__main__":
    unittest.main()
