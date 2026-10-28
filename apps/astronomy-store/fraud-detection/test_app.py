import unittest
from unittest.mock import MagicMock, patch

from app import check_order


class CheckOrderTest(unittest.TestCase):
    def test_returns_fraud_score_logs_it_and_enriches_active_span(self):
        span = MagicMock()
        order = {"order_id": "abc-123", "shipping_tracking_id": "track-456"}

        with (
            patch("app.random.randrange", return_value=42),
            patch("app.trace.get_current_span", return_value=span),
            self.assertLogs("app", level="INFO") as logs,
        ):
            fraud_score = check_order(order)

        self.assertEqual(fraud_score, 42)
        self.assertEqual(logs.records[0].msg["order_id"], "abc-123")
        self.assertEqual(logs.records[0].msg["fraud_score"], 42)
        span.set_attribute.assert_called_once_with("astronomystore.fraud_score", 42)


if __name__ == "__main__":
    unittest.main()
