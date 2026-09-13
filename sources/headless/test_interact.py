import unittest
from unittest.mock import MagicMock, patch

from sources.headless.interact import simulate_reading


class InteractTest(unittest.TestCase):
    @patch("sources.headless.interact._rng")
    def test_simulate_reading_wheels_and_moves(self, rng_mock: MagicMock) -> None:
        r = MagicMock()
        r.randint.side_effect = lambda a, b: a
        r.uniform.return_value = 0.0
        rng_mock.return_value = r

        page = MagicMock()
        page.evaluate.return_value = 2000
        page.wait_for_timeout = MagicMock()
        page.mouse.wheel = MagicMock()
        page.mouse.move = MagicMock()
        page.wait_for_load_state = MagicMock()

        simulate_reading(page, 1920, 1080)
        self.assertTrue(page.mouse.wheel.called)
        self.assertTrue(page.mouse.move.called)


if __name__ == "__main__":
    unittest.main()
